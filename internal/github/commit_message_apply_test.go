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

func TestCommitMessageApplyVerifiesRawReadback(t *testing.T) {
	for _, menu := range []settings.CommitMessageMenu{settings.SquashCommitMenu, settings.MergeCommitMenu} {
		t.Run(string(menu), func(t *testing.T) {
			fallbackTitle, fallbackMessage := "COMMIT_OR_PR_TITLE", "COMMIT_MESSAGES"
			if menu == settings.MergeCommitMenu {
				fallbackTitle, fallbackMessage = "MERGE_MESSAGE", "PR_TITLE"
			}
			for _, test := range []struct {
				name, title, message, actualTitle, actualMessage string
				partial, wantError                               bool
				warnings                                         int
			}{
				{"fallback verified", fallbackTitle, fallbackMessage, fallbackTitle, fallbackMessage, false, false, 1},
				{"different verified", "PR_TITLE", "BLANK", "PR_TITLE", "BLANK", false, false, 1},
				{"fallback unconfirmed", fallbackTitle, fallbackMessage, "FUTURE_TITLE", "FUTURE_MESSAGE", false, true, 2},
				{"different unconfirmed", "PR_TITLE", "BLANK", "FUTURE_TITLE", "FUTURE_MESSAGE", false, true, 2},
				{"title only companion preserved", "PR_TITLE", "", "PR_TITLE", "FUTURE_MESSAGE", true, false, 2},
				{"title only companion changed", "PR_TITLE", "", "PR_TITLE", "OTHER_MESSAGE", true, true, 2},
			} {
				t.Run(test.name, func(t *testing.T) {
					var operations []string
					var warnings []settings.CommitMessageNormalization
					written := false
					desired := settings.RepositoryDeclaration{}
					prefix, titleField, messageField := "squash_merge_commit_", "squashMergeCommitTitle", "squashMergeCommitMessage"
					if menu == settings.MergeCommitMenu {
						prefix, titleField, messageField = "merge_commit_", "mergeCommitTitle", "mergeCommitMessage"
						desired.MergeCommitTitle = &test.title
						if !test.partial {
							desired.MergeCommitMessage = &test.message
						}
					} else {
						desired.SquashMergeCommitTitle = &test.title
						if !test.partial {
							desired.SquashMergeCommitMessage = &test.message
						}
					}
					wantPayload := map[string]any{prefix + "title": test.title}
					if !test.partial {
						wantPayload[prefix+"message"] = test.message
					}
					client := writeTestClient(t, func(r *http.Request) (*http.Response, error) {
						operations = append(operations, r.Method+" "+r.URL.Path)
						if r.Method == "PATCH" {
							var body map[string]any
							if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !reflect.DeepEqual(body, wantPayload) || r.URL.Path != "/repos/owner/current" {
								t.Fatalf("payload=%v want=%v err=%v", body, wantPayload, err)
							}
							written = true
							return labelsResponse(r, 200, `{}`), nil
						}
						if r.Method != "POST" || r.URL.Path != "/graphql" {
							t.Fatal("unexpected operation", r.Method, r.URL)
						}
						title, message := "FUTURE_TITLE", "FUTURE_MESSAGE"
						if written {
							title, message = test.actualTitle, test.actualMessage
						}
						body, _ := json.Marshal(map[string]any{"data": map[string]any{"repository": map[string]any{titleField: title, messageField: message}}})
						return labelsResponse(r, 200, string(body)), nil
					})
					client.CommitMessageNormalizationWarning = func(n settings.CommitMessageNormalization) { warnings = append(warnings, n) }
					err := client.ApplyRepository(repository.Repository{Owner: "owner", Name: "current"}, desired)
					if (err != nil) != test.wantError || !reflect.DeepEqual(operations, []string{"POST /graphql", "PATCH /repos/owner/current", "POST /graphql"}) || len(warnings) != test.warnings {
						t.Fatalf("operations=%v warnings=%+v err=%v", operations, warnings, err)
					}
					if err != nil {
						var preflight interface{ ReadFailedBeforeWrite() }
						if errors.As(err, &preflight) || !strings.Contains(err.Error(), "may already have applied") {
							t.Fatal("uncertainty mislabeled", err)
						}
					}
					for _, warning := range warnings {
						if warning.Path != menu.Path() || warning.Fallback != "default_message" {
							t.Fatal("unmanaged menu warning", warning)
						}
					}
					if warnings[0].RawTitle != "FUTURE_TITLE" || warnings[0].RawMessage != "FUTURE_MESSAGE" {
						t.Fatal("preflight raw provenance lost", warnings)
					}
					if len(warnings) == 2 && (warnings[1].RawTitle != test.actualTitle || warnings[1].RawMessage != test.actualMessage) {
						t.Fatal("fresh raw provenance lost", warnings)
					}
				})
			}
		})
	}
}

func TestCommitMessageApplyAcquisitionFailures(t *testing.T) {
	for _, afterWrite := range []bool{false, true} {
		for _, invalid := range []string{
			`{"data":{"repository":{"mergeCommitTitle":"MERGE_MESSAGE"}}}`,
			`{"data":{"repository":{"mergeCommitTitle":"MERGE_MESSAGE","mergeCommitMessage":null}}}`,
			`{"data":{"repository":{"mergeCommitTitle":"MERGE_MESSAGE","mergeCommitMessage":false}}}`,
			`{"data":{"repository":{"mergeCommitTitle":"MERGE_MESSAGE","mergeCommitMessage":"PR_TITLE"}},"errors":[{"message":"denied"}]}`,
		} {
			t.Run(fmt.Sprintf("after=%t/%s", afterWrite, invalid), func(t *testing.T) {
				patches, warnings := 0, 0
				client := writeTestClient(t, func(r *http.Request) (*http.Response, error) {
					if r.Method == "PATCH" {
						patches++
						return labelsResponse(r, 200, `{}`), nil
					}
					body := invalid
					if afterWrite && patches == 0 {
						body = `{"data":{"repository":{"mergeCommitTitle":"MERGE_MESSAGE","mergeCommitMessage":"PR_TITLE"}}}`
					}
					return labelsResponse(r, 200, body), nil
				})
				client.CommitMessageNormalizationWarning = func(settings.CommitMessageNormalization) { warnings++ }
				title, message := "PR_TITLE", "BLANK"
				err := client.ApplyRepository(repository.Repository{Owner: "owner", Name: "current"}, settings.RepositoryDeclaration{MergeCommitTitle: &title, MergeCommitMessage: &message})
				var preflight interface{ ReadFailedBeforeWrite() }
				wantPatches := 0
				if afterWrite {
					wantPatches = 1
				}
				if err == nil || errors.As(err, &preflight) == afterWrite || patches != wantPatches || warnings != 0 {
					t.Fatalf("patches=%d warnings=%d err=%v", patches, warnings, err)
				}
			})
		}
	}
}

func TestCommitMessageApplyRequiresVerificationClient(t *testing.T) {
	requests, warnings := 0, 0
	client := topicsClient(t, func(r *http.Request) (*http.Response, error) {
		requests++
		return labelsResponse(r, 200, `{}`), nil
	})
	client.CommitMessageNormalizationWarning = func(settings.CommitMessageNormalization) { warnings++ }
	title := "PR_TITLE"
	err := client.ApplyRepository(repository.Repository{Owner: "owner", Name: "current"}, settings.RepositoryDeclaration{MergeCommitTitle: &title})
	var preflight interface{ ReadFailedBeforeWrite() }
	if err == nil || !errors.As(err, &preflight) || requests != 0 || warnings != 0 {
		t.Fatalf("requests=%d warnings=%d err=%v", requests, warnings, err)
	}
}
