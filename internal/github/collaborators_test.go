package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"io"
	"net/http"
	"strings"
	"testing"
)

const collaboratorTargetJSON = `{"id":1,"full_name":"owner/current","private":false,"archived":false,"owner":{"id":2,"login":"owner","type":"User"}}`
const collaboratorActiveJSON = `{"id":3,"login":"person","type":"User","role_name":"write","permissions":{"pull":true,"push":true,"admin":false,"triage":true,"maintain":false}}`
const collaboratorInvitationJSON = `{"id":7,"repository":{"id":1,"full_name":"owner/current"},"invitee":{"id":3,"login":"person","type":"User"},"permissions":"write","created_at":"2026-11-13T00:00:00Z"}`

func collaboratorsHTTPResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{Request: req, StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func TestCollaboratorAcquisitionRejectsIncompleteState(t *testing.T) {
	for _, test := range []struct {
		name, path, body string
		valid            bool
	}{
		{"empty", "collaborators", "[]", true}, {"active", "collaborators", "[" + collaboratorActiveJSON + "]", true}, {"pending", "invitations", "[" + collaboratorInvitationJSON + "]", true},
		{"owner excluded", "collaborators", `[{"id":2,"login":"owner","type":"User","role_name":"admin"}]`, true},
		{"null list", "collaborators", "null", false}, {"null item", "collaborators", "[null]", false}, {"missing role", "collaborators", `[{"id":3,"login":"person","type":"User"}]`, false},
		{"unsupported role", "collaborators", "[" + strings.Replace(collaboratorActiveJSON, `"write"`, `"admin"`, 1) + "]", false},
		{"unsupported actor", "collaborators", "[" + strings.Replace(collaboratorActiveJSON, `"User"`, `"Bot"`, 1) + "]", false},
		{"contradictory permission", "collaborators", "[" + strings.Replace(collaboratorActiveJSON, `"push":true`, `"push":false`, 1) + "]", false},
		{"null invitee", "invitations", "[" + strings.Replace(collaboratorInvitationJSON, `{"id":3,"login":"person","type":"User"}`, `null`, 1) + "]", false},
		{"expired", "invitations", "[" + strings.TrimSuffix(collaboratorInvitationJSON, "}") + `,"expired":true}]`, false},
		{"null expiration", "invitations", "[" + strings.TrimSuffix(collaboratorInvitationJSON, "}") + `,"expired":null}]`, false},
		{"wrong repository", "invitations", "[" + strings.Replace(collaboratorInvitationJSON, `"id":1`, `"id":9`, 1) + "]", false},
		{"private", "current", strings.Replace(collaboratorTargetJSON, `"private":false`, `"private":true`, 1), false},
		{"organization", "current", strings.Replace(collaboratorTargetJSON, `"User"`, `"Organization"`, 1), false},
		{"missing target id", "current", strings.Replace(collaboratorTargetJSON, `"id":1,`, "", 1), false},
		{"trailing data", "invitations", "[] {}", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				if req.Method != "GET" {
					t.Fatal("mutation")
				}
				body := "[]"
				if req.URL.Path == "/repos/owner/current" {
					body = collaboratorTargetJSON
				}
				if strings.HasSuffix(req.URL.Path, "/"+test.path) {
					body = test.body
				}
				return collaboratorsHTTPResponse(req, 200, body), nil
			})
			observed, err := client.ReadCollaborators(repository.Repository{Owner: "owner", Name: "current"})
			if (err == nil) != test.valid {
				t.Fatal(observed, err)
			}
		})
	}
}

func TestCollaboratorPaginationAndIdentity(t *testing.T) {
	for _, failure := range []string{"", "later active page", "later invitation page", "overlap", "duplicate", "target changed", "short page next", "redirect"} {
		t.Run(failure, func(t *testing.T) {
			calls := map[string]int{}
			client := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				key := req.URL.Path
				calls[key]++
				body := collaboratorTargetJSON
				status := 200
				switch {
				case strings.HasSuffix(key, "/collaborators"):
					items := []any{}
					if req.URL.Query().Get("page") == "1" {
						for i := 0; i < 100; i++ {
							items = append(items, map[string]any{"id": i + 100, "login": fmt.Sprintf("user%d", i), "type": "User", "role_name": "write"})
						}
					} else if failure == "later active page" {
						status = 403
					}
					if failure == "overlap" {
						items = []any{json.RawMessage(collaboratorActiveJSON)}
					}
					if failure == "duplicate" && req.URL.Query().Get("page") == "2" {
						items = []any{map[string]any{"id": 100, "login": "user0", "type": "User", "role_name": "write"}}
					}
					data, _ := json.Marshal(items)
					body = string(data)
				case strings.HasSuffix(key, "/invitations"):
					items := []any{}
					if req.URL.Query().Get("page") == "1" {
						for i := 0; i < 100; i++ {
							var item map[string]any
							json.Unmarshal([]byte(collaboratorInvitationJSON), &item)
							item["id"] = i + 200
							item["invitee"] = map[string]any{"id": i + 300, "login": fmt.Sprintf("pending%d", i), "type": "User"}
							items = append(items, item)
						}
					} else if failure == "later invitation page" {
						status = 403
					}
					if failure == "overlap" {
						items = []any{json.RawMessage(collaboratorInvitationJSON)}
					}
					data, _ := json.Marshal(items)
					body = string(data)
				default:
					if failure == "target changed" && calls[key] > 1 {
						body = strings.Replace(body, `"id":1`, `"id":4`, 1)
					}
				}
				response := collaboratorsHTTPResponse(req, status, body)
				if failure == "short page next" && strings.HasSuffix(key, "/invitations") {
					response.Body = io.NopCloser(strings.NewReader("[]"))
					response.Header.Set("Link", `<https://other.test>; rel="next"`)
				}
				if failure == "redirect" {
					response.StatusCode = 301
					response.Header.Set("Location", "https://other.test")
				}
				return response, nil
			})
			observed, err := client.ReadCollaborators(repository.Repository{Owner: "owner", Name: "current"})
			if failure != "" {
				if err == nil {
					t.Fatal("invalid snapshot accepted")
				}
				return
			}
			rows, err := observed.Values()
			if err != nil || len(rows) != 200 || calls["/repos/owner/current/collaborators"] != 2 || calls["/repos/owner/current/invitations"] != 2 {
				t.Fatal(len(rows), calls, err)
			}
		})
	}
}

func TestCollaboratorWriterLifecycle(t *testing.T) {
	for _, mode := range []string{"201 pending", "201 accepted", "204 active", "active match", "pending match", "appeared pending", "vanished pending", "renamed", "reassigned", "target changed", "archived", "network", "201 mismatch", "201 missing id", "201 verify denied", "201 verify missing", "201 wrong invitation", "204 pending", "204 body"} {
		t.Run(mode, func(t *testing.T) {
			writes, readsAfter := 0, 0
			person := settings.CollaboratorIdentity{ID: 3, Username: "person"}
			selected := settings.CollaboratorSelection{Person: person, DeclaredUsername: "person", Target: settings.CollaboratorTarget{ID: 1, OwnerID: 2, FullName: "owner/current"}, Status: "absent", Bind: true}
			if mode == "active match" {
				selected.Status = "active"
				selected.Bind = false
			}
			if mode == "pending match" || mode == "vanished pending" {
				selected.Status = "pending"
				selected.InvitationID = 7
				selected.Bind = false
			}
			if mode == "archived" {
				selected.Target.Archived = true
			}
			client := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				if req.Method == "PUT" {
					writes++
					var body []byte
					if req.Body != nil {
						body, _ = io.ReadAll(req.Body)
					}
					if len(body) != 0 || req.ContentLength != 0 || req.URL.Path != "/repos/owner/current/collaborators/person" {
						t.Fatal("not bodyless intended actor", req.URL, string(body))
					}
					if mode == "network" {
						return nil, errors.New("secret-response")
					}
					status, ack := 201, collaboratorInvitationJSON
					if strings.HasPrefix(mode, "204") {
						status, ack = 204, ""
					}
					if mode == "204 body" {
						ack = `{}`
					}
					if mode == "201 mismatch" {
						ack = strings.Replace(ack, `"id":3`, `"id":8`, 1)
					}
					if mode == "201 missing id" {
						ack = strings.Replace(ack, `"id":7,`, "", 1)
					}
					return collaboratorsHTTPResponse(req, status, ack), nil
				}
				if writes > 0 {
					readsAfter++
				}
				body := "[]"
				status := 200
				switch req.URL.Path {
				case "/repos/owner/current":
					body = collaboratorTargetJSON
					if mode == "target changed" {
						body = strings.Replace(body, `"id":1`, `"id":4`, 1)
					}
					if mode == "archived" {
						body = strings.Replace(body, `"archived":false`, `"archived":true`, 1)
					}
				case "/users/person":
					body = `{"id":3,"login":"person","type":"User"}`
					if mode == "renamed" {
						body = strings.Replace(body, "person", "renamed", 1)
					}
					if mode == "reassigned" {
						body = strings.Replace(body, `"id":3`, `"id":8`, 1)
					}
				case "/repos/owner/current/collaborators":
					if mode == "active match" || writes > 0 && (mode == "201 accepted" || mode == "204 active") {
						body = "[" + collaboratorActiveJSON + "]"
					}
				case "/repos/owner/current/invitations":
					if mode == "pending match" || mode == "appeared pending" || writes > 0 && (mode == "201 pending" || mode == "204 pending" || mode == "201 wrong invitation") {
						body = "[" + collaboratorInvitationJSON + "]"
						if mode == "201 wrong invitation" {
							body = strings.Replace(body, `"id":7`, `"id":9`, 1)
						}
					}
					if writes > 0 && mode == "201 verify denied" {
						status = 403
					}
				default:
					t.Fatal("unexpected request", req.URL)
				}
				return collaboratorsHTTPResponse(req, status, body), nil
			})
			result, err := client.ReconcileCollaborator(repository.Repository{Owner: "owner", Name: "current"}, selected)
			valid := mode == "201 pending" || mode == "201 accepted" || mode == "204 active" || mode == "active match" || mode == "pending match" || mode == "appeared pending"
			if (err == nil) != valid || writes > 1 || result.Attempted != (writes > 0) {
				t.Fatal(result, err, writes)
			}
			var before interface{ ReadFailedBeforeWrite() }
			if err != nil && errors.As(err, &before) != (writes == 0) {
				t.Fatal("incorrect certainty", result, err)
			}
			if valid && writes > 0 && readsAfter == 0 {
				t.Fatal("no independent verification")
			}
			if mode == "201 mismatch" && result.InvitationID != 7 {
				t.Fatal("lost acknowledgment recovery", result)
			}
			if err != nil && strings.Contains(err.Error(), "secret-response") {
				t.Fatal("leaked upstream error")
			}
		})
	}
}
