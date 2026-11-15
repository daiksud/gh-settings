package github

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
)

func TestEnvironmentSecretMetadataRejectsAmbiguousPages(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		ok         bool
	}{
		{"empty", `{"total_count":0,"secrets":[]}`, 200, true},
		{"metadata", `{"total_count":1,"secrets":[{"name":"SECRET_CANARY","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`, 200, true},
		{"valid surrogate pair", `{"total_count":1,"secrets":[{"name":"\ud83d\ude00","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`, 200, true},
		{"literal backslash", `{"total_count":1,"secrets":[{"name":"\\ud800","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`, 200, true},
		{"valid replacement", `{"total_count":1,"secrets":[{"name":"�","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`, 200, true},
		{"duplicate total", `{"total_count":1,"total_count":0,"secrets":[]}`, 200, false},
		{"alias total", `{"total_count":1,"TOTAL_COUNT":0,"secrets":[]}`, 200, false},
		{"duplicate list", `{"total_count":0,"secrets":[{}],"secrets":[]}`, 200, false},
		{"alias list", `{"total_count":0,"Secrets":[{}],"secrets":[]}`, 200, false},
		{"duplicate timestamp", `{"total_count":1,"secrets":[{"name":"SECRET_CANARY","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`, 200, false},
		{"alias timestamp", `{"total_count":1,"secrets":[{"name":"SECRET_CANARY","created_at":"2026-01-01T00:00:00Z","Updated_At":"2026-01-02T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`, 200, false},
		{"duplicate name", `{"total_count":1,"secrets":[{"name":"OTHER","name":"SECRET_CANARY","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`, 200, false},
		{"alias name", `{"total_count":1,"secrets":[{"Name":"OTHER","name":"SECRET_CANARY","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`, 200, false},
		{"unknown value", `{"total_count":1,"secrets":[{"name":"SECRET_CANARY","PRIVATE_FIELD_CANARY":"PRIVATE_CANARY","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`, 200, false},
		{"trailing", `{"total_count":0,"secrets":[]} {}`, 200, false},
		{"invalid utf8", "{\"total_count\":0,\"secrets\":[],\"\xff\":1}", 200, false},
		{"unpaired surrogate", `{"total_count":0,"secrets":[],"\ud800":1}`, 200, false},
		{"invalid utf8 name", "{\"total_count\":1,\"secrets\":[{\"name\":\"\xff\",\"created_at\":\"2026-01-01T00:00:00Z\",\"updated_at\":\"2026-01-01T00:00:00Z\"}]}", 200, false},
		{"unpaired surrogate name", `{"total_count":1,"secrets":[{"name":"\ud800","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`, 200, false},
		{"partial status", `{"total_count":0,"secrets":[]}`, 206, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != http.MethodGet || r.URL.Path != "/repos/owner/current/environments/prod/secrets" || r.URL.Query().Get("page") != "1" || r.URL.Query().Get("per_page") != "100" {
					t.Fatalf("unexpected request %s %s", r.Method, r.URL)
				}
				return actionsResponse(r, tc.status, tc.body), nil
			}).rest}
			values, err := c.readEnvironmentSecretMetadata(repository.Repository{Owner: "owner", Name: "current"}, "prod")
			if (err == nil) != tc.ok || calls != 1 || !tc.ok && values != nil {
				t.Fatalf("err=%v calls=%d returned=%d", err, calls, len(values))
			}
			if err != nil && (strings.Contains(err.Error(), "SECRET_CANARY") || strings.Contains(err.Error(), "PRIVATE_CANARY") || strings.Contains(err.Error(), "PRIVATE_FIELD_CANARY")) {
				t.Fatal("private metadata leaked")
			}
		})
	}
}

func TestEnvironmentSecretMetadataSecondPageAmbiguityFails(t *testing.T) {
	requests := 0
	c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
		requests++
		if r.Method != http.MethodGet || r.URL.Query().Get("page") != fmt.Sprint(requests) {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL)
		}
		if requests == 2 {
			return actionsResponse(r, 200, `{"total_count":101,"TOTAL_COUNT":100,"secrets":[]}`), nil
		}
		entries := make([]string, 100)
		for i := range entries {
			entries[i] = fmt.Sprintf(`{"name":"KEY_%d","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`, i)
		}
		return actionsResponse(r, 200, `{"total_count":101,"secrets":[`+strings.Join(entries, ",")+`]}`), nil
	}).rest}
	values, err := c.readEnvironmentSecretMetadata(repository.Repository{Owner: "owner", Name: "current"}, "prod")
	if err == nil || requests != 2 || values != nil {
		t.Fatalf("err=%v requests=%d returned=%d", err, requests, len(values))
	}
}

func TestEnvironmentSecretMetadataAmbiguityStopsPruneDelete(t *testing.T) {
	for _, mode := range []string{"duplicate timestamp", "alias count", "contradictory next"} {
		t.Run(mode, func(t *testing.T) {
			deleting, writes, secretReads := false, 0, 0
			repo := repository.Repository{Owner: "owner", Name: "current"}
			c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodDelete {
					writes++
					return actionsResponse(r, 204, ""), nil
				}
				if r.Method != http.MethodGet {
					t.Fatalf("unexpected request %s %s", r.Method, r.URL)
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					return actionsResponse(r, 200, `{"id":71,"full_name":"owner/current","archived":false}`), nil
				case "/repos/owner/current/environments":
					return actionsResponse(r, 200, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`), nil
				case "/repos/owner/current/environments/production":
					return actionsResponse(r, 200, emptyEnvironmentFixture), nil
				case "/repos/owner/current/environments/production/deployment_protection_rules":
					return actionsResponse(r, 200, customEnvironmentEmpty), nil
				case "/repos/owner/current/environments/production/variables":
					return actionsResponse(r, 200, `{"total_count":0,"variables":[]}`), nil
				case "/repos/owner/current/environments/production/secrets":
					secretReads++
					if deleting && mode == "duplicate timestamp" {
						return actionsResponse(r, 200, `{"total_count":1,"secrets":[{"name":"SECRET_CANARY","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`), nil
					}
					if deleting && mode == "alias count" {
						return actionsResponse(r, 200, `{"total_count":0,"TOTAL_COUNT":1,"secrets":[{"name":"SECRET_CANARY","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`), nil
					}
					if deleting && mode == "contradictory next" {
						response := actionsResponse(r, 200, `{"total_count":1,"secrets":[{"name":"SECRET_CANARY","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`)
						response.Header.Set("Link", `<https://foreign.invalid/PRIVATE_CANARY>; rel="next"`)
						return response, nil
					}
					return actionsResponse(r, 200, `{"total_count":1,"secrets":[{"name":"SECRET_CANARY","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`), nil
				default:
					t.Fatalf("unexpected path %s", r.URL.Path)
					return nil, nil
				}
			}).rest}
			observation, err := c.ReadEnvironmentPrune(repo, nil)
			if err != nil {
				t.Fatal(err)
			}
			candidates, err := observation.Candidates()
			if err != nil || len(candidates) != 1 {
				t.Fatalf("invalid candidate count: %v", err)
			}
			deleting = true
			err = c.DeleteEnvironment(repo, candidates[0])
			var pre *environmentPreWriteFailure
			if err == nil || !errors.As(err, &pre) || writes != 0 || secretReads != 2 || strings.Contains(err.Error(), "SECRET_CANARY") {
				t.Fatalf("err=%v secret reads=%d writes=%d", err, secretReads, writes)
			}
		})
	}
}

func TestEnvironmentSecretMetadataLinkEvidence(t *testing.T) {
	for _, tc := range []struct {
		name  string
		links []string
		ok    bool
	}{
		{"no link", nil, true},
		{"empty link", []string{" "}, false},
		{"previous", []string{`<https://foreign.invalid/PRIVATE_CANARY>; rel="prev"`}, true},
		{"next", []string{`<https://foreign.invalid/PRIVATE_CANARY>; rel="next"`}, false},
		{"multiple headers", []string{`<https://foreign.invalid/PRIVATE_CANARY>; rel="prev"`, `<https://foreign.invalid/PRIVATE_CANARY>; rel="next"`}, false},
		{"malformed", []string{`<https://foreign.invalid/PRIVATE_CANARY>; rel="next`}, false},
		{"ambiguous relation", []string{`<https://foreign.invalid/PRIVATE_CANARY>; rel=prev; rel*=utf-8''next`}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != http.MethodGet || r.URL.Path != "/repos/owner/current/environments/prod/secrets" || r.URL.Query().Get("page") != "1" {
					t.Fatalf("unexpected request %s %s", r.Method, r.URL)
				}
				response := actionsResponse(r, 200, `{"total_count":0,"secrets":[]}`)
				for _, link := range tc.links {
					response.Header.Add("Link", link)
				}
				return response, nil
			}).rest}
			values, err := c.readEnvironmentSecretMetadata(repository.Repository{Owner: "owner", Name: "current"}, "prod")
			if (err == nil) != tc.ok || calls != 1 || !tc.ok && values != nil || err != nil && strings.Contains(err.Error(), "PRIVATE_CANARY") {
				t.Fatalf("err=%v calls=%d returned=%d", err, calls, len(values))
			}
		})
	}
}

func TestEnvironmentSecretMetadataPaginationDoesNotFollowLinkURL(t *testing.T) {
	calls := 0
	c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != http.MethodGet || r.URL.Path != "/repos/owner/current/environments/prod/secrets" || r.URL.Query().Get("page") != fmt.Sprint(calls) {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL)
		}
		if calls == 1 {
			entries := make([]string, 100)
			for i := range entries {
				entries[i] = fmt.Sprintf(`{"name":"KEY_%d","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`, i)
			}
			response := actionsResponse(r, 200, `{"total_count":101,"secrets":[`+strings.Join(entries, ",")+`]}`)
			response.Header.Set("Link", `<https://foreign.invalid/PRIVATE_CANARY?page=2>; rel="next", <https://foreign.invalid/PRIVATE_CANARY?page=2>; rel="last"`)
			return response, nil
		}
		if calls == 2 {
			response := actionsResponse(r, 200, `{"total_count":101,"secrets":[{"name":"KEY_100","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`)
			response.Header.Set("Link", `<https://foreign.invalid/PRIVATE_CANARY?page=1>; rel="prev", <https://foreign.invalid/PRIVATE_CANARY?page=1>; rel="first"`)
			return response, nil
		}
		t.Fatalf("unexpected page %d", calls)
		return nil, nil
	}).rest}
	values, err := c.readEnvironmentSecretMetadata(repository.Repository{Owner: "owner", Name: "current"}, "prod")
	if err != nil || calls != 2 || len(values) != 101 {
		t.Fatalf("err=%v calls=%d returned=%d", err, calls, len(values))
	}
}

func TestEnvironmentSecretMetadataRejectsMissingNextWhenCountIncomplete(t *testing.T) {
	calls := 0
	c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			entries := make([]string, 100)
			for i := range entries {
				entries[i] = fmt.Sprintf(`{"name":"KEY_%d","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`, i)
			}
			response := actionsResponse(r, 200, `{"total_count":101,"secrets":[`+strings.Join(entries, ",")+`]}`)
			response.Header.Set("Link", `<https://foreign.invalid/PRIVATE_CANARY>; rel="prev"`)
			return response, nil
		}
		return actionsResponse(r, 200, `{"total_count":101,"secrets":[{"name":"KEY_100","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`), nil
	}).rest}
	values, err := c.readEnvironmentSecretMetadata(repository.Repository{Owner: "owner", Name: "current"}, "prod")
	if err == nil || values != nil || calls != 1 || strings.Contains(err.Error(), "PRIVATE_CANARY") {
		t.Fatalf("err=%v calls=%d returned=%d", err, calls, len(values))
	}
}

// Valid JSON whitespace makes the size limit, rather than field validation,
// the only difference between these two otherwise identical responses.
func TestEnvironmentSecretMetadataPageSizeBoundary(t *testing.T) {
	const limit = 1 << 20
	const body = `{"total_count":0,"secrets":[]}`
	for _, size := range []int{limit, limit + 1} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			responseBody := body + strings.Repeat(" ", size-len(body))
			c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				return actionsResponse(r, http.StatusOK, responseBody), nil
			}).rest}
			values, err := c.readEnvironmentSecretMetadata(repository.Repository{Owner: "owner", Name: "current"}, "prod")
			if size == limit && (err != nil || values == nil || len(values) != 0) {
				t.Fatalf("at limit: err=%v values=%v", err, values)
			}
			if size > limit && (err == nil || values != nil) {
				t.Fatalf("above limit: err=%v values=%v", err, values)
			}
		})
	}
}
