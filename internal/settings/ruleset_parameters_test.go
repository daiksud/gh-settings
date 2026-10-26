package settings

import (
	"reflect"
	"testing"
)

func paramBool(v bool) *bool { return &v }
func paramInt(v int) *int    { return &v }
func reviewPolicy() RulesetDeclaration {
	d := basicPolicy("default")
	d.BypassActors = &[]BypassActor{}
	d.Rules = &[]RulesetRule{{Type: "deletion"}, {Type: "non_fast_forward"}, {Type: "required_linear_history"}, {Type: "pull_request", Parameters: &RuleParameters{AllowedMergeMethods: rsList("squash"), DismissStaleReviewsOnPush: paramBool(true), RequireCodeOwnerReview: paramBool(true), RequireExtraApprovalForUnattributedChanges: paramBool(false), RequireLastPushApproval: paramBool(false), RequiredApprovingReviewCount: paramInt(1), RequiredReviewThreadResolution: paramBool(true), RequiredReviewers: &[]RequiredReviewer{{Reviewer: ReviewActor{ID: 21, Type: "Team"}, MinimumApprovals: paramInt(1), FilePatterns: rsList("src/**", "!src/generated/**", "src/generated/manual.go")}}, DismissalRestriction: &ReviewDismissalRestriction{Enabled: paramBool(true), AllowedActors: &[]ReviewActor{{ID: 32, Type: "Team"}}}}}, {Type: "required_status_checks", Parameters: &RuleParameters{RequiredStatusChecks: &[]RequiredStatusCheck{{Context: "pass", IntegrationID: rsID(15368)}, {Context: "gh-settings", IntegrationID: rsID(15368)}}, StrictRequiredStatusChecksPolicy: paramBool(true), DoNotEnforceOnCreate: paramBool(false)}}}
	return d
}
func ruleOf(t *testing.T, d RulesetDeclaration, kind string) *RuleParameters {
	t.Helper()
	for _, r := range *d.Rules {
		if r.Type == kind {
			return r.Parameters
		}
	}
	t.Fatal("missing rule", kind)
	return nil
}
func TestRulesetReviewChecksDefaultOmittedAndKeepExplicitValues(t *testing.T) {
	before := reviewPolicy()
	current := rsObserved(t, RulesetState{7, before})
	desired := basicPolicy("default")
	desired.ID = rsID(7)
	desired.BypassActors = nil
	desired.Rules = &[]RulesetRule{
		{Type: "pull_request", Parameters: &RuleParameters{RequiredApprovingReviewCount: paramInt(2)}},
		{Type: "required_status_checks", Parameters: &RuleParameters{}},
	}
	changes, err := ReconcileRulesets(current, rsDesired(t, desired))
	if err != nil || len(changes) != 1 {
		t.Fatal(changes, err)
	}
	after := *changes[0].After()
	pr, checks := ruleOf(t, after, "pull_request"), ruleOf(t, after, "required_status_checks")
	if *pr.RequiredApprovingReviewCount != 2 || *pr.RequireCodeOwnerReview || *pr.DismissStaleReviewsOnPush || *pr.DismissalRestriction.Enabled || len(*pr.DismissalRestriction.AllowedActors) != 0 || len(*pr.RequiredReviewers) != 0 || len(*checks.RequiredStatusChecks) != 0 || *checks.StrictRequiredStatusChecksPolicy || *checks.DoNotEnforceOnCreate {
		t.Fatal("defaults borrowed remote protection", after)
	}
	if !reflect.DeepEqual(*pr.AllowedMergeMethods, []string{"merge", "rebase", "squash"}) {
		t.Fatal("merge-method default", *pr.AllowedMergeMethods)
	}
	repeated, err := ReconcileRulesets(rsObserved(t, RulesetState{7, after}), rsDesired(t, desired))
	if err != nil || len(repeated) != 0 {
		t.Fatal("not idempotent", repeated, err)
	}
	created := desired
	created.ID, created.Name = nil, "new"
	operations, err := ReconcileRulesets(rsObserved(t), rsDesired(t, created))
	if err != nil || len(operations) != 1 {
		t.Fatal(operations, err)
	}
	createPR := ruleOf(t, *operations[0].After(), "pull_request")
	if !reflect.DeepEqual(pr, createPR) {
		t.Fatal("create/update defaults differ")
	}
	*pr.RequireCodeOwnerReview = true
	if *ruleOf(t, *changes[0].After(), "pull_request").RequireCodeOwnerReview {
		t.Fatal("mutable snapshot escaped")
	}
}
func TestRulesetReviewerPatternsOrderIsPolicy(t *testing.T) {
	before := reviewPolicy()
	p := ruleOf(t, before, "pull_request")
	*(*p.RequiredReviewers)[0].FilePatterns = []string{"**", "!docs/**", "docs/README.md", "**"}
	current := rsObserved(t, RulesetState{7, before})
	before.ID = rsID(7)
	desired := rsDesired(t, before)
	if ops, e := ReconcileRulesets(current, desired); e != nil || len(ops) != 0 {
		t.Fatal("unchanged pattern sequence differs", ops, e)
	}
	changed := cloneRuleset(before)
	patterns := (*ruleOf(t, changed, "pull_request").RequiredReviewers)[0].FilePatterns
	(*patterns)[0], (*patterns)[1] = (*patterns)[1], (*patterns)[0]
	ops, e := ReconcileRulesets(current, rsDesired(t, changed))
	if e != nil || len(ops) != 1 {
		t.Fatal("order-only change hidden", ops, e)
	}
	if !reflect.DeepEqual(*(*ruleOf(t, *ops[0].After(), "pull_request").RequiredReviewers)[0].FilePatterns, *patterns) {
		t.Fatal("reordered patterns normalized")
	}
}
func TestRulesetCheckIdentityAndCollectionOrdering(t *testing.T) {
	before := reviewPolicy()
	p := ruleOf(t, before, "required_status_checks")
	p.RequiredStatusChecks = &[]RequiredStatusCheck{{Context: "build", IntegrationID: rsID(2)}, {Context: "build"}, {Context: "build", IntegrationID: rsID(1)}}
	observed := rsObserved(t, RulesetState{7, before})
	reordered := cloneRuleset(before)
	reordered.ID = rsID(7)
	p = ruleOf(t, reordered, "required_status_checks")
	(*p.RequiredStatusChecks)[0], (*p.RequiredStatusChecks)[2] = (*p.RequiredStatusChecks)[2], (*p.RequiredStatusChecks)[0]
	if ops, e := ReconcileRulesets(observed, rsDesired(t, reordered)); e != nil || len(ops) != 0 {
		t.Fatal("check array order should be irrelevant", ops, e)
	}
	p.RequiredStatusChecks = &[]RequiredStatusCheck{{Context: "build"}}
	ops, e := ReconcileRulesets(observed, rsDesired(t, reordered))
	if e != nil || len(ops) != 1 || len(*ruleOf(t, *ops[0].After(), "required_status_checks").RequiredStatusChecks) != 1 {
		t.Fatal("integration identities conflated", ops, e)
	}
}
func TestRulesetReviewParameterValidationAndCreation(t *testing.T) {
	for _, change := range []func(*RuleParameters){func(p *RuleParameters) { p.RequiredApprovingReviewCount = paramInt(-1) }, func(p *RuleParameters) { p.AllowedMergeMethods = rsList() }, func(p *RuleParameters) { p.RequiredStatusChecks = &[]RequiredStatusCheck{} }, func(p *RuleParameters) {
		p.RequiredReviewers = &[]RequiredReviewer{{Reviewer: ReviewActor{ID: 1, Type: "User"}}}
	}, func(p *RuleParameters) {
		p.RequiredReviewers = &[]RequiredReviewer{{Reviewer: ReviewActor{ID: 1, Type: "Team"}, MinimumApprovals: paramInt(11)}}
	}, func(p *RuleParameters) {
		p.DismissalRestriction = &ReviewDismissalRestriction{AllowedActors: &[]ReviewActor{{ID: 0, Type: "Team"}}}
	}} {
		d := reviewPolicy()
		change(ruleOf(t, d, "pull_request"))
		if _, e := NewRulesetDeclarations([]RulesetDeclaration{d}); e == nil {
			t.Fatal("invalid PR parameters accepted")
		}
	}
	for _, change := range []func(*RuleParameters){func(p *RuleParameters) {
		p.RequiredStatusChecks = &[]RequiredStatusCheck{{Context: "pass"}, {Context: "pass"}}
	}, func(p *RuleParameters) {
		p.RequiredStatusChecks = &[]RequiredStatusCheck{{Context: "pass", IntegrationID: rsID(0)}}
	}, func(p *RuleParameters) { p.RequireCodeOwnerReview = paramBool(false) }} {
		d := reviewPolicy()
		change(ruleOf(t, d, "required_status_checks"))
		if _, e := NewRulesetDeclarations([]RulesetDeclaration{d}); e == nil {
			t.Fatal("invalid check params accepted")
		}
	}
	for _, kind := range []string{"pull_request", "required_status_checks"} {
		policy := basicPolicy("new")
		policy.Rules = &[]RulesetRule{{Type: kind}}
		d, err := NewRulesetDeclarations([]RulesetDeclaration{policy})
		if err != nil {
			t.Fatal(err)
		}
		if changes, err := ReconcileRulesets(rsObserved(t), &d); err != nil || len(changes) != 1 {
			t.Fatal("defaulted rule create failed", changes, err)
		}
	}
	missing := basicPolicy("selected")
	missing.Rules = &[]RulesetRule{{Type: "pull_request", Parameters: &RuleParameters{RequiredReviewers: &[]RequiredReviewer{{Reviewer: ReviewActor{ID: 99, Type: "Team"}}}}}}
	if _, err := NewRulesetDeclarations([]RulesetDeclaration{missing}); err == nil {
		t.Fatal("reviewer scope/approval intent guessed")
	}
}

func TestRulesetReviewerAndDismissalMembership(t *testing.T) {
	before := reviewPolicy()
	desired := basicPolicy("default")
	desired.ID = rsID(7)
	desired.Rules = &[]RulesetRule{{Type: "pull_request", Parameters: &RuleParameters{
		RequiredReviewers:    &[]RequiredReviewer{{Reviewer: ReviewActor{ID: 99, Type: "Team"}, MinimumApprovals: paramInt(2), FilePatterns: rsList("**")}},
		DismissalRestriction: &ReviewDismissalRestriction{AllowedActors: &[]ReviewActor{{ID: 45, Type: "User"}}},
	}}}
	ops, err := ReconcileRulesets(rsObserved(t, RulesetState{7, before}), rsDesired(t, desired))
	if err != nil || len(ops) != 1 {
		t.Fatal(ops, err)
	}
	p := ruleOf(t, *ops[0].After(), "pull_request")
	if len(*p.RequiredReviewers) != 1 || (*p.RequiredReviewers)[0].Reviewer.ID != 99 || *(*p.RequiredReviewers)[0].MinimumApprovals != 2 || *p.DismissalRestriction.Enabled || len(*p.DismissalRestriction.AllowedActors) != 1 || (*p.DismissalRestriction.AllowedActors)[0].ID != 45 {
		t.Fatal("membership or fixed restriction default changed", p)
	}
}

func TestPullRequestRuleTargetUsesDeclaredObjectNotRetainedRules(t *testing.T) {
	tag := "tag"
	invalid := reviewPolicy()
	invalid.Target = &tag
	if _, err := NewRulesetDeclarations([]RulesetDeclaration{invalid}); err == nil {
		t.Fatal("PR rule accepted on tag target")
	}
	before := reviewPolicy()
	desired := basicPolicy("default")
	desired.ID, desired.Target = rsID(7), &tag
	desired.Rules, desired.BypassActors = nil, nil
	changes, err := ReconcileRulesets(rsObserved(t, RulesetState{7, before}), rsDesired(t, desired))
	if err != nil || len(changes) != 1 || len(*changes[0].After().Rules) != 0 || *changes[0].After().Target != "tag" {
		t.Fatal("target edit retained undeclared PR rule", changes, err)
	}
}
