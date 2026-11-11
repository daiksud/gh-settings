package github

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

const deploymentRulesetJSON = `{"id":7,"name":"deployments","source_type":"Repository","source":"owner/current","target":"branch","enforcement":"active","bypass_actors":[],"conditions":{"ref_name":{"include":["~DEFAULT_BRANCH"],"exclude":[]}},"rules":[{"type":"deletion"},{"type":"required_deployments","parameters":{"required_deployment_environments":["staging"]}}]}`

func TestDeploymentsStrictAcquisitionAndUIMapping(t *testing.T) {
	for _, target := range []string{"branch", "tag"} {
		t.Run(target, func(t *testing.T) { deploymentsStrictAcquisitionAndUIMapping(t, target) })
	}
}

func deploymentsStrictAcquisitionAndUIMapping(t *testing.T, target string) {
	policyJSON := deploymentTargetJSON(deploymentRulesetJSON, target)
	policy, err := decodeRulesetPolicyFromTest([]byte(policyJSON))
	if err != nil {
		t.Fatal(err)
	}
	ui, err := RulesetObservation(policy)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*ui.Rules.RequireDeploymentsToSucceed.RequiredDeploymentEnvironments, []string{"staging"}) {
		t.Fatal(ui)
	}
	desired, err := RulesetDeclaration(ui.Ruleset)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*(*desired.Rules)[1].Parameters.RequiredDeploymentEnvironments, []string{"staging"}) {
		t.Fatal(desired)
	}
	ui.Rules.RequireDeploymentsToSucceed = &uimodel.DeploymentsRule{}
	if _, err := RulesetDeclaration(ui.Ruleset); err == nil {
		t.Fatal("missing environments accepted")
	}
	for _, replacement := range []string{`{}`, `{"required_deployment_environments":null}`, `{"required_deployment_environments":[null]}`, `{"required_deployment_environments":[false]}`, `{"required_deployment_environments":["staging"],"future":true}`, `{"Required_deployment_environments":["staging"]}`, `{"required_deployment_environments":["staging","staging"]}`} {
		malformed := strings.Replace(policyJSON, `{"required_deployment_environments":["staging"]}`, replacement, 1)
		if _, err := decodeRulesetPolicyFromTest([]byte(malformed)); err == nil {
			t.Fatal("accepted incomplete/unknown protection", replacement)
		}
	}
}

func TestDeploymentsUpdateAndRemovalRequireIndependentReadback(t *testing.T) {
	for _, target := range []string{"branch", "tag"} {
		t.Run(target, func(t *testing.T) { deploymentsUpdateAndRemovalRequireIndependentReadback(t, target) })
	}
}

func deploymentsUpdateAndRemovalRequireIndependentReadback(t *testing.T, target string) {
	policyJSON := deploymentTargetJSON(deploymentRulesetJSON, target)
	for _, remove := range []bool{false, true} {
		for _, badReadback := range []bool{false, true} {
			t.Run(strings.Join([]string{map[bool]string{false: "update", true: "remove"}[remove], map[bool]string{false: "valid", true: "retained"}[badReadback]}, "/"), func(t *testing.T) {
				before, err := decodeRulesetPolicyFromTest([]byte(policyJSON))
				if err != nil {
					t.Fatal(err)
				}
				obs, err := settings.NewRulesetObservation([]settings.RulesetState{{ID: 7, Policy: before}}, 0)
				if err != nil {
					t.Fatal(err)
				}
				desired, _ := decodeRulesetPolicyFromTest([]byte(policyJSON))
				desired.ID = rulesetPtr(int64(7))
				if remove {
					desired.Rules = &[]settings.RulesetRule{{Type: "deletion"}}
				} else {
					(*desired.Rules)[1].Parameters.RequiredDeploymentEnvironments = rulesetPtr([]string{"production", "staging"})
				}
				declarations, err := settings.NewRulesetDeclarations([]settings.RulesetDeclaration{desired})
				if err != nil {
					t.Fatal(err)
				}
				changes, err := settings.ReconcileRulesets(obs, &declarations)
				if err != nil || len(changes) != 1 {
					t.Fatal(changes, err)
				}
				requests := []string{}
				client := rulesetMutationClient(t, changes[0], transportFunc(func(r *http.Request) (*http.Response, error) {
					requests = append(requests, r.Method)
					if r.URL.Path != "/repos/owner/current/rulesets/7" {
						t.Fatal(r.URL)
					}
					expected := `{"name":"deployments","target":"branch","enforcement":"active","bypass_actors":[],"conditions":{"ref_name":{"include":["~DEFAULT_BRANCH"],"exclude":[]}},"rules":[{"type":"deletion"},{"type":"required_deployments","parameters":{"required_deployment_environments":["production","staging"]}}]}`
					if remove {
						expected = strings.Replace(expected, `,{"type":"required_deployments","parameters":{"required_deployment_environments":["production","staging"]}}`, "", 1)
					}
					expected = deploymentTargetJSON(expected, target)
					if r.Method == "PUT" {
						var actual, want any
						json.NewDecoder(r.Body).Decode(&actual)
						json.Unmarshal([]byte(expected), &want)
						if !reflect.DeepEqual(actual, want) {
							t.Fatal("lost whole-object protection", actual, want)
						}
					} else if r.Method != "GET" {
						t.Fatal(r.Method)
					}
					if r.Method == "GET" && badReadback {
						return labelsResponse(r, 200, policyJSON), nil
					}
					return rulesetMutationResponse(t, r, 200, expected, 7), nil
				}))
				_, err = client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, changes[0])
				if (err != nil) != badReadback || !reflect.DeepEqual(requests, []string{"PUT", "GET"}) {
					t.Fatal(requests, err)
				}
				if !badReadback {
					after, _ := settings.NewRulesetObservation([]settings.RulesetState{{ID: 7, Policy: *changes[0].After()}}, 0)
					again, err := settings.ReconcileRulesets(after, &declarations)
					if err != nil || len(again) != 0 {
						t.Fatal("repeat differs", again, err)
					}
				}
			})
		}
	}
}

func deploymentTargetJSON(payload, target string) string {
	if target == "tag" {
		payload = strings.ReplaceAll(payload, `"target":"branch"`, `"target":"tag"`)
		payload = strings.ReplaceAll(payload, `"~DEFAULT_BRANCH"`, `"refs/tags/release/*"`)
		payload = strings.ReplaceAll(payload, `"enforcement":"active"`, `"enforcement":"disabled"`)
	}
	return payload
}
