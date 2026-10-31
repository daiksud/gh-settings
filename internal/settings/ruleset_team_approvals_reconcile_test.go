package settings

import (
	"fmt"
	"reflect"
	"testing"
)

func TestTeamApprovalsReconciliationKeepsSelectedIDBoundIntent(t *testing.T) {
	for _, companions := range []bool{false, true} {
		for _, mode := range []string{"equal", "different", "reviewer removed", "reviewer replaced", "reviewers omitted", "PR removed", "rules omitted"} {
			t.Run(fmt.Sprintf("%t/%s", companions, mode), func(t *testing.T) {
				state := normalizedTeamApprovalsState(companions)
				private := normalizedTeamApprovalsState(false)
				private.ID, private.Policy.ID, private.Policy.Name = 9, rsID(9), "PRIVATE_TEAM_POLICY"
				private.TeamApprovalsNormalizations[0].RawValue = 101
				after := cloneRuleset(state.Policy)
				reviewers := ruleOfForApprovals(after).RequiredReviewers
				switch mode {
				case "different":
					*(*reviewers)[2].MinimumApprovals = 10
				case "reviewer removed":
					*reviewers = (*reviewers)[:2]
				case "reviewer replaced":
					(*reviewers)[2].Reviewer.ID = 99
					*(*reviewers)[2].MinimumApprovals = 4
				case "reviewers omitted":
					ruleOfForApprovals(after).RequiredReviewers = nil
				case "PR removed":
					rules := []RulesetRule{}
					for _, r := range *after.Rules {
						if r.Type != "pull_request" {
							rules = append(rules, r)
						}
					}
					after.Rules = &rules
				case "rules omitted":
					after.Rules = nil
				}
				observed := rsObserved(t, private, state)
				changes, err := ReconcileRulesets(observed, rsDesired(t, after))
				if err != nil || len(changes) != 1 || changes[0].ID() != 7 || changes[0].Operation() != "update" {
					t.Fatal("selected intent lost", changes, err)
				}
				if !reflect.DeepEqual(changes[0].Normalizations(), teamApprovalsRecords(state)) {
					t.Fatal("record/companions lost", changes[0].Normalizations())
				}
				actual := rulesetRequiredReviewers(*changes[0].After())
				if mode == "PR removed" || mode == "rules omitted" {
					if actual != nil {
						t.Fatal("removal invented reviewers")
					}
				} else if mode == "reviewers omitted" {
					if actual == nil || len(*actual) != 0 {
						t.Fatal("omission retained reviewer", actual)
					}
				}
				for _, desired := range []*RulesetDeclarations{nil, rsDesired(t)} {
					if got, err := ReconcileRulesets(observed, desired); err != nil || len(got) != 0 {
						t.Fatal("unmanaged scope selected", got, err)
					}
				}
			})
		}
	}
}

func TestTeamApprovalsReconciliationRetainsCanonicalNoopAndStrictDesiredEntries(t *testing.T) {
	normalized := normalizedTeamApprovalsState(false)
	canonical := normalizedTeamApprovalsState(false)
	canonical.ID, canonical.Policy.ID, canonical.Policy.Name = 8, rsID(8), "canonical"
	canonical.TeamApprovalsNormalizations = nil
	if got, err := ReconcileRulesets(rsObserved(t, normalized, canonical), rsDesired(t, canonical.Policy)); err != nil || len(got) != 0 {
		t.Fatal("canonical/unlisted became actionable", got, err)
	}
	for _, field := range []string{"count", "patterns"} {
		desired := cloneRuleset(normalized.Policy)
		reviewers := ruleOfForApprovals(desired).RequiredReviewers
		if field == "count" {
			(*reviewers)[0].MinimumApprovals = nil
		} else {
			(*reviewers)[0].FilePatterns = nil
		}
		if _, err := NewRulesetDeclarations([]RulesetDeclaration{desired}); err == nil {
			t.Fatal("omitted required team entry was defaulted", field)
		}
	}
}
