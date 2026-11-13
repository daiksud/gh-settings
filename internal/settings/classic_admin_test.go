package settings

import "testing"

func TestClassicAdminExportCloneAndOmission(t *testing.T) {
	for _, value := range []bool{false, true} {
		s := classicModelState(7, "main")
		s.Rule.DoNotAllowBypassingTheAboveSettings = &value
		o := ClassicObservation{RepositoryID: "R", Rules: []ClassicState{s}}
		exported, err := ExportClassicRules(o)
		if err != nil || exported[0].DoNotAllowBypassingTheAboveSettings == nil || *exported[0].DoNotAllowBypassingTheAboveSettings != value {
			t.Fatal(exported, err)
		}
		if changes, err := CompareClassicRules(o, exported); err != nil || len(changes) != 0 {
			t.Fatal(changes, err)
		}
		*exported[0].DoNotAllowBypassingTheAboveSettings = !value
		if *s.Rule.DoNotAllowBypassingTheAboveSettings != value {
			t.Fatal("export aliases acquired administrator enforcement")
		}
		partial := ClassicRule{ID: s.Rule.ID, BranchNamePattern: "main"}
		if !ClassicMatches(s.Rule, partial) {
			t.Fatal("omitted administrator enforcement acquired ownership")
		}
	}
}
