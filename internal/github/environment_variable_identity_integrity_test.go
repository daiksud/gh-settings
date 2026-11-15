package github

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestEnvironmentVariableIdentityPreservesValidUnicodeAndUnownedProtection(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"�", `{"id":42,"name":"�","protection_rules":[{"type":"future"}]}`},
		{"😀", `{"id":42,"name":"\ud83d\ude00","unknown":{"type":"future"}}`},
		{`\ud800`, `{"id":42,"name":"\\ud800"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				return labelsResponse(r, 200, tc.body), nil
			}))
			_, id, err := client.environmentVariableScope(repository.Repository{Owner: "owner", Name: "current"}, tc.name)
			if err != nil || id != 42 || calls != 1 {
				t.Fatalf("id=%d calls=%d err=%v", id, calls, err)
			}
		})
	}
}

func TestEnvironmentVariableIdentityRejectsLossyUnicodeMatch(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"invalid utf8", "{\"id\":42,\"name\":\"\xff\"}"},
		{"unpaired high", `{"id":42,"name":"\ud800"}`},
		{"unpaired low", `{"id":42,"name":"\udc00"}`},
	} {
		for _, write := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{false: " read", true: " write"}[write], func(t *testing.T) {
				calls := 0
				client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					if calls != 1 || r.Method != http.MethodGet {
						t.Fatalf("unexpected variable request %s %s", r.Method, r.URL)
					}
					return labelsResponse(r, 200, tc.body), nil
				}))
				repo := repository.Repository{Owner: "owner", Name: "current"}
				var err error
				if write {
					_, err = client.UpsertEnvironmentVariable(repo, "�", settings.RepositoryVariable{Name: "ITEM", Value: "desired"})
				} else {
					_, err = client.ReadEnvironmentVariable(repo, "�", "ITEM")
				}
				var pre interface{ ReadFailedBeforeWrite() }
				if err == nil || errors.As(err, &pre) != write || calls != 1 {
					t.Fatalf("err=%v calls=%d write=%v", err, calls, write)
				}
			})
		}
	}
}

func TestEnvironmentVariablePruneRejectsAmbiguousParentIdentity(t *testing.T) {
	for _, stage := range []string{"first inventory parent", "fresh inventory parent", "before delete", "after delete"} {
		t.Run(stage, func(t *testing.T) {
			parentReads, variableReads, deletes := 0, 0, 0
			client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/repos/owner/current/environments/prod" {
					parentReads++
					bad := stage == "first inventory parent" && parentReads == 1 || stage == "fresh inventory parent" && parentReads == 2 || stage == "before delete" && parentReads == 3 || stage == "after delete" && parentReads == 4
					if bad {
						return labelsResponse(r, 200, `{"id":99,"ID":42,"id":42,"name":"prod"}`), nil
					}
					return labelsResponse(r, 200, `{"id":42,"name":"prod","protection_rules":[{"type":"future"}]}`), nil
				}
				if !strings.HasPrefix(r.URL.Path, "/repos/owner/current/environments/prod/variables") {
					t.Fatalf("unexpected request %s %s", r.Method, r.URL)
				}
				if r.Method == http.MethodDelete {
					deletes++
					return labelsResponse(r, 204, ""), nil
				}
				variableReads++
				if strings.HasSuffix(r.URL.Path, "/OLD") {
					return labelsResponse(r, 200, `{"name":"OLD","value":"before"}`), nil
				}
				if deletes > 0 {
					return labelsResponse(r, 200, `{"total_count":0,"variables":[]}`), nil
				}
				return labelsResponse(r, 200, `{"total_count":1,"variables":[{"name":"OLD","value":"before"}]}`), nil
			}))
			repo := repository.Repository{Owner: "owner", Name: "current"}
			inventory, err := client.ReadEnvironmentVariables(repo, "prod")
			if stage == "first inventory parent" || stage == "fresh inventory parent" {
				wantReads := 1
				if stage == "fresh inventory parent" {
					wantReads = 2
				}
				if err == nil || parentReads != wantReads || deletes != 0 || stage == "first inventory parent" && variableReads != 0 {
					t.Fatalf("err=%v parent=%d variable=%d deletes=%d", err, parentReads, variableReads, deletes)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			id, err := inventory.EnvironmentID("prod")
			if err != nil {
				t.Fatal(err)
			}
			change, err := settings.NewVariableDelete(settings.RepositoryVariable{Name: "OLD", Value: "before"})
			if err != nil {
				t.Fatal(err)
			}
			err = client.DeleteEnvironmentVariable(repo, "prod", id, change)
			var pre interface{ ReadFailedBeforeWrite() }
			if stage == "before delete" {
				if err == nil || !errors.As(err, &pre) || parentReads != 3 || deletes != 0 || variableReads != 1 {
					t.Fatalf("err=%v parent=%d variable=%d deletes=%d", err, parentReads, variableReads, deletes)
				}
			} else if err == nil || errors.As(err, &pre) || !strings.Contains(err.Error(), "write may have applied") || parentReads != 4 || deletes != 1 {
				t.Fatalf("err=%v parent=%d variable=%d deletes=%d", err, parentReads, variableReads, deletes)
			}
		})
	}
}

// Environment detail metadata is unowned, but the response must still fit the
// same bounded, complete JSON envelope used by other Environment reads.
func TestEnvironmentVariableIdentityResponseSizeBoundary(t *testing.T) {
	const prefix = `{"id":42,"name":"prod","unknown":"`
	const suffix = `"}`
	for _, tc := range []struct {
		name  string
		size  int
		valid bool
	}{
		{"at limit", environmentResponseLimit, true},
		{"over limit", environmentResponseLimit + 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := prefix + strings.Repeat("x", tc.size-len(prefix)-len(suffix)) + suffix
			client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
				return labelsResponse(r, http.StatusOK, body), nil
			}))
			_, id, err := client.environmentVariableScope(repository.Repository{Owner: "owner", Name: "current"}, "prod")
			if (err == nil) != tc.valid || tc.valid && id != 42 {
				t.Fatalf("id=%d err=%v", id, err)
			}
		})
	}
}
