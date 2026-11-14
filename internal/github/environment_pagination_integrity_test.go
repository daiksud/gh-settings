package github

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
)

func TestEnvironmentInventoryAndPatternPaginationEvidence(t *testing.T) {
	for _, family := range []string{"environments", "patterns"} {
		for _, scenario := range []string{"empty next", "oversized page", "nonfinal prev only", "invalid link", "valid absent link", "valid next link"} {
			t.Run(family+"/"+scenario, func(t *testing.T) {
				calls := 0
				c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
					calls++
					path := "/repos/owner/current/environments"
					if family == "patterns" {
						path += "/production/deployment-branch-policies"
					}
					if r.Method != http.MethodGet || r.URL.Path != path || r.URL.Query().Get("page") != fmt.Sprint(calls) {
						t.Fatalf("unexpected request %s %s", r.Method, r.URL)
					}
					key := "environments"
					if family == "patterns" {
						key = "branch_policies"
					}
					entries := make([]string, 0, 101)
					size, total := 0, 0
					switch scenario {
					case "oversized page":
						size, total = 101, 101
					case "nonfinal prev only", "valid absent link", "valid next link":
						if calls == 1 {
							size, total = 100, 101
						} else {
							size, total = 1, 101
						}
					}
					for i := 0; i < size; i++ {
						id := i + 1
						if calls == 2 {
							id = 101
						}
						if family == "patterns" {
							entries = append(entries, fmt.Sprintf(`{"id":%d,"name":"branch-%d","type":"branch"}`, id, id))
						} else {
							entries = append(entries, fmt.Sprintf(`{"id":%d,"name":"env-%d"}`, id, id))
						}
					}
					body := fmt.Sprintf(`{"total_count":%d,%q:[%s]}`, total, key, strings.Join(entries, ","))
					response := actionsResponse(r, http.StatusOK, body)
					switch scenario {
					case "empty next":
						response.Header.Set("Link", `<https://foreign.invalid/ignored>; rel="next"`)
					case "nonfinal prev only":
						response.Header.Set("Link", `<https://foreign.invalid/ignored>; rel="prev"`)
					case "invalid link":
						response.Header.Set("Link", `<https://foreign.invalid/ignored>; rel="next`)
					case "valid next link":
						if calls == 1 {
							response.Header.Set("Link", `<https://foreign.invalid/ignored>; rel="next"`)
						} else {
							response.Header.Set("Link", `<https://foreign.invalid/ignored>; rel="prev"`)
						}
					}
					return response, nil
				}).rest}
				repo := repository.Repository{Owner: "owner", Name: "current"}
				var err error
				if family == "patterns" {
					_, err = c.readPatterns(repo, "production")
				} else {
					_, err = c.inventory(repo)
				}
				want := scenario == "valid absent link" || scenario == "valid next link"
				if (err == nil) != want || want && calls != 2 || !want && calls != 1 {
					t.Fatalf("err=%v calls=%d", err, calls)
				}
			})
		}
	}
}

func TestEnvironmentPruneRejectsContradictoryTerminalLink(t *testing.T) {
	writes, inventoryReads := 0, 0
	c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet {
			writes++
			return actionsResponse(r, http.StatusNoContent, ""), nil
		}
		switch r.URL.Path {
		case "/repos/owner/current":
			return actionsResponse(r, 200, `{"id":71,"full_name":"owner/current","archived":false}`), nil
		case "/repos/owner/current/environments":
			inventoryReads++
			resp := actionsResponse(r, 200, `{"total_count":0,"environments":[]}`)
			resp.Header.Set("Link", `<https://foreign.invalid/ignored>; rel="next"`)
			return resp, nil
		default:
			t.Fatalf("unexpected read %s", r.URL)
			return nil, nil
		}
	}).rest}
	_, err := c.ReadEnvironmentPrune(repository.Repository{Owner: "owner", Name: "current"}, nil)
	if err == nil || writes != 0 || inventoryReads != 1 {
		t.Fatalf("err=%v inventory=%d writes=%d", err, inventoryReads, writes)
	}
}
