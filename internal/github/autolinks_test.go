package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type autolinkFixture struct {
	client *Client
	rows   []map[string]any
	calls  []string
	writes int
	alpha  bool
	hook   func(*http.Request, *http.Response)
}

func autolinkRaw(id int, prefix string) map[string]any {
	return map[string]any{"id": id, "key_prefix": prefix, "url_template": "https://target.test/<num>?Case=A%2FB", "is_alphanumeric": false}
}
func autolinkRepo() repository.Repository {
	return repository.Repository{Owner: "owner", Name: "current"}
}
func newAutolinkFixture(t *testing.T) *autolinkFixture {
	f := &autolinkFixture{rows: []map[string]any{autolinkRaw(7, "OLD-")}}
	f.client = topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
		f.calls = append(f.calls, req.Method+" "+req.URL.Path)
		status := 200
		var body any
		path := strings.TrimPrefix(req.URL.Path, "/repos/owner/current")
		switch {
		case req.Method == "GET" && path == "":
			body = map[string]any{"id": 1, "full_name": "owner/current", "private": false, "archived": false, "owner": map[string]any{"id": 2, "login": "owner", "type": "User"}}
		case req.Method == "GET" && path == "/autolinks":
			body = f.rows
		case req.Method == "POST" && path == "/autolinks":
			var input map[string]any
			if json.NewDecoder(req.Body).Decode(&input) != nil {
				t.Fatal("invalid POST")
			}
			if !reflect.DeepEqual(input, map[string]any{"key_prefix": "NEW-", "url_template": "https://target.test/<num>?Case=A%2FB", "is_alphanumeric": f.alpha}) {
				t.Fatal(input)
			}
			f.writes++
			row := autolinkRaw(8, "NEW-")
			row["is_alphanumeric"] = input["is_alphanumeric"]
			f.rows = append(f.rows, row)
			body = row
			status = 201
		case strings.HasPrefix(path, "/autolinks/"):
			id, err := strconv.Atoi(strings.TrimPrefix(path, "/autolinks/"))
			if err != nil {
				t.Fatal(err)
			}
			found := -1
			for i, r := range f.rows {
				if r["id"] == id {
					found = i
				}
			}
			if found < 0 {
				status = 404
				body = map[string]any{"message": "PRIVATE_URL_CANARY"}
			} else if req.Method == "GET" {
				body = f.rows[found]
			} else if req.Method == "DELETE" {
				f.writes++
				status = 204
				f.rows = append(f.rows[:found], f.rows[found+1:]...)
			} else {
				t.Fatal(req.Method, path)
			}
		default:
			t.Fatal(req.Method, path)
		}
		data, _ := json.Marshal(body)
		if status == 204 {
			data = nil
		}
		response := labelsResponse(req, status, string(data))
		if f.hook != nil {
			f.hook(req, response)
		}
		return response, nil
	}))
	return f
}
func autolinkDesired() settings.Autolink {
	return settings.Autolink{ReferencePrefix: "NEW-", TargetURL: "https://target.test/<num>?Case=A%2FB", ReferenceFormat: "numeric"}
}
func TestAutolinkCreateACKReadbackAndRecovery(t *testing.T) {
	for _, mode := range []string{"success", "alphanumeric", "status200", "error-with-id", "missing-field", "known-id", "missing-id", "readback-denied", "readback-drift", "extra-inventory", "retained-drift", "metadata-drift"} {
		t.Run(mode, func(t *testing.T) {
			f := newAutolinkFixture(t)
			f.alpha = mode == "alphanumeric"
			before, err := f.client.ReadAutolinks(autolinkRepo())
			if err != nil {
				t.Fatal(err)
			}
			f.hook = func(req *http.Request, response *http.Response) {
				replace := func(status int, body string) { *response = *labelsResponse(req, status, body) }
				if req.Method == "POST" {
					switch mode {
					case "status200":
						response.StatusCode = 200
					case "error-with-id":
						replace(422, `{"id":8,"message":"PRIVATE_URL_CANARY"}`)
					case "missing-field":
						replace(201, `{"id":8}`)
					case "known-id":
						replace(201, `{"id":7}`)
					case "missing-id":
						replace(201, `{}`)
					}
				}
				if req.Method == "GET" && req.URL.Path == "/repos/owner/current/autolinks/8" {
					if mode == "readback-denied" {
						replace(403, `{"message":"PRIVATE_URL_CANARY"}`)
					}
					if mode == "readback-drift" {
						replace(200, `{"id":8,"key_prefix":"WRONG","url_template":"https://x/<num>","is_alphanumeric":false}`)
					}
				}
				if req.Method == "GET" && req.URL.Path == "/repos/owner/current/autolinks" && f.writes > 0 {
					var rows []map[string]any
					for _, row := range f.rows {
						copy := map[string]any{}
						for k, v := range row {
							copy[k] = v
						}
						rows = append(rows, copy)
					}
					switch mode {
					case "extra-inventory":
						rows = append(rows, autolinkRaw(9, "OTHER-"))
					case "retained-drift":
						rows[0]["url_template"] = "https://different/<num>"
					case "metadata-drift":
						rows[0]["updated_at"] = nil
					}
					data, _ := json.Marshal(rows)
					replace(200, string(data))
				}
			}
			desired := autolinkDesired()
			if f.alpha {
				desired.ReferenceFormat = "alphanumeric"
			}
			result, err := f.client.CreateAutolink(autolinkRepo(), before, desired)
			if (err == nil) != (mode == "success" || mode == "alphanumeric") || !result.Attempted || f.writes != 1 {
				t.Fatal(result, err, f.calls)
			}
			recovery := mode != "status200" && mode != "error-with-id" && mode != "known-id" && mode != "missing-id"
			if (result.RecoveryID == 8) != recovery {
				t.Fatal("recovery boundary", result, err)
			}
			if (result.Verified != nil) != (mode == "success" || mode == "alphanumeric") {
				t.Fatal(result, err)
			}
			if err != nil && strings.Contains(err.Error(), "PRIVATE") {
				t.Fatal("upstream detail leaked", err)
			}
			if (mode == "success" || mode == "alphanumeric") && len(f.calls) != 8 {
				t.Fatal("expected initial/prewrite inventory plus independent object and inventory", f.calls)
			}
		})
	}
}
func TestAutolinkDeleteFixedInventoryAndAbsence(t *testing.T) {
	for _, mode := range []string{"success", "candidate-drift", "addition", "wrong-object", "error", "nonempty-ack", "retained-after-delete", "readback-denied"} {
		t.Run(mode, func(t *testing.T) {
			f := newAutolinkFixture(t)
			before, err := f.client.ReadAutolinks(autolinkRepo())
			if err != nil {
				t.Fatal(err)
			}
			if mode == "candidate-drift" {
				f.rows[0]["url_template"] = "https://changed/<num>"
			}
			if mode == "addition" {
				f.rows = append(f.rows, autolinkRaw(9, "EXTRA-"))
			}
			f.hook = func(req *http.Request, response *http.Response) {
				if req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/7") && mode == "wrong-object" {
					*response = *labelsResponse(req, 200, `{"id":9,"key_prefix":"OLD-","url_template":"https://target.test/<num>?Case=A%2FB","is_alphanumeric":false}`)
				}
				if req.Method == "DELETE" {
					if mode == "error" {
						*response = *labelsResponse(req, 403, `{"message":"PRIVATE_URL_CANARY"}`)
					}
					if mode == "nonempty-ack" {
						*response = *labelsResponse(req, 204, " ")
					}
					if mode == "retained-after-delete" {
						f.rows = append(f.rows, autolinkRaw(7, "OLD-"))
					}
				}
				if req.Method == "GET" && f.writes > 0 && mode == "readback-denied" {
					*response = *labelsResponse(req, 403, `{}`)
				}
			}
			attempted, err := f.client.DeleteAutolink(autolinkRepo(), before, 7)
			if (err == nil) != (mode == "success") {
				t.Fatal(attempted, err)
			}
			pre := mode == "candidate-drift" || mode == "addition" || mode == "wrong-object"
			if attempted == pre || (f.writes == 0) != pre {
				t.Fatal(attempted, err, f.calls)
			}
		})
	}
}
func TestAutolinkReadRejectsUnknownIncompleteAndNonTarget(t *testing.T) {
	for _, mode := range []string{"null-list", "missing-format", "duplicate-id", "overlap", "unknown", "null-key", "org", "private", "foreign", "link", "missing-owner"} {
		t.Run(mode, func(t *testing.T) {
			f := newAutolinkFixture(t)
			f.hook = func(req *http.Request, response *http.Response) {
				if strings.HasSuffix(req.URL.Path, "/autolinks") {
					switch mode {
					case "null-list":
						*response = *labelsResponse(req, 200, `null`)
					case "link":
						response.Header.Set("Link", `<https://api.github.com/else>; rel="next"`)
					default:
						row := autolinkRaw(7, "OLD-")
						rows := []map[string]any{row}
						switch mode {
						case "missing-format":
							delete(row, "is_alphanumeric")
						case "null-key":
							row["key_prefix"] = nil
						case "unknown":
							row["future_policy"] = true
						case "duplicate-id":
							rows = append(rows, autolinkRaw(7, "NEW-"))
						case "overlap":
							rows = append(rows, autolinkRaw(8, "OLD"))
						}
						data, _ := json.Marshal(rows)
						*response = *labelsResponse(req, 200, string(data))
					}
				} else {
					kind := "User"
					private := false
					name := "owner/current"
					owner := `{"id":2,"login":"owner","type":"User"}`
					switch mode {
					case "org":
						kind = "Organization"
					case "private":
						private = true
					case "foreign":
						name = "else/current"
					case "missing-owner":
						owner = "null"
					}
					if mode != "missing-owner" {
						owner = fmt.Sprintf(`{"id":2,"login":"owner","type":%q}`, kind)
					}
					*response = *labelsResponse(req, 200, fmt.Sprintf(`{"id":1,"full_name":%q,"private":%t,"archived":false,"owner":%s}`, name, private, owner))
				}
			}
			if _, err := f.client.ReadAutolinks(autolinkRepo()); err == nil {
				t.Fatal("incomplete accepted")
			}
		})
	}
}
