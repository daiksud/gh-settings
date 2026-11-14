package github

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
)

// A duplicate count or list must not make enabled custom protection appear
// empty at either pre-delete parent read, even if the last value is empty.
func TestEnvironmentPruneRejectsAmbiguousCustomProtectionBeforeDelete(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"duplicate count", `{"total_count":1,"total_count":0,"custom_deployment_protection_rules":[]}`},
		{"duplicate list", `{"total_count":0,"custom_deployment_protection_rules":[{"id":9}],"custom_deployment_protection_rules":[]}`},
	} {
		for _, corruptRead := range []int{1, 2} {
			t.Run(fmt.Sprintf("%s/read_%d", tc.name, corruptRead), func(t *testing.T) {
				phase := "acquire"
				customReads, writes := 0, 0
				c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
					if r.Method == http.MethodDelete {
						writes++
						return actionsResponse(r, http.StatusNoContent, ""), nil
					}
					if r.Method != http.MethodGet {
						t.Fatalf("unexpected method %s", r.Method)
					}
					switch r.URL.Path {
					case "/repos/owner/current":
						return actionsResponse(r, 200, `{"id":71,"full_name":"owner/current","archived":false}`), nil
					case "/repos/owner/current/environments":
						if writes != 0 {
							return actionsResponse(r, 200, `{"total_count":0,"environments":[]}`), nil
						}
						return actionsResponse(r, 200, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`), nil
					case "/repos/owner/current/environments/production":
						return actionsResponse(r, 200, emptyEnvironmentFixture), nil
					case "/repos/owner/current/environments/production/deployment_protection_rules":
						if phase == "delete" {
							customReads++
							if customReads == corruptRead {
								return actionsResponse(r, 200, tc.body), nil
							}
						}
						return actionsResponse(r, 200, customEnvironmentEmpty), nil
					case "/repos/owner/current/environments/production/variables":
						return actionsResponse(r, 200, `{"total_count":0,"variables":[]}`), nil
					case "/repos/owner/current/environments/production/secrets":
						return actionsResponse(r, 200, `{"total_count":0,"secrets":[]}`), nil
					default:
						t.Fatalf("unexpected request %s", r.URL)
						return nil, nil
					}
				}).rest}
				repo := repository.Repository{Owner: "owner", Name: "current"}
				observed, err := c.ReadEnvironmentPrune(repo, nil)
				if err != nil {
					t.Fatalf("acquire clean candidate: %v", err)
				}
				candidates, err := observed.Candidates()
				if err != nil || len(candidates) != 1 {
					t.Fatalf("candidate count=%d err=%v", len(candidates), err)
				}
				phase = "delete"
				err = c.DeleteEnvironment(repo, candidates[0])
				var prewrite interface{ ReadFailedBeforeWrite() }
				if err == nil || !errors.As(err, &prewrite) || customReads != corruptRead || writes != 0 {
					t.Fatalf("err=%v custom reads=%d DELETEs=%d", err, customReads, writes)
				}
			})
		}
	}
}
