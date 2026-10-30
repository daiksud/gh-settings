package settings

import (
	"reflect"
	"testing"
)

func teamApprovalsRecords(state RulesetState) RulesetNormalizations {
	records := mergeMethodRecords(state)
	records.TeamApprovals = state.TeamApprovalsNormalizations
	return records
}

func TestTeamApprovalsChangesOwnPerReviewerEvidenceForChoicesAndRemoval(t *testing.T) {
	for _, companions := range []bool{false, true} {
		for _, mode := range []string{"equal", "different", "reviewer removed", "reviewer replaced", "PR removed"} {
			state := normalizedTeamApprovalsState(companions)
			before, after := cloneRuleset(state.Policy), cloneRuleset(state.Policy)
			reviewers := ruleOfForApprovals(after).RequiredReviewers
			switch mode {
			case "different":
				*(*reviewers)[2].MinimumApprovals = 10
			case "reviewer removed":
				*reviewers = (*reviewers)[:2]
			case "reviewer replaced":
				(*reviewers)[2].Reviewer.ID = 99
				*(*reviewers)[2].MinimumApprovals = 4
			case "PR removed":
				rules := []RulesetRule{}
				for _, r := range *after.Rules {
					if r.Type != "pull_request" {
						rules = append(rules, r)
					}
				}
				after.Rules = &rules
			}
			records := teamApprovalsRecords(state)
			want := teamApprovalsRecords(normalizedTeamApprovalsState(companions))
			change, err := NewRulesetChangeWithNormalizations("update", 7, &before, &after, records)
			if err != nil || change.ID() != 7 || !reflect.DeepEqual(change.Normalizations(), want) {
				t.Fatal(mode, change, err)
			}
			display, err := NewRulesetDisplayChangeWithNormalizations("update", &before, &after, records)
			if err != nil || !reflect.DeepEqual(display.Normalizations(), want) {
				t.Fatal(display, err)
			}
			records.TeamApprovals[0].ReviewerID = 999
			for _, returned := range []RulesetNormalizations{change.Normalizations(), change.Display().Normalizations(), display.Normalizations()} {
				returned.TeamApprovals[0].RawValue = 1
			}
			if !reflect.DeepEqual(change.Normalizations(), want) || !reflect.DeepEqual(change.Display().Normalizations(), want) || !reflect.DeepEqual(display.Normalizations(), want) {
				t.Fatal("evidence alias")
			}
			if mode == "equal" {
				if _, err := NewRulesetChange("update", 7, &before, &after); err == nil {
					t.Fatal("ordinary equal update became valid")
				}
			}
		}
	}
}

func TestTeamApprovalsChangesRejectForgedMetadataBesideValidCompanions(t *testing.T) {
	for _, companions := range []bool{false, true} {
		for _, mutate := range []func(*RulesetDeclaration, *RulesetDeclaration, *RulesetNormalizations){
			func(b, a *RulesetDeclaration, n *RulesetNormalizations) {
				n.TeamApprovals[0] = RulesetTeamApprovalsNormalization{}
			},
			func(b, a *RulesetDeclaration, n *RulesetNormalizations) { n.TeamApprovals[0].ReviewerID = 99 },
			func(b, a *RulesetDeclaration, n *RulesetNormalizations) { n.TeamApprovals[0].RawValue = 10 },
			func(b, a *RulesetDeclaration, n *RulesetNormalizations) { n.TeamApprovals[0].Fallback = 1 },
			func(b, a *RulesetDeclaration, n *RulesetNormalizations) { b.ID = nil },
			func(b, a *RulesetDeclaration, n *RulesetNormalizations) { a.ID = rsID(8) },
			func(b, a *RulesetDeclaration, n *RulesetNormalizations) {
				*(*ruleOfForApprovals(*a).RequiredReviewers)[0].MinimumApprovals = 11
			},
		} {
			state := normalizedTeamApprovalsState(companions)
			before, after := cloneRuleset(state.Policy), cloneRuleset(state.Policy)
			records := teamApprovalsRecords(state)
			mutate(&before, &after, &records)
			if c, err := NewRulesetChangeWithNormalizations("update", 7, &before, &after, records); err == nil || !reflect.DeepEqual(c, RulesetChange{}) {
				t.Fatal(c, err)
			}
			if c, err := NewRulesetDisplayChangeWithNormalizations("update", &before, &after, records); err == nil || !reflect.DeepEqual(c, RulesetDisplayChange{}) {
				t.Fatal(c, err)
			}
		}
	}
}
