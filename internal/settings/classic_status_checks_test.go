package settings

import (
	"reflect"
	"testing"
)

func statusPolicy(strict bool, checks ...ClassicStatusCheck) *ClassicStatusChecks {
	yes := true
	values := append([]ClassicStatusCheck{}, checks...)
	return &ClassicStatusChecks{Enabled: &yes, RequireBranchesToBeUpToDateBeforeMerging: &strict, StatusChecks: &values}
}

func statusState(t *testing.T, raw ClassicCompanions) ClassicState {
	t.Helper()
	s := classicModelState(7, "main")
	s.Companions.RequiresStatusChecks, s.Companions.RequiresStrictStatusChecks = raw.RequiresStatusChecks, raw.RequiresStrictStatusChecks
	s.Companions.Contexts, s.Companions.Checks = raw.Contexts, raw.Checks
	var err error
	s.Rule.RequireStatusChecksToPassBeforeMerging, err = ProjectClassicStatusChecks(s.Companions)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestClassicStatusChecksUnionAndExactIdentity(t *testing.T) {
	yes, no := true, false
	checks := []ClassicStatusCheck{{Name: "screening", AcceptUpdatesFrom: "APP_one"}}
	for _, tc := range []struct {
		name string
		g    *ClassicStatusChecks
		bad  bool
	}{
		{"omitted", nil, false}, {"empty", &ClassicStatusChecks{}, false}, {"disabled", &ClassicStatusChecks{Enabled: &no}, false},
		{"active empty", statusPolicy(false), false}, {"exact names", statusPolicy(true, ClassicStatusCheck{Name: " name ", AcceptUpdatesFrom: "APP_one"}, ClassicStatusCheck{Name: "NAME", AcceptUpdatesFrom: "APP_one"}), false},
		{"missing enabled", &ClassicStatusChecks{StatusChecks: &checks}, true}, {"missing children", &ClassicStatusChecks{Enabled: &yes}, true},
		{"disabled children", &ClassicStatusChecks{Enabled: &no, StatusChecks: &checks}, true}, {"duplicate sources", statusPolicy(false, checks[0], ClassicStatusCheck{Name: "screening", AcceptUpdatesFrom: "any"}), true},
		{"blank name", statusPolicy(false, ClassicStatusCheck{Name: " ", AcceptUpdatesFrom: "any"}), true}, {"missing source", statusPolicy(false, ClassicStatusCheck{Name: "x", AcceptUpdatesFrom: ""}), true}, {"control source", statusPolicy(false, ClassicStatusCheck{Name: "x", AcceptUpdatesFrom: "APP\n"}), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if (ValidateClassicStatusChecks(tc.g) != nil) != tc.bad {
				t.Fatal(tc.name)
			}
		})
	}
	g := statusPolicy(false, ClassicStatusCheck{Name: "b", AcceptUpdatesFrom: "APP_one"}, ClassicStatusCheck{Name: "a", AcceptUpdatesFrom: "APP_one"})
	clone := CloneClassicStatusChecks(g)
	(*clone.StatusChecks)[0].Name = "changed"
	*clone.Enabled = false
	if (*g.StatusChecks)[0].Name != "b" || !*g.Enabled {
		t.Fatal("clone aliases original")
	}
	if !reflect.DeepEqual(ClassicStatusCheckAppIDs(g), []string{"APP_one"}) {
		t.Fatal("lookup identity deduplication")
	}
	if !ClassicStatusChecksMatch(g, statusPolicy(false, ClassicStatusCheck{Name: "a", AcceptUpdatesFrom: "APP_one"}, ClassicStatusCheck{Name: "b", AcceptUpdatesFrom: "APP_one"})) {
		t.Fatal("order affected active equality")
	}
	if ClassicStatusChecksMatch(g, statusPolicy(false, ClassicStatusCheck{Name: "a", AcceptUpdatesFrom: "any"}, ClassicStatusCheck{Name: "b", AcceptUpdatesFrom: "APP_one"})) {
		t.Fatal("source identity lost")
	}
}

func TestClassicStatusChecksProjectionCoherenceAndLatentState(t *testing.T) {
	app := "APP_one"
	contexts := []string{"b", "a"}
	checks := []ClassicCheck{{Context: "a"}, {Context: "b", AppID: &app}}
	base := ClassicCompanions{RequiresStatusChecks: true, Contexts: ClassicNullable[[]string]{Present: true, Value: &contexts}, Checks: ClassicNullable[[]ClassicCheck]{Present: true, Value: &checks}}
	g, err := ProjectClassicStatusChecks(base)
	if err != nil || (*g.StatusChecks)[0].Name != "a" || (*g.StatusChecks)[0].AcceptUpdatesFrom != "any" || contexts[0] != "b" {
		t.Fatal(g, err)
	}
	for _, mode := range []string{"missing", "null", "duplicate contexts", "duplicate checks", "divergent", "invalid app"} {
		raw := base
		switch mode {
		case "missing":
			raw.Checks.Present = false
		case "null":
			raw.Contexts.Value = nil
		case "duplicate contexts":
			x := []string{"a", "a"}
			raw.Contexts.Value = &x
		case "duplicate checks":
			x := []ClassicCheck{{Context: "a"}, {Context: "a", AppID: &app}}
			raw.Checks.Value = &x
		case "divergent":
			x := []string{"a", "other"}
			raw.Contexts.Value = &x
		case "invalid app":
			id := ""
			x := []ClassicCheck{{Context: "a", AppID: &id}}
			raw.Checks.Value = &x
		}
		if _, err := ProjectClassicStatusChecks(raw); err == nil {
			t.Fatal("accepted active", mode)
		}
		if mode == "divergent" || mode == "duplicate contexts" || mode == "duplicate checks" || mode == "null" {
			raw.RequiresStatusChecks = false
			g, err := ProjectClassicStatusChecks(raw)
			if err != nil || *g.Enabled || g.StatusChecks != nil {
				t.Fatal("lost latent typed state", mode, err)
			}
		}
	}
	before := statusState(t, base)
	after := cloneClassicState(before)
	reordered := []string{"a", "b"}
	after.Companions.Contexts.Value = &reordered
	desired := ClassicRule{ID: before.Rule.ID, BranchNamePattern: "main"}
	if VerifyClassicPreservation(before, after, desired) == nil {
		t.Fatal("omitted raw ordering lost")
	}
	desired.RequireStatusChecksToPassBeforeMerging = CloneClassicStatusChecks(g)
	if err := VerifyClassicPreservation(before, after, desired); err != nil {
		t.Fatal("selected set order", err)
	}
	prune, err := PrepareClassicPrune(ClassicObservation{RepositoryID: "R", Rules: []ClassicState{before}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := prune.Recheck(ClassicObservation{RepositoryID: "R", Rules: []ClassicState{after}}); err == nil {
		t.Fatal("prune lost raw ordering")
	}
	no := false
	disabled := cloneClassicState(before)
	disabled.Companions.RequiresStatusChecks = false
	disabled.Companions.Contexts.Value = nil
	disabled.Companions.Checks.Value = nil
	disabled.Rule.RequireStatusChecksToPassBeforeMerging = &ClassicStatusChecks{Enabled: &no}
	desired.RequireStatusChecksToPassBeforeMerging = &ClassicStatusChecks{Enabled: &no}
	if err := VerifyClassicPreservation(before, disabled, desired); err != nil {
		t.Fatal("selected disable incorrectly preserves latent state", err)
	}
}

func TestClassicStatusChecksNativePayload(t *testing.T) {
	no := false
	desired := statusPolicy(false)
	for _, before := range []*ClassicStatusChecks{nil, {Enabled: &no}} {
		input := ClassicStatusChecksInput(before, desired)
		if len(input) != 3 || input["requiresStatusChecks"] != true || input["requiresStrictStatusChecks"] != false || len(input["requiredStatusChecks"].([]map[string]string)) != 0 {
			t.Fatal(input)
		}
	}
	before := statusPolicy(true, ClassicStatusCheck{Name: "x", AcceptUpdatesFrom: "APP_one"})
	input := ClassicStatusChecksInput(before, statusPolicy(true, ClassicStatusCheck{Name: "x", AcceptUpdatesFrom: "any"}))
	if len(input) != 1 || input["requiredStatusChecks"].([]map[string]string)[0]["appId"] != "any" {
		t.Fatal(input)
	}
	if input := ClassicStatusChecksInput(before, &ClassicStatusChecks{Enabled: &no}); !reflect.DeepEqual(input, map[string]any{"requiresStatusChecks": false}) {
		t.Fatal(input)
	}
}
