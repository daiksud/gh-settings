package github

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

var deployRepo = repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}

const deployTargetJSON = `{"id":1,"full_name":"owner/current","archived":false,"visibility":"public","owner":{"id":2,"login":"owner","type":"User"}}`
const deployPublicOne = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB"
const deployPublicTwo = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIC"

func deployKeyJSON(id int64, key string) map[string]any {
	return map[string]any{"id": id, "title": "same title", "key": key, "read_only": true, "verified": false, "created_at": "2026-11-14T00:00:00Z", "url": fmt.Sprintf("https://api.github.com/repos/owner/current/keys/%d", id)}
}
func deployJSON(value any) string { data, _ := json.Marshal(value); return string(data) }
func deployObservation(t *testing.T, rows ...map[string]any) settings.DeployKeyObservation {
	t.Helper()
	values := []settings.DeployKeyState{}
	for _, row := range rows {
		var raw map[string]json.RawMessage
		json.Unmarshal([]byte(deployJSON(row)), &raw)
		v, e := decodeDeployKey(deployRepo, raw)
		if e != nil {
			t.Fatal(e)
		}
		values = append(values, v)
	}
	result, e := settings.NewDeployKeyObservation(settings.DeployKeyTarget{ID: 1, OwnerID: 2, FullName: "owner/current"}, values)
	if e != nil {
		t.Fatal(e)
	}
	return result
}
func TestDeployKeyCompleteAcquisition(t *testing.T) {
	for _, mode := range []string{"empty", "complete", "unknown enabled", "false enabled", "missing key", "null readonly", "null enabled", "wrong enabled", "bad timestamp", "wrong attribution", "unknown field", "foreign URL", "redirect URL", "target", "null list", "trailing", "missing verified"} {
		t.Run(mode, func(t *testing.T) {
			row := deployKeyJSON(3, deployPublicOne)
			row["added_by"] = nil
			row["last_used"] = nil
			target := deployTargetJSON
			body := ""
			switch mode {
			case "empty":
				body = "[]"
			case "complete":
				row["enabled"] = true
				row["last_used"] = "2026-11-14T01:00:00Z"
			case "false enabled":
				row["enabled"] = false
			case "missing key":
				delete(row, "key")
			case "null readonly":
				row["read_only"] = nil
			case "null enabled":
				row["enabled"] = nil
			case "wrong enabled":
				row["enabled"] = "true"
			case "bad timestamp":
				row["last_used"] = "yesterday"
			case "wrong attribution":
				row["added_by"] = 12
			case "unknown field":
				row["unowned_grant"] = true
			case "foreign URL":
				row["url"] = "https://api.github.com/repos/other/current/keys/3"
			case "redirect URL":
				row["url"] = "https://api.github.com/repos/owner/current/keys/3?other=true"
			case "target":
				target = strings.Replace(target, `"User"`, `"Organization"`, 1)
			case "null list":
				body = "null"
			case "trailing":
				body = "[] {}"
			case "missing verified":
				delete(row, "verified")
			}
			if body == "" {
				body = deployJSON([]any{row})
			}
			c := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				b := body
				if req.URL.Path == "/repos/owner/current" {
					b = target
				}
				return collaboratorsHTTPResponse(req, 200, b), nil
			})
			_, err := c.ReadDeployKeys(deployRepo)
			valid := mode == "empty" || mode == "complete" || mode == "unknown enabled" || mode == "false enabled"
			if (err == nil) != valid {
				t.Fatal(mode, err)
			}
		})
	}
}
func TestDeployKeyPagination(t *testing.T) {
	for _, mode := range []string{"complete", "later denied", "repeat id", "repeat material", "short next"} {
		t.Run(mode, func(t *testing.T) {
			pages := 0
			c := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == "/repos/owner/current" {
					return collaboratorsHTTPResponse(req, 200, deployTargetJSON), nil
				}
				pages++
				values := []any{}
				status := 200
				if pages == 1 {
					for i := 0; i < 100; i++ {
						raw, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(deployPublicOne, "ssh-ed25519 "))
						raw[len(raw)-1] = byte(i)
						key := "ssh-ed25519 " + base64.StdEncoding.EncodeToString(raw)
						values = append(values, deployKeyJSON(int64(i+1), key))
					}
				}
				if pages == 2 {
					switch mode {
					case "later denied":
						status = 403
					case "repeat id":
						values = append(values, deployKeyJSON(1, deployPublicTwo))
					case "repeat material":
						values = append(values, deployKeyJSON(101, deployPublicOne))
					}
				}
				if mode == "short next" {
					values = []any{}
				}
				response := collaboratorsHTTPResponse(req, status, deployJSON(values))
				if mode == "short next" {
					response.Header.Set("Link", `<https://untrusted.test>; rel="next"`)
				}
				return response, nil
			})
			_, err := c.ReadDeployKeys(deployRepo)
			if (err == nil) != (mode == "complete") || mode == "complete" && pages != 2 {
				t.Fatal(pages, err)
			}
		})
	}
}
func TestDeployKeyCreationVerificationAndRecovery(t *testing.T) {
	for _, mode := range []string{"complete", "enabled enrichment", "dynamic metadata", "preflight drift", "bad acknowledgement", "wrong status", "reused id", "detail denied", "detail identity", "enabled contradiction", "inventory addition", "retained change"} {
		t.Run(mode, func(t *testing.T) {
			old := deployKeyJSON(3, deployPublicOne)
			created := deployKeyJSON(4, deployPublicTwo)
			created["read_only"] = false
			expected := deployObservation(t, old)
			key, _ := settings.ParseDeployPublicKey(deployPublicTwo + " comment")
			desired := settings.DeployKeyDeclaration{Title: "same title", Key: key, AllowWriteAccess: true}
			posted := false
			detail := 0
			c := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				status := 200
				var body string
				switch req.Method + " " + req.URL.Path {
				case "GET /repos/owner/current":
					body = deployTargetJSON
				case "GET /repos/owner/current/keys":
					rows := []any{old}
					if posted {
						rows = append(rows, created)
					}
					if mode == "preflight drift" && !posted {
						rows = []any{}
					}
					if mode == "inventory addition" && posted {
						rows = []any{created}
					}
					if mode == "retained change" && posted {
						old["enabled"] = false
					}
					body = deployJSON(rows)
				case "POST /repos/owner/current/keys":
					posted = true
					var input map[string]any
					json.NewDecoder(req.Body).Decode(&input)
					if len(input) != 3 || input["key"] != deployPublicTwo || input["read_only"] != false || input["title"] != "same title" {
						t.Fatal("incorrect payload")
					}
					status = 201
					ack := deployKeyJSON(4, deployPublicTwo)
					ack["read_only"] = false
					switch mode {
					case "bad acknowledgement":
						delete(ack, "key")
					case "wrong status":
						status = 422
					case "reused id":
						ack["id"] = 3
					case "enabled contradiction":
						ack["enabled"] = false
					}
					body = deployJSON(ack)
				case "GET /repos/owner/current/keys/4":
					detail++
					if mode == "detail denied" {
						status = 403
					}
					if mode == "detail identity" {
						created["id"] = 8
					}
					if mode == "enabled enrichment" || mode == "enabled contradiction" {
						created["enabled"] = true
					}
					if mode == "dynamic metadata" {
						created["verified"] = true
						created["added_by"] = "private-attribution"
						created["last_used"] = "2026-11-14T01:00:00Z"
					}
					body = deployJSON(created)
				default:
					t.Fatal(req.Method, req.URL)
				}
				return collaboratorsHTTPResponse(req, status, body), nil
			})
			result, err := c.CreateDeployKey(deployRepo, expected, desired)
			success := mode == "complete" || mode == "enabled enrichment" || mode == "dynamic metadata"
			if (err == nil) != success {
				t.Fatal(mode, err)
			}
			recover := mode != "preflight drift" && mode != "wrong status" && mode != "reused id"
			if (result.ID > 0) != recover {
				t.Fatal("recovery evidence", result.ID, err)
			}
			var pre interface{ ReadFailedBeforeWrite() }
			if mode == "preflight drift" && (!errors.As(err, &pre) || posted) {
				t.Fatal("pre-write certainty", err)
			}
			if success && (!result.Observation.HasID(4) || detail != 1) {
				t.Fatal("missing independent readback")
			}
			if err != nil && strings.Contains(err.Error(), "private-attribution") {
				t.Fatal("private error")
			}
		})
	}
}
func TestDeployKeyDeleteFrozenStateAndAbsence(t *testing.T) {
	for _, mode := range []string{"complete", "dynamic metadata", "stable unknown", "enabled presence", "enabled value", "target drift", "candidate detail", "external member", "status404", "not absent", "postread denied"} {
		t.Run(mode, func(t *testing.T) {
			row := deployKeyJSON(3, deployPublicOne)
			if mode != "stable unknown" {
				row["enabled"] = false
			}
			expected := deployObservation(t, row)
			deleted := false
			details := 0
			c := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				status := 200
				body := ""
				switch req.Method + " " + req.URL.Path {
				case "GET /repos/owner/current":
					body = deployTargetJSON
					if mode == "target drift" {
						body = strings.Replace(body, `"id":1`, `"id":8`, 1)
					}
				case "GET /repos/owner/current/keys":
					rows := []any{row}
					if deleted && mode != "not absent" {
						rows = []any{}
					}
					switch mode {
					case "enabled presence":
						delete(row, "enabled")
					case "enabled value":
						row["enabled"] = true
					case "external member":
						rows = append(rows, deployKeyJSON(4, deployPublicTwo))
					case "dynamic metadata":
						row["verified"] = true
						row["added_by"] = "updated"
						row["last_used"] = nil
					}
					if deleted && mode == "postread denied" {
						status = 403
					}
					body = deployJSON(rows)
				case "GET /repos/owner/current/keys/3":
					details++
					if mode == "candidate detail" {
						row["title"] = "changed"
					}
					body = deployJSON(row)
				case "DELETE /repos/owner/current/keys/3":
					deleted = true
					status = 204
					if mode == "status404" {
						status = 404
					}
				default:
					t.Fatal(req.Method, req.URL)
				}
				return collaboratorsHTTPResponse(req, status, body), nil
			})
			after, err := c.DeleteDeployKey(deployRepo, expected, 3)
			success := mode == "complete" || mode == "dynamic metadata" || mode == "stable unknown"
			if (err == nil) != success {
				t.Fatal(err)
			}
			if success && (after.HasID(3) || !deleted || details != 1) {
				t.Fatal("unverified absence")
			}
			prewrite := mode == "enabled presence" || mode == "enabled value" || mode == "target drift" || mode == "candidate detail" || mode == "external member"
			var pre interface{ ReadFailedBeforeWrite() }
			if prewrite && (!errors.As(err, &pre) || deleted) {
				t.Fatal("unsafe prewrite", err)
			}
			if !success && !prewrite && (!deleted || errors.As(err, &pre)) {
				t.Fatal("false postwrite certainty", err)
			}
		})
	}
}
