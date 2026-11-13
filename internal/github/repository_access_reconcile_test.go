package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestRepositoryAccessReconcilesEachSelectedControl(t *testing.T) {
	yes, no, all, limited := true, false, "ALL", "COLLABORATORS_ONLY"
	for _, test := range []struct {
		field         string
		desired       settings.RepositoryDeclaration
		before, after any
	}{
		{"hasPullRequestsEnabled", settings.RepositoryDeclaration{HasPullRequests: &no}, true, false},
		{"hasSponsorshipsEnabled", settings.RepositoryDeclaration{HasSponsorships: &yes}, false, true},
		{"pullRequestCreationPolicy", settings.RepositoryDeclaration{PullRequestCreationPolicy: &all}, "COLLABORATORS_ONLY", "ALL"},
		{"issueCreationPolicy", settings.RepositoryDeclaration{IssueCreationPolicy: &limited}, "ALL", "COLLABORATORS_ONLY"},
	} {
		t.Run(test.field, func(t *testing.T) {
			var operations []string
			value := test.before
			client := prAccessTestClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
				var request struct {
					Query     string
					Variables map[string]any
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Fatal(err)
				}
				fields := "id nameWithOwner " + test.field
				if !strings.Contains(request.Query, "{ "+fields+" }") {
					t.Fatal("selection broadened", request.Query)
				}
				response := map[string]any{"id": "repo-id", "nameWithOwner": "OWNER/CURRENT", test.field: value}
				var data any = map[string]any{"repository": response}
				if strings.HasPrefix(request.Query, "mutation") {
					operations = append(operations, "mutation")
					if !reflect.DeepEqual(request.Variables, map[string]any{"input": map[string]any{"repositoryId": "repo-id", test.field: test.after}}) {
						t.Fatal("unexpected mutation", request.Variables)
					}
					value = test.after
					response[test.field] = value
					data = map[string]any{"updateRepository": map[string]any{"repository": response}}
				} else {
					operations = append(operations, "query")
					if !strings.Contains(request.Query, "followRenames: false") || !reflect.DeepEqual(request.Variables, map[string]any{"owner": "owner", "name": "current"}) {
						t.Fatal("target changed", request)
					}
				}
				body, _ := json.Marshal(map[string]any{"data": data})
				return labelsResponse(r, 200, string(body)), nil
			}))
			for _, wantWrite := range []bool{true, false} {
				wrote, err := client.ApplyRepositoryAccess(repository.Repository{Owner: "owner", Name: "current"}, test.desired)
				if err != nil || wrote != wantWrite {
					t.Fatal("write evidence differs", wrote, err)
				}
			}
			if !reflect.DeepEqual(operations, []string{"query", "mutation", "query", "query"}) {
				t.Fatal(operations)
			}
		})
	}
}

func TestRepositoryAccessFailureBoundaries(t *testing.T) {
	for _, stage := range []string{"preflight", "ack", "readback"} {
		for _, defect := range []string{"missing", "null", "wrong type", "partial", "denied", "redirect", "transport", "wrong name", "wrong id", "stale"} {
			if stage == "preflight" && (defect == "wrong id" || defect == "stale") {
				continue
			}
			t.Run(stage+"/"+defect, func(t *testing.T) {
				calls := 0
				cause := errors.New("fixture transport failed")
				client := prAccessTestClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					at := map[string]int{"preflight": 1, "ack": 2, "readback": 3}[stage]
					bad := calls == at
					if bad && defect == "transport" {
						return nil, cause
					}
					if bad && defect == "denied" {
						return labelsResponse(r, 403, `{"message":"denied"}`), nil
					}
					if bad && defect == "redirect" {
						resp := labelsResponse(r, 307, `{}`)
						resp.Header.Set("Location", "https://api.github.com/other")
						return resp, nil
					}
					fields := map[string]any{"id": "repo-id", "nameWithOwner": "owner/current", "hasSponsorshipsEnabled": calls != 1}
					if bad {
						switch defect {
						case "missing":
							delete(fields, "hasSponsorshipsEnabled")
						case "null":
							fields["hasSponsorshipsEnabled"] = nil
						case "wrong type":
							fields["hasSponsorshipsEnabled"] = "true"
						case "wrong name":
							fields["nameWithOwner"] = "owner/other"
						case "wrong id":
							fields["id"] = "replacement-id"
						case "stale":
							fields["hasSponsorshipsEnabled"] = false
						}
					}
					data := map[string]any{"repository": fields}
					if calls == 2 {
						data = map[string]any{"updateRepository": map[string]any{"repository": fields}}
					}
					body := map[string]any{"data": data}
					if bad && defect == "partial" {
						body["errors"] = []map[string]string{{"message": "partial fixture"}}
					}
					encoded, _ := json.Marshal(body)
					return labelsResponse(r, 200, string(encoded)), nil
				}))
				yes := true
				wrote, err := client.ApplyRepositoryAccess(repository.Repository{Owner: "owner", Name: "current"}, settings.RepositoryDeclaration{HasSponsorships: &yes})
				var pre interface{ ReadFailedBeforeWrite() }
				wantWrote := stage == "readback" || stage == "ack" && defect != "partial" && defect != "denied" && defect != "redirect" && defect != "transport"
				if err == nil || wrote != wantWrote || errors.As(err, &pre) != (stage == "preflight") || calls != map[string]int{"preflight": 1, "ack": 2, "readback": 3}[stage] {
					t.Fatal("failure boundary lost", calls, wrote, err)
				}
				if stage != "preflight" && !strings.Contains(err.Error(), "may already have applied") {
					t.Fatal("uncertainty omitted", err)
				}
				if defect == "transport" && !errors.Is(err, cause) {
					t.Fatal(fmt.Sprintf("cause lost: %v", err))
				}
			})
		}
	}
}
