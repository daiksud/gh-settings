package settings

import (
	"testing"
)

func TestRulesetUnownedPreviewPreservation(t *testing.T) {
	for _, value := range []bool{false, true} {
		before := reviewPolicy()
		before.ID = rsID(7)
		ruleOf(t, before, "pull_request").RequireExtraApprovalForUnattributedChanges = paramBool(value)
		desired := cloneRuleset(before)
		ruleOf(t, desired, "pull_request").RequireExtraApprovalForUnattributedChanges = nil
		if changes, err := ReconcileRulesets(rsObserved(t, RulesetState{ID: 7, Policy: before}), rsDesired(t, desired)); err != nil || len(changes) != 0 {
			t.Fatal("unowned metadata caused difference", changes, err)
		}
		ruleOf(t, desired, "pull_request").RequiredApprovingReviewCount = paramInt(2)
		changes, err := ReconcileRulesets(rsObserved(t, RulesetState{ID: 7, Policy: before}), rsDesired(t, desired))
		if err != nil || len(changes) != 1 {
			t.Fatal(changes, err)
		}
		actual, present := RulesetPreviewApproval(changes[0].After())
		if !present || actual == nil || *actual != value {
			t.Fatal("lost acquired flag", actual)
		}
		*actual = !value
		again, _ := RulesetPreviewApproval(changes[0].After())
		if *again != value {
			t.Fatal("metadata alias")
		}
		tampered := cloneRuleset(*changes[0].After())
		ruleOf(t, tampered, "pull_request").RequireExtraApprovalForUnattributedChanges = paramBool(!value)
		if _, err := NewRulesetChange("update", 7, &before, &tampered); err == nil {
			t.Fatal("accepted unmanaged flag change")
		}
		for _, after := range []*RulesetDeclaration{nil, func() *RulesetDeclaration {
			p := cloneRuleset(before)
			p.Rules = &[]RulesetRule{{Type: "deletion"}}
			return &p
		}()} {
			if err := PreserveRulesetUnmanagedParameters(&before, after); err == nil {
				t.Fatal("removed unowned container")
			}
		}
	}
}
