package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"net/http"
	"strings"
	"testing"
)

type collaboratorPruneFixture struct {
	active, pending bool
	operator, mode  string
	inviter         string
	deleted         []string
	requests        []string
	afterDelete     bool
	archived        bool
	accepted        bool
}

func collaboratorPruneClient(t *testing.T, f *collaboratorPruneFixture) *Client {
	t.Helper()
	return topicsClient(t, func(req *http.Request) (*http.Response, error) {
		f.requests = append(f.requests, req.Method+" "+req.URL.RequestURI())
		path := req.URL.Path
		body := ""
		status := 200
		if req.Method == "DELETE" {
			if req.Body != nil || req.ContentLength != 0 {
				t.Fatal("DELETE body", req.ContentLength)
			}
			f.deleted = append(f.deleted, path)
			f.afterDelete = true
			if f.mode == "network" {
				return nil, errors.New("private-inviter-email@example.test")
			}
			if f.mode == "403" {
				return collaboratorsHTTPResponse(req, 403, `{"message":"private-inviter-email@example.test"}`), nil
			}
			if f.mode == "201" {
				return collaboratorsHTTPResponse(req, 201, ""), nil
			}
			if f.mode == "body" {
				return collaboratorsHTTPResponse(req, 204, "unexpected"), nil
			}
			switch path {
			case "/repos/owner/current/invitations/7":
				if f.mode != "unchanged" {
					f.pending = false
				}
			case "/repos/owner/current/collaborators/person":
				if f.mode != "unchanged" {
					f.active = false
				}
			default:
				t.Fatal("unexpected deletion", path)
			}
			return collaboratorsHTTPResponse(req, 204, ""), nil
		}
		if req.Method != "GET" {
			t.Fatal("unexpected mutation", req.Method)
		}
		switch path {
		case "/repos/owner/current":
			body = collaboratorTargetJSON
			if f.archived {
				body = strings.Replace(body, `"archived":false`, `"archived":true`, 1)
			}
			if f.afterDelete && f.mode == "target drift" {
				body = strings.Replace(body, `"id":1`, `"id":99`, 1)
			}
		case "/repos/owner/current/collaborators":
			body = "[]"
			if f.active {
				body = "[" + collaboratorActiveJSON + "]"
			}
			if f.accepted {
				body = "[" + collaboratorActiveJSON + `,{"id":4,"login":"pending","type":"User","role_name":"write"}]`
			}
		case "/repos/owner/current/invitations":
			body = "[]"
			if f.pending {
				invite := strings.ReplaceAll(collaboratorInvitationJSON, `"id":3,"login":"person"`, `"id":4,"login":"pending"`)
				if f.inviter != "" {
					invite = strings.TrimSuffix(invite, "}") + `,"inviter":` + f.inviter + `}`
				}
				body = "[" + invite + "]"
			}
			if f.afterDelete && f.mode == "readback denied" {
				status = 403
				body = `{"message":"private-inviter-email@example.test"}`
			}
		case "/user":
			body = `{"id":2,"login":"owner","type":"User"}`
			if f.operator == "installation" || f.operator == "unknown" {
				status = 403
			}
			if f.operator == "nonowner" {
				body = `{"id":3,"login":"person","type":"User"}`
			}
			if f.operator == "malformed" {
				body = `{"id":2}`
			}
		case "/installation/repositories":
			status = 403
			if f.operator == "installation" || f.operator == "ambiguous" {
				status = 200
				body = `{"total_count":1,"repositories":[` + collaboratorTargetJSON + `]}`
				if f.archived {
					body = strings.Replace(body, `"archived":false`, `"archived":true`, 1)
				}
			}
		case "/user/3", "/users/person":
			body = `{"id":3,"login":"person","type":"User"}`
			if f.mode == "reassigned" && path == "/users/person" {
				body = strings.Replace(body, `"id":3`, `"id":99`, 1)
			}
		case "/user/4", "/users/pending":
			body = `{"id":4,"login":"pending","type":"User"}`
		default:
			t.Fatal("unexpected request", req.Method, path)
		}
		return collaboratorsHTTPResponse(req, status, body), nil
	})
}
func TestCollaboratorPruneDeleteAndReadback(t *testing.T) {
	for _, principal := range []string{"owner", "installation"} {
		t.Run(principal, func(t *testing.T) {
			f := &collaboratorPruneFixture{active: true, pending: true, operator: principal, inviter: `{"id":3,"login":"person","type":"User"}`}
			client := collaboratorPruneClient(t, f)
			repo := repository.Repository{Owner: "owner", Name: "current"}
			empty, _ := settings.NewCollaboratorDeclarations(nil)
			prepared, err := client.ReadCollaboratorPrune(repo, empty)
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range prepared.Deletions() {
				var result settings.CollaboratorDeleteResult
				prepared, result, err = client.DeleteCollaborator(repo, prepared, d)
				if err != nil || !result.Attempted || !result.Acknowledged || !result.Verified {
					t.Fatal(result, err)
				}
			}
			if fmt.Sprint(f.deleted) != "[/repos/owner/current/invitations/7 /repos/owner/current/collaborators/person]" || f.active || f.pending {
				t.Fatal(f.deleted)
			}
			if len(prepared.Deletions()) != 0 {
				t.Fatal("completed candidates retained")
			}
			encoded, _ := json.Marshal(prepared)
			if string(encoded) != "{}" {
				t.Fatal("private snapshot leaked", string(encoded))
			}
		})
	}
}
func TestCollaboratorPruneOperatorAndRetainedInviter(t *testing.T) {
	for _, mode := range []string{"nonowner", "ambiguous", "unknown", "malformed", "retained dependency", "unknown inviter", "bot inviter"} {
		t.Run(mode, func(t *testing.T) {
			f := &collaboratorPruneFixture{active: true, pending: true, operator: mode, inviter: `{"id":2,"login":"owner","type":"User"}`}
			var people []settings.CollaboratorDeclaration
			if strings.Contains(mode, "inviter") || mode == "retained dependency" {
				f.operator = "owner"
				people = []settings.CollaboratorDeclaration{{Username: "pending"}}
			}
			if mode == "retained dependency" {
				f.inviter = `{"id":3,"login":"person","type":"User"}`
			}
			if mode == "unknown inviter" {
				f.inviter = "null"
			}
			if mode == "bot inviter" {
				f.inviter = `{"id":99,"login":"app[bot]","type":"Bot"}`
			}
			client := collaboratorPruneClient(t, f)
			d, _ := settings.NewCollaboratorDeclarations(people)
			_, err := client.ReadCollaboratorPrune(repository.Repository{Owner: "owner", Name: "current"}, d)
			if (err == nil) != (mode == "bot inviter") || len(f.deleted) != 0 {
				t.Fatal(err, f.deleted)
			}
		})
	}
}
func TestCollaboratorPruneFailureEvidence(t *testing.T) {
	for _, mode := range []string{"reassigned", "network", "403", "201", "body", "readback denied", "unchanged", "target drift"} {
		t.Run(mode, func(t *testing.T) {
			f := &collaboratorPruneFixture{active: true, operator: "owner"}
			client := collaboratorPruneClient(t, f)
			repo := repository.Repository{Owner: "owner", Name: "current"}
			empty, _ := settings.NewCollaboratorDeclarations(nil)
			prepared, err := client.ReadCollaboratorPrune(repo, empty)
			if err != nil {
				t.Fatal(err)
			}
			f.mode = mode
			_, result, err := client.DeleteCollaborator(repo, prepared, prepared.Deletions()[0])
			if err == nil || result.Verified || strings.Contains(err.Error(), "private-inviter") {
				t.Fatal(result, err)
			}
			var pre interface{ ReadFailedBeforeWrite() }
			if errors.As(err, &pre) != (mode == "reassigned") || result.Attempted == (mode == "reassigned") {
				t.Fatal("incorrect progress", result, err)
			}
			acknowledged := mode == "readback denied" || mode == "unchanged" || mode == "target drift"
			if result.Acknowledged != acknowledged {
				t.Fatal(result)
			}
		})
	}
}
func TestCollaboratorPruneDriftAndOwnUnarchive(t *testing.T) {
	for _, mode := range []string{"accepted", "vanished", "new invite", "archive", "own unarchive"} {
		t.Run(mode, func(t *testing.T) {
			f := &collaboratorPruneFixture{active: true, pending: true, operator: "owner"}
			if mode == "new invite" {
				f.pending = false
			}
			if mode == "own unarchive" {
				f.archived = true
			}
			client := collaboratorPruneClient(t, f)
			repo := repository.Repository{Owner: "owner", Name: "current"}
			empty, _ := settings.NewCollaboratorDeclarations(nil)
			prepared, err := client.ReadCollaboratorPrune(repo, empty)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "accepted":
				f.pending = false
				f.accepted = true
			case "vanished":
				f.pending = false
			case "new invite":
				f.pending = true
			case "archive":
				f.archived = true
			case "own unarchive":
				if prepared.ValidateApply(false) == nil {
					t.Fatal("archive guard missing")
				}
				prepared = prepared.AfterUnarchive()
				f.archived = false
			}
			_, result, err := client.DeleteCollaborator(repo, prepared, prepared.Deletions()[0])
			if (err == nil) != (mode == "own unarchive") {
				t.Fatal(result, err)
			}
			if err != nil && len(f.deleted) != 0 {
				t.Fatal(f.deleted)
			}
		})
	}
}

func TestCollaboratorPruneInstallationPaginationAndClosedIdentity(t *testing.T) {
	for _, mode := range []string{"complete", "missing count", "inconsistent count", "later denied", "duplicate", "missing target", "malformed target", "short next", "malformed user", "dual positive", "transport failure"} {
		t.Run(mode, func(t *testing.T) {
			pages := 0
			client := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == "/user" {
					if mode == "malformed user" {
						return collaboratorsHTTPResponse(req, 200, `{"id":2}`), nil
					}
					if mode == "dual positive" {
						return collaboratorsHTTPResponse(req, 200, `{"id":2,"login":"owner","type":"User"}`), nil
					}
					if mode == "transport failure" {
						return nil, errors.New("private upstream error")
					}
					return collaboratorsHTTPResponse(req, 403, `{}`), nil
				}
				if req.URL.Path != "/installation/repositories" || req.URL.Query().Get("per_page") != "100" {
					t.Fatal(req.URL)
				}
				pages++
				rows := []any{}
				if pages == 1 {
					for i := 0; i < 99; i++ {
						rows = append(rows, map[string]any{"id": i + 100, "full_name": fmt.Sprintf("owner/repo%d", i), "owner": map[string]any{"id": 2, "login": "owner", "type": "User"}})
					}
					var target map[string]any
					json.Unmarshal([]byte(collaboratorTargetJSON), &target)
					if mode == "missing target" {
						target["id"] = 999
						target["full_name"] = "owner/other"
					}
					if mode == "malformed target" {
						delete(target, "private")
					}
					rows = append(rows, target)
				}
				if pages == 2 && mode == "later denied" {
					return collaboratorsHTTPResponse(req, 403, `{}`), nil
				}
				if pages == 2 && mode == "duplicate" {
					var target any
					json.Unmarshal([]byte(collaboratorTargetJSON), &target)
					rows = append(rows, target)
				}
				if mode == "short next" {
					rows = rows[:1]
				}
				body := map[string]any{"total_count": 100, "repositories": rows}
				if mode == "missing count" {
					delete(body, "total_count")
				}
				if mode == "inconsistent count" && pages == 2 {
					body["total_count"] = 101
				}
				data, _ := json.Marshal(body)
				response := collaboratorsHTTPResponse(req, 200, string(data))
				if mode == "short next" {
					response.Header.Set("Link", `<https://api.github.com/installation/repositories?page=2>; rel="next"`)
				}
				return response, nil
			})
			got, err := client.collaboratorOperator(settings.CollaboratorTarget{ID: 1, OwnerID: 2, FullName: "owner/current"})
			if (err == nil) != (mode == "complete") {
				t.Fatal(got, err)
			}
			if mode == "complete" && (pages != 2 || got.Kind != "installation") {
				t.Fatal(pages, got)
			}
		})
	}
}
