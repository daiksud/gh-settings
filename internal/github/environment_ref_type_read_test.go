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
)

func environmentRefTypeReader(t *testing.T, policy, patterns string) *EnvironmentsClient {
	t.Helper()
	responses := map[string]string{
		"/repos/owner/current/environments":                                        `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`,
		"/repos/owner/current/environments/production":                             strings.Replace(emptyEnvironmentFixture, `"deployment_branch_policy":null`, `"deployment_branch_policy":`+policy, 1),
		"/repos/owner/current/environments/production/deployment_protection_rules": customEnvironmentEmpty,
		"/repos/owner/current/environments/production/deployment-branch-policies":  patterns,
	}
	return &EnvironmentsClient{
		rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
			if req.Method != "GET" {
				t.Fatal("pure reader wrote", req.Method)
			}
			body, ok := responses[req.URL.Path]
			if !ok {
				t.Fatal("unexpected read", req.URL)
			}
			return actionsResponse(req, 200, body), nil
		}).rest,
		NormalizationWarning:        func(string, settings.EnvironmentDeploymentNormalization) { t.Fatal("pure reader warned") },
		RefTypeNormalizationWarning: func(string, settings.EnvironmentPatternState) { t.Fatal("pure reader warned") },
	}
}

func TestEnvironmentRefTypeReaderPublishesOnlyCompletePagination(t *testing.T) {
	for _, latent := range []bool{false, true} {
		t.Run(strconv.FormatBool(latent), func(t *testing.T) {
			pages := 0
			client := &EnvironmentsClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
				if req.Method != "GET" {
					t.Fatal("pure reader wrote", req.Method)
				}
				switch req.URL.Path {
				case "/repos/owner/current/environments":
					return actionsResponse(req, 200, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`), nil
				case "/repos/owner/current/environments/production":
					policy := fmt.Sprintf(`{"protected_branches":%t,"custom_branch_policies":true}`, latent)
					return actionsResponse(req, 200, strings.Replace(emptyEnvironmentFixture, `"deployment_branch_policy":null`, `"deployment_branch_policy":`+policy, 1)), nil
				case "/repos/owner/current/environments/production/deployment_protection_rules":
					return actionsResponse(req, 200, customEnvironmentEmpty), nil
				case "/repos/owner/current/environments/production/deployment-branch-policies":
					pages++
					page, _ := strconv.Atoi(req.URL.Query().Get("page"))
					entries := []any{}
					for i := (page-1)*100 + 1; i <= 101 && i <= page*100; i++ {
						kind := "branch"
						if i == 1 {
							kind = "future"
						}
						if i == 101 {
							kind = "\t "
						}
						entries = append(entries, map[string]any{"id": i, "type": kind, "name": fmt.Sprintf("release-%03d", i)})
					}
					body, _ := json.Marshal(map[string]any{"total_count": 101, "branch_policies": entries})
					return actionsResponse(req, 200, string(body)), nil
				default:
					t.Fatal("unexpected read", req.URL)
					return nil, nil
				}
			}).rest,
				NormalizationWarning:        func(string, settings.EnvironmentDeploymentNormalization) { t.Fatal("pure reader warned") },
				RefTypeNormalizationWarning: func(string, settings.EnvironmentPatternState) { t.Fatal("pure reader warned") },
			}
			observed, err := client.Read(repository.Repository{Owner: "owner", Name: "current"}, []string{"production"})
			if err != nil {
				t.Fatal(err)
			}
			states, err := observed.Values()
			if err != nil || len(states) != 1 || len(states[0].Patterns) != 101 || pages != 2 {
				t.Fatal("incomplete acquisition published", states, pages, err)
			}
			wantFirst := settings.EnvironmentPatternState{ID: 1, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "release-001"}, RefTypeNormalization: settings.ObservationNormalization{Path: settings.EnvironmentRefTypePath, RawValue: "future", Fallback: "branch"}}
			wantLast := settings.EnvironmentPatternState{ID: 101, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "release-101"}, RefTypeNormalization: settings.ObservationNormalization{Path: settings.EnvironmentRefTypePath, RawValue: "\t ", Fallback: "branch"}}
			if states[0].Patterns[0] != wantFirst || states[0].Patterns[100] != wantLast {
				t.Fatal("cross-page raw evidence lost", states[0].Patterns)
			}
			policy, err := states[0].EffectivePolicy()
			if err != nil || latent && (policy.DeploymentBranchPolicy != nil || policy.DeploymentBranchPatterns != nil) || !latent && (policy.DeploymentBranchPatterns == nil || len(*policy.DeploymentBranchPatterns) != 101) {
				t.Fatal("active/latent projection boundary", policy, err)
			}
		})
	}
}

func TestEnvironmentRefTypeReaderPreservesCompleteRawAndEffectiveState(t *testing.T) {
	client := environmentRefTypeReader(t, `{"protected_branches":false,"custom_branch_policies":true}`, `{"total_count":3,"branch_policies":[{"id":7,"type":"future","name":"main"},{"id":8,"type":"tag","name":"main"},{"id":9,"type":"branch","name":"release"}]}`)
	observed, err := client.Read(repository.Repository{Owner: "owner", Name: "current"}, nil)
	if err != nil {
		t.Fatal("complete unsupported ref type rejected", err)
	}
	states, err := observed.Values()
	if err != nil || len(states) != 1 {
		t.Fatal(states, err)
	}
	want := []settings.EnvironmentPatternState{
		{ID: 7, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "main"}, RefTypeNormalization: settings.ObservationNormalization{Path: settings.EnvironmentRefTypePath, RawValue: "future", Fallback: "branch"}},
		{ID: 8, Pattern: settings.EnvironmentPattern{Type: "tag", Name: "main"}},
		{ID: 9, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "release"}},
	}
	if states[0].ID != 1 || states[0].Policy.Name != "production" || !reflect.DeepEqual(states[0].Patterns, want) || !reflect.DeepEqual(*states[0].Policy.DeploymentBranchPatterns, []settings.EnvironmentPattern{{Type: "branch", Name: "main"}, {Type: "branch", Name: "release"}, {Type: "tag", Name: "main"}}) {
		t.Fatal("acquired identities or effective projection lost", states)
	}
	raw, err := states[0].Patterns[0].RawPattern()
	if err != nil || raw != (settings.EnvironmentPattern{Type: "future", Name: "main"}) {
		t.Fatal("raw type replaced by fallback", raw, err)
	}
}

func TestEnvironmentRefTypeReaderRetainsCollidingRawIdentitiesAndLatents(t *testing.T) {
	for _, latent := range []bool{false, true} {
		t.Run(strconv.FormatBool(latent), func(t *testing.T) {
			policy := `{"protected_branches":false,"custom_branch_policies":true}`
			if latent {
				policy = `{"protected_branches":true,"custom_branch_policies":true}`
			}
			client := environmentRefTypeReader(t, policy, `{"total_count":2,"branch_policies":[{"id":7,"type":"future","name":"main"},{"id":8,"type":"branch","name":"main"}]}`)
			observed, err := client.Read(repository.Repository{Owner: "owner", Name: "current"}, nil)
			if err != nil {
				t.Fatal("complete distinct raw identities rejected", latent, err)
			}
			states, err := observed.Values()
			if err != nil || len(states) != 1 || len(states[0].Patterns) != 2 || states[0].Policy.DeploymentBranchPatterns != nil {
				t.Fatal("raw collision collapsed or materialized", latent, states, err)
			}
			want := []settings.EnvironmentPatternState{
				{ID: 7, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "main"}, RefTypeNormalization: settings.ObservationNormalization{Path: settings.EnvironmentRefTypePath, RawValue: "future", Fallback: "branch"}},
				{ID: 8, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "main"}},
			}
			if !reflect.DeepEqual(states[0].Patterns, want) {
				t.Fatal("raw acquisition changed", states[0].Patterns)
			}
			rawMode, err := states[0].RawDeploymentBranchPolicy()
			if err != nil || rawMode == nil || *rawMode != (settings.EnvironmentBranchPolicy{ProtectedBranches: latent, CustomBranchPolicies: true}) {
				t.Fatal("raw mode lost", rawMode, err)
			}
			effective, err := states[0].EffectivePolicy()
			if (err != nil) == latent || latent && (effective.DeploymentBranchPolicy != nil || effective.DeploymentBranchPatterns != nil) {
				t.Fatal("collision silently materialized or latent patterns activated", effective, err)
			}
		})
	}
}
