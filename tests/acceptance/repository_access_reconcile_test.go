//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestRepositoryAccessReconciliationAcceptance(t *testing.T) {
	for _, mode := range []string{"matching", "changed", "stale checkbox", "matching member drift", "denied readback"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			remote := map[string]any{"id": "opaque-id", "nameWithOwner": "owner/current", "hasPullRequestsEnabled": true, "pullRequestCreationPolicy": "ALL", "issueCreationPolicy": "COLLABORATORS_ONLY", "hasSponsorshipsEnabled": false}
			if mode == "matching" {
				remote["hasSponsorshipsEnabled"] = true
			}
			var operations []string
			var payloads []map[string]any
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Method != http.MethodPost || r.URL.Path != "/graphql" {
					t.Error("unexpected later request", r.Method, r.URL.Path)
					w.WriteHeader(400)
					return
				}
				var request struct {
					Query     string
					Variables map[string]any
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
				}
				fields := "id nameWithOwner hasPullRequestsEnabled pullRequestCreationPolicy issueCreationPolicy hasSponsorshipsEnabled"
				if mode == "stale checkbox" {
					fields = "id nameWithOwner hasSponsorshipsEnabled"
				}
				if !strings.Contains(request.Query, "{ "+fields+" }") {
					t.Error("original selection changed", request.Query)
				}
				if strings.HasPrefix(request.Query, "query") {
					operations = append(operations, "query")
					if !reflect.DeepEqual(request.Variables, map[string]any{"owner": "owner", "name": "current"}) || !strings.Contains(request.Query, "followRenames: false") {
						t.Error("unexpected target query", request)
					}
					if mode == "denied readback" && len(payloads) > 0 {
						fmt.Fprint(w, `{"data":{"repository":null},"errors":[{"message":"fixture denied readback"}]}`)
						return
					}
					json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"repository": remote}})
					return
				}
				operations = append(operations, "mutation")
				input, _ := request.Variables["input"].(map[string]any)
				payloads = append(payloads, input)
				ack := make(map[string]any, len(remote))
				for key, value := range remote {
					ack[key] = value
				}
				for key, value := range input {
					if key != "repositoryId" {
						ack[key] = value
						if mode != "stale checkbox" {
							remote[key] = value
						}
					}
				}
				if mode == "matching member drift" {
					remote["hasPullRequestsEnabled"] = false
				}
				json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"updateRepository": map[string]any{"repository": ack}}})
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			fields := "features: {pull_requests: true, pull_request_permissions: {creation_allowed_by: all_users}, issue_permissions: {creation_allowed_by: collaborators_only}, sponsorships: true}\n"
			if mode == "stale checkbox" {
				fields = "features: {sponsorships: true}\n"
			}
			input := generalInput(fields)
			wantFailure := mode != "matching" && mode != "changed"
			if wantFailure {
				input = repositoryInput("about: {topics: [must-not-be-written]}\nsettings:\n  general:\n" + indentInput(fields, 4))
			}
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			attempts := 1
			if !wantFailure {
				attempts = 2
			}
			for attempt := 0; attempt < attempts; attempt++ {
				result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"apply"}, false)
				if err != nil || (result.runErr != nil) != wantFailure {
					t.Fatalf("attempt=%d run=%v error=%v stdout=%q stderr=%q", attempt, result.runErr, err, result.stdout.String(), result.stderr.String())
				}
				if wantFailure && (!strings.Contains(result.stderr.String(), "may already have applied") || strings.Contains(result.stderr.String(), "earlier repository writes remain applied")) {
					t.Fatal("uncertain access write misreported", result.stderr.String())
				}
			}
			mu.Lock()
			defer mu.Unlock()
			want := []string{"query", "mutation", "query"}
			if mode == "matching" {
				want = []string{"query"}
			} else if !reflect.DeepEqual(payloads, []map[string]any{{"repositoryId": "opaque-id", "hasSponsorshipsEnabled": true}}) {
				t.Fatal("mutation was not a minimal declared delta", payloads)
			}
			if attempts == 2 {
				want = append(want, "query")
			}
			if !reflect.DeepEqual(operations, want) || mode == "matching" && len(payloads) != 0 {
				t.Fatalf("operations=%v payloads=%v, want %v", operations, payloads, want)
			}
			content, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			if err != nil || string(content) != input {
				t.Fatal("apply changed the declaration", string(content), err)
			}
		})
	}
}
