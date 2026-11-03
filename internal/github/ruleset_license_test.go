package github

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

const licenseUI = `version: daiksud/gh-settings@v0
repository:
  settings:
    rulesets:
      - id: 7
        ruleset_name: license
        enforcement_status: active
        target_branches: {include_default_branch: true}
        rules: {require_license_compliance_results_before_merging: true}
`
const licensePolicy = `{"name":"license","target":"branch","enforcement":"active","conditions":{"ref_name":{"include":["~DEFAULT_BRANCH"],"exclude":[]}},"bypass_actors":[],"rules":[{"type":"license_compliance_scanning"}]}`

func TestLicenseRulesetDeclarationAndStrictAcquisition(t *testing.T) {
	ui, err := workspace.DecodeConfig([]byte(licenseUI))
	if err != nil {
		t.Fatal(err)
	}
	declaration, err := DeclarationFromUI(ui)
	if err != nil {
		t.Fatal(err)
	}
	rules := *declaration.Rulesets.Values()[0].Rules
	if !reflect.DeepEqual(rules, []settings.RulesetRule{{Type: "license_compliance_scanning"}}) {
		t.Fatal(rules)
	}
	for _, bad := range []string{
		strings.Replace(licensePolicy, `"target":"branch"`, `"target":"tag"`, 1),
		strings.Replace(licensePolicy, `"type":"license_compliance_scanning"`, `"type":"license_compliance_scanning","parameters":{"future":true}`, 1),
		strings.Replace(licensePolicy, `"type":"license_compliance_scanning"`, `"type":"license_compliance_scanning","parameters":{"severity":"errors"}`, 1),
		strings.Replace(licensePolicy, `"type":"license_compliance_scanning"`, `"type":"license_compliance_scanning","parameters":null`, 1),
		strings.Replace(licensePolicy, `"bypass_actors":[],`, "", 1),
	} {
		if _, err := decodeRulesetPolicyFromTest([]byte(bad)); err == nil {
			t.Fatalf("accepted incomplete/unsupported policy: %s", bad)
		}
	}
	for _, bad := range []string{"null", "1", "'true'", "{}"} {
		if _, err := workspace.DecodeConfig([]byte(strings.Replace(licenseUI, "before_merging: true", "before_merging: "+bad, 1))); err == nil {
			t.Fatal("accepted invalid checkbox", bad)
		}
	}
}

func TestLicenseRulesetIndependentReadbackAndRemoval(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	for _, operation := range []string{"create", "update", "remove"} {
		removal := operation == "remove"
		for _, drift := range []bool{false, true} {
			beforeBody, afterBody := strings.Replace(licensePolicy, `[{"type":"license_compliance_scanning"}]`, `[]`, 1), licensePolicy
			if removal {
				beforeBody, afterBody = afterBody, beforeBody
			}
			before, err := decodeRulesetPolicyFromTest([]byte(beforeBody))
			if err != nil {
				t.Fatal(err)
			}
			after, err := decodeRulesetPolicyFromTest([]byte(afterBody))
			if err != nil {
				t.Fatal(err)
			}
			method, path, selectedID, kind := "PUT", "/repos/owner/current/rulesets/7", int64(7), "update"
			prior := &before
			if operation == "create" {
				method, path, selectedID, kind, prior = "POST", "/repos/owner/current/rulesets", 0, "create", nil
			}
			change, err := settings.NewRulesetChange(kind, selectedID, prior, &after)
			if err != nil {
				t.Fatal(err)
			}
			response := func(body string) string {
				return strings.Replace(body, `{"name":`, `{"id":7,"source":"owner/current","source_type":"Repository","name":`, 1)
			}
			calls := 0
			client := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					var got, want any
					if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal([]byte(afterBody), &want); err != nil {
						t.Fatal(err)
					}
					if req.Method != method || req.URL.Path != path || !reflect.DeepEqual(got, want) {
						t.Fatal(req.Method, req.URL, got, want)
					}
					return labelsResponse(req, 200, response(afterBody)), nil
				}
				if calls != 2 || req.Method != "GET" || req.URL.Path != "/repos/owner/current/rulesets/7" {
					t.Fatal(req.Method, req.URL)
				}
				body := afterBody
				if drift {
					body = beforeBody
				}
				return labelsResponse(req, 200, response(body)), nil
			})
			_, err = client.ApplyRuleset(repo, change)
			if (err != nil) != drift || calls != 2 {
				t.Fatal("wrong independent verification", removal, drift, calls, err)
			}
			if drift {
				continue
			}
			current, err := settings.NewRulesetObservation([]settings.RulesetState{{ID: 7, Policy: after}}, 0)
			if err != nil {
				t.Fatal(err)
			}
			exported, err := current.ExportDeclarations()
			if err != nil {
				t.Fatal(err)
			}
			ui, err := RulesetsObservation(current)
			if err != nil {
				t.Fatal(err)
			}
			if len(ui.Rulesets) != 1 || ui.Rulesets[0].Rules.RequireLicenseComplianceResultsBeforeMerging == nil || *ui.Rulesets[0].Rules.RequireLicenseComplianceResultsBeforeMerging == removal {
				t.Fatal(ui)
			}
			changes, err := settings.ReconcileRulesets(current, &exported)
			if err != nil || len(changes) != 0 {
				t.Fatal("repeat produced drift", changes, err)
			}
		}
	}
}
