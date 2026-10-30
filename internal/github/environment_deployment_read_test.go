package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestEnvironmentDeploymentReaderCompletesRawCustomPatternsBeforePublishing(t *testing.T) {
	for _, policy := range []string{`null`, `{"protected_branches":true,"custom_branch_policies":false}`, `{"protected_branches":false,"custom_branch_policies":true}`, `{"protected_branches":false,"custom_branch_policies":false}`, `{"protected_branches":true,"custom_branch_policies":true}`} {
		for _, count := range []int{0, 101} {
			t.Run(policy+strconv.Itoa(count), func(t *testing.T) {
				patternCalls, calls := 0, 0
				custom := strings.Contains(policy, `"custom_branch_policies":true`)
				client := &EnvironmentsClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
					calls++
					if req.Method != "GET" {
						t.Fatal("pure read wrote", req.Method)
					}
					switch req.URL.Path {
					case "/repos/owner/current/environments":
						return actionsResponse(req, 200, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`), nil
					case "/repos/owner/current/environments/production":
						return actionsResponse(req, 200, strings.Replace(emptyEnvironmentFixture, `"deployment_branch_policy":null`, `"deployment_branch_policy":`+policy, 1)), nil
					case "/repos/owner/current/environments/production/deployment_protection_rules":
						return actionsResponse(req, 200, customEnvironmentEmpty), nil
					case "/repos/owner/current/environments/production/deployment-branch-policies":
						if !custom {
							t.Fatal("pattern applicability inferred from a non-custom raw mode")
						}
						patternCalls++
						page, _ := strconv.Atoi(req.URL.Query().Get("page"))
						entries := []any{}
						for i := (page-1)*100 + 1; i <= count && i <= page*100; i++ {
							entries = append(entries, map[string]any{"id": i, "type": "branch", "name": fmt.Sprintf("release-%d", i)})
						}
						body, _ := json.Marshal(map[string]any{"total_count": count, "branch_policies": entries})
						return actionsResponse(req, 200, string(body)), nil
					default:
						t.Fatal(req.URL)
						return nil, nil
					}
				}).rest, NormalizationWarning: func(string, settings.EnvironmentDeploymentNormalization) { t.Fatal("pure reader warned") }}
				observed, err := client.Read(repository.Repository{Owner: "owner", Name: "current"}, nil)
				if err != nil {
					t.Fatal(err)
				}
				values, err := observed.Values()
				if err != nil || len(values) != 1 {
					t.Fatal(values, err)
				}
				state := values[0]
				wantPatternCalls := 0
				if custom {
					wantPatternCalls = 1
					if count == 101 {
						wantPatternCalls = 2
					}
				}
				if patternCalls != wantPatternCalls || calls != 3+wantPatternCalls || state.ID != 1 || state.Policy.Name != "production" {
					t.Fatal("complete request boundary", calls, patternCalls, state)
				}
				var rawMode *settings.EnvironmentBranchPolicy
				json.Unmarshal([]byte(policy), &rawMode)
				got, err := state.RawDeploymentBranchPolicy()
				if err != nil || !reflect.DeepEqual(got, rawMode) {
					t.Fatal("raw pair/null lost", got, rawMode, err)
				}
				normalized := policy == `{"protected_branches":false,"custom_branch_policies":false}` || policy == `{"protected_branches":true,"custom_branch_policies":true}`
				wantRecord := settings.EnvironmentDeploymentNormalization{}
				if normalized {
					wantRecord = settings.EnvironmentDeploymentNormalization{Path: settings.EnvironmentDeploymentModePath, RawPolicy: *rawMode, Fallback: uimodel.NoRestriction}
				}
				if state.DeploymentModeNormalization != wantRecord || (custom && (state.Patterns == nil || len(state.Patterns) != count)) || (!custom && state.Patterns != nil) {
					t.Fatal("latent or per-resource provenance lost", state)
				}
				if normalized {
					if state.Policy.DeploymentBranchPolicy != nil || state.Policy.DeploymentBranchPatterns != nil || !state.Policy.DeploymentBranchPolicyDeclared {
						t.Fatal("latent patterns replayed as active policy", state)
					}
				} else if custom && (state.Policy.DeploymentBranchPatterns == nil || len(*state.Policy.DeploymentBranchPatterns) != count) {
					t.Fatal("canonical active patterns lost", state)
				}
			})
		}
	}
}

func TestEnvironmentDeploymentReaderDropsPartialResultsOnLaterAcquisitionFailures(t *testing.T) {
	for _, mode := range []string{"pattern-denied", "pattern-null", "pattern-partial", "page-denied", "page-count", "page-duplicate", "page-type", "custom-denied", "custom-active", "custom-incomplete", "wrong-name", "missing-pair", "later-detail"} {
		t.Run(mode, func(t *testing.T) {
			pages, detailReads := 0, 0
			client := &EnvironmentsClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
				if req.Method != "GET" {
					t.Fatal("failure read wrote", req.Method)
				}
				switch req.URL.Path {
				case "/repos/owner/current/environments":
					return actionsResponse(req, 200, `{"total_count":2,"environments":[{"id":1,"name":"production"},{"id":2,"name":"staging"}]}`), nil
				case "/repos/owner/current/environments/production", "/repos/owner/current/environments/staging":
					detailReads++
					var body map[string]any
					json.Unmarshal([]byte(emptyEnvironmentFixture), &body)
					body["deployment_branch_policy"] = map[string]any{"protected_branches": true, "custom_branch_policies": true}
					if strings.HasSuffix(req.URL.Path, "/staging") {
						body["id"], body["name"] = 2, "staging"
						if mode == "later-detail" {
							delete(body, "can_admins_bypass")
						}
					}
					if mode == "wrong-name" {
						body["name"] = "other"
					}
					if mode == "missing-pair" {
						body["deployment_branch_policy"] = map[string]any{"protected_branches": true}
					}
					data, _ := json.Marshal(body)
					return actionsResponse(req, 200, string(data)), nil
				case "/repos/owner/current/environments/production/deployment_protection_rules":
					if mode == "custom-denied" {
						return actionsResponse(req, 403, `{}`), nil
					}
					if mode == "custom-active" {
						return actionsResponse(req, 200, `{"total_count":1,"custom_deployment_protection_rules":[{}]}`), nil
					}
					if mode == "custom-incomplete" {
						return actionsResponse(req, 200, `{"custom_deployment_protection_rules":[]}`), nil
					}
					return actionsResponse(req, 200, customEnvironmentEmpty), nil
				case "/repos/owner/current/environments/production/deployment-branch-policies":
					pages++
					page, _ := strconv.Atoi(req.URL.Query().Get("page"))
					if mode == "pattern-denied" || mode == "page-denied" && page == 2 {
						return actionsResponse(req, 403, `{}`), nil
					}
					if mode == "pattern-null" {
						return actionsResponse(req, 200, `{"total_count":101,"branch_policies":null}`), nil
					}
					entries := []any{}
					for i := (page-1)*100 + 1; i <= 101 && i <= page*100; i++ {
						entries = append(entries, map[string]any{"id": i, "type": "branch", "name": fmt.Sprintf("release-%d", i)})
					}
					total := 101
					if mode == "pattern-partial" {
						entries = entries[:1]
					}
					if page == 2 {
						if mode == "page-count" {
							total = 102
						}
						if mode == "page-duplicate" {
							entries[0].(map[string]any)["id"] = 1
						}
						if mode == "page-type" {
							entries[0].(map[string]any)["type"] = false
						}
					}
					data, _ := json.Marshal(map[string]any{"total_count": total, "branch_policies": entries})
					return actionsResponse(req, 200, string(data)), nil
				default:
					t.Fatal("unexpected failure continuation", req.URL)
					return nil, nil
				}
			}).rest, NormalizationWarning: func(string, settings.EnvironmentDeploymentNormalization) { t.Fatal("failed acquisition warned") }}
			observed, err := client.Read(repository.Repository{Owner: "owner", Name: "current"}, nil)
			expectedPages := 2
			if mode == "pattern-denied" || mode == "pattern-null" || mode == "pattern-partial" {
				expectedPages = 1
			}
			if strings.HasPrefix(mode, "custom-") || mode == "wrong-name" || mode == "missing-pair" {
				expectedPages = 0
			}
			expectedDetails := 1
			if mode == "later-detail" {
				expectedDetails = 2
			}
			if err == nil || !reflect.DeepEqual(observed, settings.EnvironmentObservation{}) || pages != expectedPages || detailReads != expectedDetails {
				t.Fatal("incomplete collection/pagination boundary", observed, pages, expectedPages, detailReads, err)
			}
		})
	}
}
