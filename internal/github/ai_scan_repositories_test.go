package github

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

const repositoryAIScanTarget = `{"id":1,"full_name":"owner/current","visibility":"public","owner":{"id":2,"type":"User"}}`

func aiRepositoryResponse(status int, body string) *http.Response {
	return &http.Response{Request: &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: "api.github.com", Path: "/repos/owner/current"}}, StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}
}

func TestAIScanRepositoriesRequiresAcquiredState(t *testing.T) {
	for _, body := range []string{`{}`, `null`, `{"pr_scan":"enabled"}`, `{"pr_scan":"enabled","repository_scan":null}`, `{"pr_scan":"enabled","repository_scan":false}`, `{"pr_scan":"enabled","repository_scan":"unknown"}`, `{"pr_scan":"disabled","repository_scan":"disabled"} {}`} {
		c := &AIScanRepositoriesClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
			if r.URL.Path == "/repos/owner/current" {
				return aiRepositoryResponse(200, repositoryAIScanTarget), nil
			}
			return aiRepositoryResponse(200, body), nil
		}).rest}
		if _, err := c.Read(repository.Repository{Owner: "owner", Name: "current"}); err == nil {
			t.Fatal("unknown became acquired", body)
		}
	}
	for _, target := range []string{`{}`, strings.Replace(repositoryAIScanTarget, `"User"`, `"Organization"`, 1), strings.Replace(repositoryAIScanTarget, `"public"`, `"private"`, 1), strings.Replace(repositoryAIScanTarget, `"id":1`, `"id":0`, 1), strings.Replace(repositoryAIScanTarget, `owner/current`, `owner/replaced`, 1)} {
		calls := 0
		c := &AIScanRepositoriesClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) { calls++; return aiRepositoryResponse(200, target), nil }).rest}
		if _, err := c.Read(repository.Repository{Owner: "owner", Name: "current"}); err == nil || calls != 1 {
			t.Fatal(target, err, calls)
		}
	}
}

func TestAIScanRepositoriesConditionalWriteAndVerification(t *testing.T) {
	for _, mode := range []string{"enable", "disable", "already matching", "prewrite denied", "write denied", "write network", "wrong ACK", "ACK companion drift", "ACK incomplete", "readback mismatch", "readback denied", "readback companion drift", "target identity drift", "owner identity drift"} {
		t.Run(mode, func(t *testing.T) {
			after := mode != "disable"
			current := !after
			if mode == "already matching" {
				current = after
			}
			writes := 0
			c := &AIScanRepositoriesClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/repos/owner/current" {
					if mode == "prewrite denied" {
						return aiRepositoryResponse(403, `{"message":"private-canary"}`), nil
					}
					target := repositoryAIScanTarget
					if writes > 0 && mode == "target identity drift" {
						target = strings.Replace(target, `"id":1`, `"id":9`, 1)
					}
					if writes > 0 && mode == "owner identity drift" {
						target = strings.Replace(target, `"id":2`, `"id":9`, 1)
					}
					return aiRepositoryResponse(200, target), nil
				}
				if r.URL.Path != "/repos/owner/current/code-scanning/ai-scan" {
					t.Fatal("unexpected endpoint", r.URL)
				}
				state := "disabled"
				if current {
					state = "enabled"
				}
				if r.Method == "GET" {
					if writes > 0 && mode == "readback denied" {
						return aiRepositoryResponse(404, `{}`), nil
					}
					pr := "enabled"
					if writes > 0 && mode == "readback companion drift" {
						pr = "disabled"
					}
					return aiRepositoryResponse(200, fmt.Sprintf(`{"repository_scan":%q,"pr_scan":%q}`, state, pr)), nil
				}
				if r.Method != "PATCH" {
					t.Fatal(r.Method)
				}
				writes++
				body, _ := io.ReadAll(r.Body)
				wanted := "disabled"
				if after {
					wanted = "enabled"
				}
				if string(body) != fmt.Sprintf(`{"repository_scan":%q}`, wanted) {
					t.Fatal("wrong payload", string(body))
				}
				if mode == "write network" {
					return nil, errors.New("private-canary")
				}
				if mode == "write denied" {
					return aiRepositoryResponse(403, `{"message":"private-canary"}`), nil
				}
				if mode != "readback mismatch" {
					current = after
				}
				if mode == "wrong ACK" {
					return aiRepositoryResponse(200, fmt.Sprintf(`{"repository_scan":%q,"pr_scan":"enabled"}`, state)), nil
				}
				if mode == "ACK companion drift" {
					return aiRepositoryResponse(200, fmt.Sprintf(`{"repository_scan":%q,"pr_scan":"disabled"}`, wanted)), nil
				}
				if mode == "ACK incomplete" {
					return aiRepositoryResponse(200, `{"pr_scan":"enabled"}`), nil
				}
				return aiRepositoryResponse(200, fmt.Sprintf(`{"repository_scan":%q,"pr_scan":"enabled"}`, wanted)), nil
			}).rest}
			change, _ := settings.NewSecurityChange("security.ai_scan_repositories", !after, after)
			wrote, err := c.Update(repository.Repository{Owner: "owner", Name: "current"}, change)
			success := mode == "enable" || mode == "disable" || mode == "already matching"
			if (err == nil) != success || wrote != (success && mode != "already matching") {
				t.Fatal(wrote, err)
			}
			wantWrites := 1
			if mode == "already matching" || mode == "prewrite denied" {
				wantWrites = 0
			}
			if writes != wantWrites {
				t.Fatal("unexpected request count", writes)
			}
			if err != nil {
				var pre interface{ ReadFailedBeforeWrite() }
				if errors.As(err, &pre) != (mode == "prewrite denied") || strings.Contains(err.Error(), "private-canary") {
					t.Fatal("wrong failure evidence", err)
				}
			}
		})
	}
}

func TestAIScanRepositoriesObservationIndependentOfPRCompanion(t *testing.T) {
	for _, body := range []string{`{"repository_scan":"enabled"}`, `{"repository_scan":"enabled","pr_scan":null}`, `{"repository_scan":"enabled","pr_scan":"future"}`} {
		c := &AIScanRepositoriesClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
			if r.Method != "GET" {
				t.Fatal("unavailable companion allowed mutation")
			}
			if r.URL.Path == "/repos/owner/current" {
				return aiRepositoryResponse(200, repositoryAIScanTarget), nil
			}
			return aiRepositoryResponse(200, body), nil
		}).rest}
		repo := repository.Repository{Owner: "owner", Name: "current"}
		observed, err := c.Read(repo)
		value, valueErr := observed.Value()
		if err != nil || valueErr != nil || !value {
			t.Fatal("independent native status lost", observed, err, valueErr)
		}
		change, _ := settings.NewSecurityChange("security.ai_scan_repositories", true, false)
		if _, err := c.Update(repo, change); err == nil {
			t.Fatal("unverified companion allowed update")
		}
	}
}
