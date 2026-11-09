package settings

import (
	"encoding/json"
	"testing"
)

// A complete known parameter belonging to another supported rule must still
// fail. A global JSON-key allowlist alone cannot establish rule ownership.
func TestRulesetParameterFamiliesRemainIsolated(t *testing.T) {
	fixtures := []struct{ kind, parameters string }{
		{"creation", `{}`},
		{"file_path_restriction", `{"restricted_file_paths":[],"ignored_file_paths":[]}`},
		{"max_file_path_length", `{"max_file_path_length":240}`},
		{"file_extension_restriction", `{"restricted_file_extensions":[]}`},
		{"max_file_size", `{"max_file_size":20,"ignored_file_paths":[]}`},
		{"deletion", `{}`},
		{"update", `{"update_allows_fetch_and_merge":false}`},
		{"required_deployments", `{"required_deployment_environments":[]}`},
		{"copilot_code_review", `{"review_draft_pull_requests":false,"review_on_push":false}`},
		{"code_coverage", `{"minimum_coverage":0,"max_coverage_drop":0}`},
		{"pull_request", `{"allowed_merge_methods":["squash"],"dismiss_stale_reviews_on_push":false,"require_code_owner_review":false,"require_extra_approval_for_unattributed_changes":false,"require_last_push_approval":false,"required_approving_review_count":0,"required_review_thread_resolution":false,"required_reviewers":[]}`},
		{"required_status_checks", `{"required_status_checks":[],"strict_required_status_checks_policy":false,"do_not_enforce_on_create":false}`},
		{"code_scanning", `{"code_scanning_tools":[]}`},
		{"code_quality", `{"severity":"errors"}`},
	}
	for _, receiver := range fixtures {
		if isPushRule(receiver.kind) || receiver.kind == "code_quality" || receiver.kind == "code_coverage" {
			continue
		}
		var own map[string]json.RawMessage
		if err := json.Unmarshal([]byte(receiver.parameters), &own); err != nil {
			t.Fatal(err)
		}
		decode := func(parameters map[string]json.RawMessage) RulesetRule {
			t.Helper()
			data, err := json.Marshal(parameters)
			if err != nil {
				t.Fatal(err)
			}
			var p RuleParameters
			if err := json.Unmarshal(data, &p); err != nil {
				t.Fatal(err)
			}
			return RulesetRule{Type: receiver.kind, Parameters: &p}
		}
		valid := decode(own)
		if err := validateRule(valid, true); err != nil {
			t.Fatalf("invalid baseline %s: %v", receiver.kind, err)
		}
		for _, donor := range fixtures {
			if donor.kind == receiver.kind {
				continue
			}
			var foreign map[string]json.RawMessage
			if err := json.Unmarshal([]byte(donor.parameters), &foreign); err != nil {
				t.Fatal(err)
			}
			for key, value := range foreign {
				if _, shared := own[key]; shared {
					continue
				}
				t.Run(receiver.kind+"/reject_"+key, func(t *testing.T) {
					mixed := make(map[string]json.RawMessage, len(own)+1)
					for k, v := range own {
						mixed[k] = v
					}
					mixed[key] = value
					rule := decode(mixed)
					if err := validateRule(rule, true); err == nil {
						t.Fatal("accepted foreign complete parameter", donor.kind, key)
					}
					policy := basicPolicy("isolated")
					policy.Rules = &[]RulesetRule{rule}
					if _, err := NewRulesetDeclarations([]RulesetDeclaration{policy}); err == nil {
						t.Fatal("declaration cloning/defaults discarded foreign parameter", donor.kind, key)
					}
				})
			}
		}
	}
}
