package github

import (
	"context"
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

func repositoryPatchBaseFixture() map[string]any {
	return map[string]any{"description": "", "homepage": "", "default_branch": "main", "has_issues": false, "has_projects": false, "has_wiki": false, "has_discussions": false, "web_commit_signoff_required": false, "is_template": false}
}

func TestRepositoryPatchReconcilesAllScalarControls(t *testing.T) {
	for _, test := range []struct {
		field, graphql string
		value          any
		declare        func(any) settings.RepositoryDeclaration
	}{
		{"description", "", "", func(v any) settings.RepositoryDeclaration {
			return settings.RepositoryDeclaration{Description: pointer(v.(string))}
		}},
		{"homepage", "", "", func(v any) settings.RepositoryDeclaration {
			return settings.RepositoryDeclaration{Homepage: pointer(v.(string))}
		}},
		{"default_branch", "", "release/v2", func(v any) settings.RepositoryDeclaration {
			return settings.RepositoryDeclaration{DefaultBranch: pointer(v.(string))}
		}},
		{"has_issues", "", false, func(v any) settings.RepositoryDeclaration {
			return settings.RepositoryDeclaration{HasIssues: pointer(v.(bool))}
		}},
		{"has_projects", "", false, func(v any) settings.RepositoryDeclaration {
			return settings.RepositoryDeclaration{HasProjects: pointer(v.(bool))}
		}},
		{"has_wiki", "", false, func(v any) settings.RepositoryDeclaration {
			return settings.RepositoryDeclaration{HasWiki: pointer(v.(bool))}
		}},
		{"has_discussions", "", false, func(v any) settings.RepositoryDeclaration {
			return settings.RepositoryDeclaration{HasDiscussions: pointer(v.(bool))}
		}},
		{"web_commit_signoff_required", "", false, func(v any) settings.RepositoryDeclaration {
			return settings.RepositoryDeclaration{WebCommitSignoffRequired: pointer(v.(bool))}
		}},
		{"is_template", "", false, func(v any) settings.RepositoryDeclaration {
			return settings.RepositoryDeclaration{IsTemplate: pointer(v.(bool))}
		}},
		{"allow_merge_commit", "mergeCommitAllowed", false, func(v any) settings.RepositoryDeclaration {
			return settings.RepositoryDeclaration{AllowMergeCommit: pointer(v.(bool))}
		}},
		{"allow_squash_merge", "squashMergeAllowed", false, func(v any) settings.RepositoryDeclaration {
			return settings.RepositoryDeclaration{AllowSquashMerge: pointer(v.(bool))}
		}},
		{"allow_rebase_merge", "rebaseMergeAllowed", false, func(v any) settings.RepositoryDeclaration {
			return settings.RepositoryDeclaration{AllowRebaseMerge: pointer(v.(bool))}
		}},
		{"delete_branch_on_merge", "deleteBranchOnMerge", false, func(v any) settings.RepositoryDeclaration {
			return settings.RepositoryDeclaration{DeleteBranchOnMerge: pointer(v.(bool))}
		}},
		{"allow_auto_merge", "autoMergeAllowed", false, func(v any) settings.RepositoryDeclaration {
			return settings.RepositoryDeclaration{AllowAutoMerge: pointer(v.(bool))}
		}},
		{"allow_update_branch", "allowUpdateBranch", false, func(v any) settings.RepositoryDeclaration {
			return settings.RepositoryDeclaration{AllowUpdateBranch: pointer(v.(bool))}
		}},
	} {
		values := []any{test.value}
		if _, ok := test.value.(bool); ok {
			values = append(values, true)
		}
		for _, value := range values {
			for _, matching := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%v/matching=%v", test.field, value, matching), func(t *testing.T) {
					patches, reads, warnings := 0, 0, 0
					client := writeTestClient(t, func(req *http.Request) (*http.Response, error) {
						if req.Method == "PATCH" {
							patches++
							var body map[string]any
							if err := json.NewDecoder(req.Body).Decode(&body); err != nil || !reflect.DeepEqual(body, map[string]any{test.field: value}) {
								t.Fatal("nonminimal payload", body, err)
							}
							data, _ := json.Marshal(body)
							return labelsResponse(req, 200, string(data)), nil
						}
						reads++
						actual := value
						if !matching && patches == 0 {
							if boolean, ok := value.(bool); ok {
								actual = !boolean
							} else {
								actual = "before"
							}
						}
						fields := repositoryPatchBaseFixture()
						fields[test.field] = actual
						var response any = fields
						if test.graphql != "" {
							if req.Method != "POST" || req.URL.Path != "/graphql" {
								t.Fatal("merge-only read reached REST", req.Method, req.URL)
							}
							assertRepositoryPatchQuery(t, req, []string{test.graphql})
							response = map[string]any{"data": map[string]any{"repository": map[string]any{test.graphql: actual}}}
						} else if req.Method != "GET" || req.URL.Path != "/repos/owner/current" {
							t.Fatal("base-only read reached unrelated API", req.Method, req.URL)
						}
						data, _ := json.Marshal(response)
						return labelsResponse(req, 200, string(data)), nil
					})
					client.CommitMessageNormalizationWarning = func(settings.CommitMessageNormalization) { warnings++ }
					wrote, err := client.ApplyRepository(repository.Repository{Owner: "owner", Name: "current"}, test.declare(value))
					wantPatches, wantReads := 1, 2
					if matching {
						wantPatches, wantReads = 0, 1
					}
					if err != nil || wrote == matching || patches != wantPatches || reads != wantReads || warnings != 0 {
						t.Fatalf("wrote=%v patches=%d reads=%d warnings=%d err=%v", wrote, patches, reads, warnings, err)
					}
				})
			}
		}
	}
}

func assertRepositoryPatchQuery(t *testing.T, req *http.Request, want []string) {
	t.Helper()
	var body struct {
		Query     string
		Variables map[string]any
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	start := strings.Index(body.Query, "repository(")
	if start < 0 || !strings.Contains(body.Query, "followRenames: false") || !reflect.DeepEqual(body.Variables, map[string]any{"owner": "owner", "name": "current"}) {
		t.Fatal("unexpected query", body)
	}
	start += strings.Index(body.Query[start:], "{")
	end := start + strings.Index(body.Query[start:], "}")
	if got := strings.Fields(body.Query[start+1 : end]); !reflect.DeepEqual(got, want) {
		t.Fatalf("query fields=%v want=%v", got, want)
	}
}

func TestRepositoryPatchMenusAreConditionalWholeControls(t *testing.T) {
	for _, menu := range []string{"merge", "squash"} {
		for _, matching := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/matching=%v", menu, matching), func(t *testing.T) {
				patches, reads := 0, 0
				titleField, messageField, prefix := "mergeCommitTitle", "mergeCommitMessage", "merge_commit_"
				queryFields := []string{"mergeCommitAllowed", titleField, messageField}
				desired := settings.RepositoryDeclaration{MergeCommitTitle: pointer("PR_TITLE"), MergeCommitMessage: pointer("PR_BODY")}
				if menu == "squash" {
					titleField, messageField, prefix = "squashMergeCommitTitle", "squashMergeCommitMessage", "squash_merge_commit_"
					queryFields = []string{titleField, messageField}
					desired = settings.RepositoryDeclaration{SquashMergeCommitTitle: pointer("PR_TITLE"), SquashMergeCommitMessage: pointer("PR_BODY")}
				}
				client := writeTestClient(t, func(req *http.Request) (*http.Response, error) {
					if req.Method == "PATCH" {
						patches++
						var body map[string]any
						if err := json.NewDecoder(req.Body).Decode(&body); err != nil || !reflect.DeepEqual(body, map[string]any{prefix + "title": "PR_TITLE", prefix + "message": "PR_BODY"}) {
							t.Fatal("split pair or invented enabling field", body, err)
						}
						return labelsResponse(req, 200, `{}`), nil
					}
					reads++
					assertRepositoryPatchQuery(t, req, queryFields)
					message := "BLANK"
					if matching || patches > 0 {
						message = "PR_BODY"
					}
					fields := map[string]any{titleField: "PR_TITLE", messageField: message}
					if menu == "merge" {
						fields["mergeCommitAllowed"] = true
					}
					data, _ := json.Marshal(map[string]any{"data": map[string]any{"repository": fields}})
					return labelsResponse(req, 200, string(data)), nil
				})
				client.CommitMessageNormalizationWarning = func(settings.CommitMessageNormalization) { t.Fatal("canonical menu emitted normalization") }
				wrote, err := client.ApplyRepository(repository.Repository{Owner: "owner", Name: "current"}, desired)
				wantPatches, wantReads := 1, 2
				if matching {
					wantPatches, wantReads = 0, 1
				}
				if err != nil || wrote == matching || patches != wantPatches || reads != wantReads {
					t.Fatal(wrote, patches, reads, err)
				}
				// The same declaration converges; the next fresh observation is
				// sufficient and must not repeat either mutation or verification.
				wrote, err = client.ApplyRepository(repository.Repository{Owner: "owner", Name: "current"}, desired)
				if err != nil || wrote || patches != wantPatches || reads != wantReads+1 {
					t.Fatal("repeated menu did not converge", wrote, patches, reads, err)
				}
			})
		}
	}
}

func TestRepositoryPatchObservedDisabledMergeMenuFailsBeforeWrite(t *testing.T) {
	requests := 0
	client := writeTestClient(t, func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != "POST" || req.URL.Path != "/graphql" {
			t.Fatal("disabled menu mutated", req.Method, req.URL)
		}
		assertRepositoryPatchQuery(t, req, []string{"mergeCommitAllowed", "mergeCommitTitle", "mergeCommitMessage"})
		return labelsResponse(req, 200, `{"data":{"repository":{"mergeCommitAllowed":false,"mergeCommitTitle":"PR_TITLE","mergeCommitMessage":"BLANK"}}}`), nil
	})
	wrote, err := client.ApplyRepository(repository.Repository{Owner: "owner", Name: "current"}, settings.RepositoryDeclaration{MergeCommitTitle: pointer("PR_TITLE"), MergeCommitMessage: pointer("PR_BODY")})
	var prerequisite interface{ ReadFailedBeforeWrite() }
	if err == nil || wrote || requests != 1 || !errors.As(err, &prerequisite) || !strings.Contains(err.Error(), "allow_merge_commits: true") {
		t.Fatal(wrote, requests, err)
	}
}

func TestRepositoryPatchVerifiesOriginalSelection(t *testing.T) {
	for _, drift := range []string{"none", "matching base", "matching merge"} {
		t.Run(drift, func(t *testing.T) {
			patches := 0
			var operations []string
			client := writeTestClient(t, func(req *http.Request) (*http.Response, error) {
				operations = append(operations, req.Method+" "+req.URL.Path)
				if req.Method == "PATCH" {
					patches++
					var body map[string]any
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil || !reflect.DeepEqual(body, map[string]any{"description": "after"}) {
						t.Fatal(body, err)
					}
					return labelsResponse(req, 200, `{}`), nil
				}
				if req.Method == "GET" {
					fields := repositoryPatchBaseFixture()
					fields["has_issues"] = true
					if patches > 0 {
						fields["description"] = "after"
						if drift == "matching base" {
							fields["has_issues"] = false
						}
					}
					data, _ := json.Marshal(fields)
					return labelsResponse(req, 200, string(data)), nil
				}
				assertRepositoryPatchQuery(t, req, []string{"autoMergeAllowed"})
				allowed := true
				if patches > 0 && drift == "matching merge" {
					allowed = false
				}
				return labelsResponse(req, 200, fmt.Sprintf(`{"data":{"repository":{"autoMergeAllowed":%t}}}`, allowed)), nil
			})
			wrote, err := client.ApplyRepository(repository.Repository{Owner: "owner", Name: "current"}, settings.RepositoryDeclaration{Description: pointer("after"), HasIssues: pointer(true), AllowAutoMerge: pointer(true)})
			var prerequisite interface{ ReadFailedBeforeWrite() }
			if !wrote || patches != 1 || (err != nil) != (drift != "none") || errors.As(err, &prerequisite) || !reflect.DeepEqual(operations, []string{"GET /repos/owner/current", "POST /graphql", "PATCH /repos/owner/current", "GET /repos/owner/current", "POST /graphql"}) {
				t.Fatal(wrote, operations, err)
			}
		})
	}
}

func TestRepositoryPatchReadFailuresHaveCorrectCertainty(t *testing.T) {
	for _, graphql := range []bool{false, true} {
		for _, afterWrite := range []bool{false, true} {
			for _, failure := range []string{"denied", "missing", "null", "mistyped", "malformed", "timeout", "redirect", "partial GraphQL"} {
				if failure == "partial GraphQL" && !graphql {
					continue
				}
				t.Run(fmt.Sprintf("graphql=%v/after=%v/%s", graphql, afterWrite, failure), func(t *testing.T) {
					patches := 0
					client := writeTestClient(t, func(req *http.Request) (*http.Response, error) {
						if req.Method == "PATCH" {
							patches++
							return labelsResponse(req, 200, `{}`), nil
						}
						fields := repositoryPatchBaseFixture()
						name := "has_issues"
						if graphql {
							fields = map[string]any{"autoMergeAllowed": false}
							name = "autoMergeAllowed"
						}
						if !afterWrite || patches > 0 {
							switch failure {
							case "denied":
								return labelsResponse(req, 403, `{"message":"denied"}`), nil
							case "missing":
								delete(fields, name)
							case "null":
								fields[name] = nil
							case "mistyped":
								fields[name] = "false"
							case "malformed":
								return labelsResponse(req, 200, `invalid`), nil
							case "timeout":
								return nil, context.DeadlineExceeded
							case "redirect":
								response := labelsResponse(req, 302, `{}`)
								response.Header.Set("Location", "https://other.test/repository")
								return response, nil
							}
						}
						var body any = fields
						if graphql {
							envelope := map[string]any{"data": map[string]any{"repository": fields}}
							if failure == "partial GraphQL" && (!afterWrite || patches > 0) {
								envelope["errors"] = []any{map[string]any{"message": "denied"}}
							}
							body = envelope
						}
						data, _ := json.Marshal(body)
						return labelsResponse(req, 200, string(data)), nil
					})
					desired := settings.RepositoryDeclaration{HasIssues: pointer(true)}
					if graphql {
						desired = settings.RepositoryDeclaration{AllowAutoMerge: pointer(true)}
					}
					wrote, err := client.ApplyRepository(repository.Repository{Owner: "owner", Name: "current"}, desired)
					var prerequisite interface{ ReadFailedBeforeWrite() }
					wantPatches := 0
					if afterWrite {
						wantPatches = 1
					}
					if err == nil || wrote != afterWrite || patches != wantPatches || errors.As(err, &prerequisite) == afterWrite {
						t.Fatalf("wrote=%v patches=%d err=%v", wrote, patches, err)
					}
				})
			}
		}
	}
}
