package github

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
)

// Integration-only fixture: both child collections contain private evidence.
func TestMixedEnvironmentPruneChildEvidence(t *testing.T) {
	for _, mode := range []string{"complete", "ambiguous variables", "ambiguous secrets", "ambiguous variables acquisition", "ambiguous secrets acquisition"} {
		t.Run(mode, func(t *testing.T) {
			fresh, deleted := false, false
			variableReads, secretReads, deletes, absenceReads := 0, 0, 0, 0
			repo := repository.Repository{Owner: "owner", Name: "current"}
			client := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodDelete {
					if r.URL.Path != "/repos/owner/current/environments/production" {
						t.Fatalf("unexpected DELETE path %s", r.URL.Path)
					}
					deletes++
					deleted = true
					return actionsResponse(r, 204, ""), nil
				}
				if r.Method != http.MethodGet {
					t.Fatalf("unexpected method %s", r.Method)
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					return actionsResponse(r, 200, `{"id":71,"full_name":"owner/current","archived":false}`), nil
				case "/repos/owner/current/environments":
					if deleted {
						absenceReads++
						return actionsResponse(r, 200, `{"total_count":0,"environments":[]}`), nil
					}
					return actionsResponse(r, 200, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`), nil
				case "/repos/owner/current/environments/production":
					return actionsResponse(r, 200, emptyEnvironmentFixture), nil
				case "/repos/owner/current/environments/production/deployment_protection_rules":
					return actionsResponse(r, 200, customEnvironmentEmpty), nil
				case "/repos/owner/current/environments/production/variables":
					variableReads++
					if fresh && mode == "ambiguous variables" || !fresh && mode == "ambiguous variables acquisition" {
						return actionsResponse(r, 200, `{"total_count":1,"total_count":0,"variables":[{"name":"KEY","value":"PRIVATE_VARIABLE_CANARY"}]}`), nil
					}
					return actionsResponse(r, 200, `{"total_count":1,"variables":[{"name":"KEY","value":"PRIVATE_VARIABLE_CANARY"}]}`), nil
				case "/repos/owner/current/environments/production/secrets":
					secretReads++
					if fresh && mode == "ambiguous secrets" || !fresh && mode == "ambiguous secrets acquisition" {
						return actionsResponse(r, 200, `{"total_count":1,"secrets":[{"name":"PRIVATE_SECRET_CANARY","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`), nil
					}
					return actionsResponse(r, 200, `{"total_count":1,"secrets":[{"name":"PRIVATE_SECRET_CANARY","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`), nil
				default:
					t.Fatalf("unexpected path %s", r.URL.Path)
					return nil, nil
				}
			}).rest}
			observation, err := client.ReadEnvironmentPrune(repo, nil)
			if strings.HasSuffix(mode, "acquisition") {
				wantSecrets := 1
				if mode == "ambiguous variables acquisition" {
					wantSecrets = 0
				}
				if err == nil || deletes != 0 || variableReads != 1 || secretReads != wantSecrets || strings.Contains(err.Error(), "PRIVATE_VARIABLE_CANARY") || strings.Contains(err.Error(), "PRIVATE_SECRET_CANARY") {
					t.Fatalf("err=%v deletes=%d variable reads=%d secret reads=%d", err, deletes, variableReads, secretReads)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			candidates, err := observation.Candidates()
			if err != nil || len(candidates) != 1 || variableReads != 1 || secretReads != 1 {
				t.Fatalf("candidates=%d err=%v variable reads=%d secret reads=%d", len(candidates), err, variableReads, secretReads)
			}
			fresh = true
			err = client.DeleteEnvironment(repo, candidates[0])
			if mode == "complete" {
				if err != nil || deletes != 1 || absenceReads != 1 || variableReads != 2 || secretReads != 2 {
					t.Fatalf("err=%v deletes=%d absence reads=%d variable reads=%d secret reads=%d", err, deletes, absenceReads, variableReads, secretReads)
				}
				return
			}
			var pre *environmentPreWriteFailure
			wantSecrets := 2
			if mode == "ambiguous variables" {
				wantSecrets = 1
			}
			if err == nil || !errors.As(err, &pre) || deletes != 0 || variableReads != 2 || secretReads != wantSecrets || strings.Contains(err.Error(), "PRIVATE_VARIABLE_CANARY") || strings.Contains(err.Error(), "PRIVATE_SECRET_CANARY") {
				t.Fatalf("err=%v deletes=%d variable reads=%d secret reads=%d", err, deletes, variableReads, secretReads)
			}
		})
	}
}
