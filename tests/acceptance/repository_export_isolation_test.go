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
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestRepositoryExportIndependentObservationsAcceptance(t *testing.T) {
	const upstream = "private-upstream-detail-must-not-escape"
	const prior = `version: daiksud/gh-settings@v0
repository:
  about: {description: ''}
  settings:
    general:
      features:
        wikis: false
        issue_permissions: {creation_allowed_by: collaborators_only}
      pull_requests: {allow_auto_merge: true}
`
	for _, failure := range []string{"merge", "base", "all", "disabled-issues"} {
		for _, existing := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/prior=%t", failure, existing), func(t *testing.T) {
				fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					merge := isMergeMethodsFixtureRead(r)
					issue := isIssueCreationPolicyFixtureRead(r)
					coreQuery := merge || issue || isPRAccessFixtureRead(r) || isSponsorshipsFixtureRead(r)
					if coreQuery && (failure == "all" || failure == "merge" && merge || failure == "disabled-issues" && issue) {
						// Even plausible complete data must be discarded when GraphQL reports errors.
						var response map[string]any
						if err := json.Unmarshal([]byte(mergeMethodsFixtureResponse), &response); err != nil {
							t.Error(err)
						}
						response["errors"] = []any{map[string]string{"message": upstream}}
						_ = json.NewEncoder(w).Encode(response)
						return
					}
					if servePRAccessFixtureRead(w, r) || serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
						return
					}
					if merge {
						fmt.Fprint(w, mergeMethodsFixtureResponse)
						return
					}
					switch r.URL.Path {
					case "/repos/owner/current":
						if failure == "base" || failure == "all" {
							w.WriteHeader(http.StatusForbidden)
							_ = json.NewEncoder(w).Encode(map[string]string{"message": upstream})
						} else if failure == "disabled-issues" {
							fmt.Fprint(w, strings.Replace(planResponse, `"has_issues":true`, `"has_issues":false`, 1))
						} else {
							fmt.Fprint(w, planResponse)
						}
					case "/repos/owner/current/topics":
						fmt.Fprint(w, `{"names":["fresh"]}`)
					case "/repos/owner/current/labels":
						fmt.Fprint(w, `[]`)
					default:
						t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
						w.WriteHeader(http.StatusBadRequest)
					}
				}, true)
				if err != nil {
					t.Fatal(err)
				}
				defer fixture.close()
				env := exportFixtureEnv(fixture.cfg)
				if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
					t.Fatal(err)
				}
				if existing {
					if err := writeSettingsFile(fixture.work, prior); err != nil {
						t.Fatal(err)
					}
				}
				binary := acceptanceBinary(t)
				result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"export"}, false)
				if err != nil || result.runErr != nil || result.stdout.Len() != 0 || hasNonInfoDiagnostic(result.stderr.String()) {
					t.Fatalf("independent acquisition failed: err=%v result=%+v", err, result)
				}
				infoPath := map[string]string{"merge": "repository.settings.general.pull_requests", "base": "repository", "all": "repository", "disabled-issues": "repository.settings.general.features.issue_permissions"}[failure]
				if !strings.Contains(result.stderr.String(), "info: export "+infoPath+":") || strings.Contains(result.stderr.String(), upstream) {
					t.Fatalf("missing or unbounded family diagnostic: %s", result.stderr.String())
				}
				if failure == "disabled-issues" && existing && !strings.Contains(result.stderr.String(), "where compatible with acquired settings") {
					t.Fatal("suppressed prior policy was described as unconditionally retained", result.stderr.String())
				}
				raw, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
				if err != nil {
					t.Fatal(err)
				}
				var document map[string]any
				if err := yaml.Unmarshal(raw, &document); err != nil {
					t.Fatal(err)
				}
				repo := exportMap(document, "repository")
				about := exportMap(repo, "about")
				general := exportMap(repo, "settings", "general")
				features := exportMap(general, "features")
				pullRequests := exportMap(general, "pull_requests")
				wantAbout := map[string]any{"topics": []any{"fresh"}}
				wantFeatures := map[string]any{}
				if failure != "base" && failure != "all" {
					wantAbout["description"], wantAbout["website"] = "old", "old-url"
					wantFeatures = map[string]any{"wikis": true, "issues": failure != "disabled-issues", "projects": true, "discussions": false}
					if general["default_branch"] != "main" {
						t.Fatal("readable base values lost", general)
					}
				} else {
					if _, found := general["default_branch"]; found {
						t.Fatal("unobserved default branch invented", general)
					}
					if existing {
						wantAbout["description"] = ""
						wantFeatures["wikis"] = false
					}
				}
				if failure != "all" {
					wantFeatures["sponsorships"], wantFeatures["pull_requests"] = false, false
					if failure != "disabled-issues" {
						wantFeatures["issue_permissions"] = map[string]any{"creation_allowed_by": "all_users"}
					}
				} else if existing {
					wantFeatures["issue_permissions"] = map[string]any{"creation_allowed_by": "collaborators_only"}
				}
				wantPullRequests := map[string]any{}
				if failure != "merge" && failure != "all" {
					wantPullRequests = map[string]any{"allow_merge_commits": false, "allow_squash_merging": false, "allow_rebase_merging": false, "always_suggest_updating_pull_request_branches": false, "allow_auto_merge": false, "automatically_delete_head_branches": false}
				} else if existing {
					wantPullRequests["allow_auto_merge"] = true
				}
				if !reflect.DeepEqual(about, wantAbout) || !reflect.DeepEqual(features, wantFeatures) || !reflect.DeepEqual(pullRequests, wantPullRequests) || !reflect.DeepEqual(repo["labels"], []any{}) {
					t.Fatalf("export=%s\nwant About=%v Features=%v Pull requests=%v and empty Labels", raw, wantAbout, wantFeatures, wantPullRequests)
				}
				reads := fixture.reads.Load()
				validated, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"validate"}, false)
				if err != nil || validated.runErr != nil || fixture.reads.Load() != reads {
					t.Fatalf("partial export did not validate locally: %v %+v", err, validated)
				}
				if err := fixture.checkRequests(false); err != nil {
					t.Fatal(err)
				}
				if failure == "merge" || failure == "all" {
					if err := writeSettingsFile(fixture.work, generalInput("pull_requests: {allow_auto_merge: true}\n")); err != nil {
						t.Fatal(err)
					}
					planned, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"plan"}, false)
					if err != nil || planned.runErr == nil || planned.stdout.Len() != 0 {
						t.Fatalf("plan lost its strict observation boundary: %v %+v", err, planned)
					}
				}
			})
		}
	}
}

func exportMap(value map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		child, ok := value[key].(map[string]any)
		if !ok {
			return map[string]any{}
		}
		value = child
	}
	return value
}

func readExportDocument(t *testing.T, directory string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(directory, ".github/settings.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	return document
}
