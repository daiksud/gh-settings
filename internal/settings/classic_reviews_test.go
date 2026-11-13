package settings

import (
	"reflect"
	"testing"
)

func modelReviews(enabled bool, count int) *ClassicPullRequestReviews {
	v := &ClassicPullRequestReviews{Enabled: &enabled}
	if enabled {
		no, approve := false, count > 0
		v.RequireApprovals = &approve
		v.DismissStalePullRequestApprovalsWhenNewCommitsArePushed = &no
		v.RequireReviewFromCodeOwners = &no
		v.RequireApprovalOfTheMostRecentReviewablePush = &no
		if approve {
			v.RequiredNumberOfApprovalsBeforeMerging = &count
		}
	}
	return v
}

func TestClassicReviewProjectionAndRawOwnership(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, count := range []*int{nil, new(0), new(1), new(6), new(7), new(-1)} {
			state := classicModelState(7, "main")
			c := &state.Companions
			c.RequiresApprovingReviews = enabled
			c.ReviewCount.Value = count
			c.DismissesStaleReviews = true
			c.RequiresCodeOwnerReviews = true
			c.RequireLastPushApproval = true
			group, err := ProjectClassicReviews(*c)
			bad := enabled && (count == nil || *count < 0 || *count > 6)
			if (err != nil) != bad {
				t.Fatal(enabled, count, group, err)
			}
			if bad {
				continue
			}
			state.Rule.RequireAPullRequestBeforeMerging = group
			o := ClassicObservation{RepositoryID: "R", Rules: []ClassicState{state}}
			exported, err := ExportClassicRules(o)
			if err != nil {
				t.Fatal(err)
			}
			if !ClassicMatches(state.Rule, exported[0]) {
				t.Fatal("roundtrip")
			}
			*exported[0].RequireAPullRequestBeforeMerging.Enabled = !enabled
			if *group.Enabled != enabled {
				t.Fatal("export aliases observation")
			}
			if !enabled && !reflect.DeepEqual(group, modelReviews(false, 0)) {
				t.Fatal("inactive values leaked", group)
			}
			prepared, err := PrepareClassicPrune(o, []ClassicRule{})
			if err != nil {
				t.Fatal(err)
			}
			changed := cloneClassicState(state)
			changed.Companions.ReviewCount.Value = new(3)
			if prepared.Recheck(ClassicObservation{RepositoryID: "R", Rules: []ClassicState{changed}}) == nil {
				t.Fatal("prune ignored raw count")
			}
			omitted := ClassicRule{ID: state.Rule.ID, BranchNamePattern: "main"}
			if VerifyClassicPreservation(state, changed, omitted) == nil {
				t.Fatal("omission ignored raw count")
			}
		}
	}
}

func TestClassicReviewTransitionGuards(t *testing.T) {
	for _, mode := range []string{"dismissal-flag", "dismissal-actor", "bypass-actor"} {
		state := classicModelState(7, "main")
		state.Companions.RequiresApprovingReviews = true
		state.Rule.RequireAPullRequestBeforeMerging, _ = ProjectClassicReviews(state.Companions)
		switch mode {
		case "dismissal-flag":
			state.Companions.RestrictsReviewDismissals = true
		case "dismissal-actor":
			state.Companions.ReviewDismissalActors = []ClassicActor{{Kind: "User", ID: "U"}}
		case "bypass-actor":
			state.Companions.PullRequestActors = []ClassicActor{{Kind: "User", ID: "U"}}
		}
		d := ClassicRule{ID: state.Rule.ID, BranchNamePattern: "main", RequireAPullRequestBeforeMerging: modelReviews(false, 0)}
		if _, err := CompareClassicRules(ClassicObservation{RepositoryID: "R", Rules: []ClassicState{state}}, []ClassicRule{d}); err == nil {
			t.Fatal(mode, "plan accepted disabling")
		}
		d.RequireAPullRequestBeforeMerging = modelReviews(true, 0)
		if err := ValidateClassicReviewTransition(state, d); err != nil {
			t.Fatal("count zero blocked", err)
		}
		state.Companions.RequiresApprovingReviews = false
		state.Rule.RequireAPullRequestBeforeMerging, _ = ProjectClassicReviews(state.Companions)
		d.RequireAPullRequestBeforeMerging = modelReviews(false, 0)
		if changes, err := CompareClassicRules(ClassicObservation{RepositoryID: "R", Rules: []ClassicState{state}}, []ClassicRule{d}); err != nil || len(changes) != 0 {
			t.Fatal("latent no-op blocked", changes, err)
		}
	}
}

func TestClassicReviewsCloneAllChildren(t *testing.T) {
	group := modelReviews(true, 6)
	cloned := CloneClassicReviews(group)
	*cloned.Enabled = false
	*cloned.RequireApprovals = false
	*cloned.RequiredNumberOfApprovalsBeforeMerging = 1
	*cloned.DismissStalePullRequestApprovalsWhenNewCommitsArePushed = true
	*cloned.RequireReviewFromCodeOwners = true
	*cloned.RequireApprovalOfTheMostRecentReviewablePush = true
	if !reflect.DeepEqual(group, modelReviews(true, 6)) {
		t.Fatal("clone aliases children")
	}
}
