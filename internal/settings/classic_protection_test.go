package settings

import (
	"reflect"
	"testing"
)

func classicModelState(id int64, pattern string) ClassicState {
	no := false
	n := 2
	empty := []string{}
	checks := []ClassicCheck{}
	return ClassicState{NodeID: "B" + pattern, RepositoryID: "R", Rule: ClassicRule{ID: &id, BranchNamePattern: pattern, RequireConversationResolutionBeforeMerging: &no, RequireSignedCommits: &no, RequireLinearHistory: &no, AllowDeletions: &no, LockBranch: &no, DoNotAllowBypassingTheAboveSettings: &no, RequireDeploymentsToSucceedBeforeMerging: &ClassicDeployments{Enabled: &no}, AllowForcePushes: &ClassicForcePush{Enabled: &no}}, Companions: ClassicCompanions{ReviewCount: ClassicNullable[int]{Present: true, Value: &n}, Deployments: ClassicNullable[[]string]{Present: true, Value: &empty}, Contexts: ClassicNullable[[]string]{Present: true, Value: &empty}, Checks: ClassicNullable[[]ClassicCheck]{Present: true, Value: &checks}, PushActors: []ClassicActor{}, ReviewDismissalActors: []ClassicActor{}, ForcePushActors: []ClassicActor{}, PullRequestActors: []ClassicActor{}}}
}
func TestClassicSelectionOwnershipCollisionsAndPriority(t *testing.T) {
	id, missing := int64(7), int64(90)
	yes, no := true, false
	observed := ClassicObservation{RepositoryID: "R", Rules: []ClassicState{classicModelState(7, "release/**"), classicModelState(8, "main")}}
	for _, tc := range []struct {
		name    string
		desired []ClassicRule
		bad     bool
		changes int
	}{
		{"omitted", nil, false, 0}, {"empty", []ClassicRule{}, false, 0}, {"name-only", []ClassicRule{{ID: &id, BranchNamePattern: "release/**"}}, false, 0}, {"false", []ClassicRule{{ID: &id, BranchNamePattern: "release/**", AllowDeletions: &no}}, false, 0}, {"true", []ClassicRule{{ID: &id, BranchNamePattern: "release/**", AllowDeletions: &yes}}, false, 1},
		{"create-collision", []ClassicRule{{BranchNamePattern: "main"}}, true, 0}, {"missing-id", []ClassicRule{{ID: &missing, BranchNamePattern: "new"}}, true, 0}, {"rename-collision", []ClassicRule{{ID: &id, BranchNamePattern: "main"}}, true, 0}, {"duplicate-id", []ClassicRule{{ID: &id, BranchNamePattern: "a"}, {ID: &id, BranchNamePattern: "b"}}, true, 0}, {"create-order", []ClassicRule{{BranchNamePattern: "z/**"}, {BranchNamePattern: "z/*"}}, false, 2},
		{"requested-collision", []ClassicRule{{ID: &id, BranchNamePattern: "new"}, {BranchNamePattern: "new"}}, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CompareClassicRules(observed, tc.desired)
			if (err != nil) != tc.bad || (!tc.bad && len(got) != tc.changes) {
				t.Fatal(got, err)
			}
			if tc.name == "create-order" && (got[0].After.BranchNamePattern != "z/**" || got[1].After.BranchNamePattern != "z/*") {
				t.Fatal("creation priority reordered", got)
			}
		})
	}
	// Existing duplicate patterns remain ID-distinct, without manufacturing priority.
	observed.Rules[1].Rule.BranchNamePattern = "release/**"
	if _, err := PrepareClassicRules(observed, []ClassicRule{{ID: &id, BranchNamePattern: "release/**", AllowDeletions: &yes}}); err != nil {
		t.Fatal(err)
	}
}
func TestClassicNullableCompanionsAndExport(t *testing.T) {
	s := classicModelState(7, "qa/**/*")
	s.Companions.ReviewCount = ClassicNullable[int]{Present: true}
	s.Companions.Deployments = ClassicNullable[[]string]{Present: true}
	o := ClassicObservation{RepositoryID: "R", Rules: []ClassicState{s}}
	rules, err := ExportClassicRules(o)
	if err != nil || len(rules) != 1 || rules[0].AllowDeletions == nil || *rules[0].AllowDeletions {
		t.Fatal(rules, err)
	}
	*rules[0].ID = 999
	if *s.Rule.ID != 7 {
		t.Fatal("export identity aliased")
	}
	d := ClassicRule{ID: s.Rule.ID, BranchNamePattern: s.Rule.BranchNamePattern}
	after := s
	empty := []string{}
	after.Companions.Deployments.Value = &empty
	if VerifyClassicPreservation(s, after, d) == nil {
		t.Fatal("null collapsed to empty")
	}
	s.Companions.RequiresApprovingReviews = true
	o.Rules[0] = s
	if o.Validate() == nil {
		t.Fatal("enabled null count accepted")
	}
	if reflect.DeepEqual(ClassicObservation{}, ClassicObservation{RepositoryID: "R", Rules: []ClassicState{}}) {
		t.Fatal("unacquired treated as empty")
	}
}
