package settings

import "testing"

func TestClassicLockExportCloneAndOmission(t *testing.T) {
	for _, value := range []bool{false, true} {
		s := classicModelState(7, "main")
		s.Rule.LockBranch = &value
		o := ClassicObservation{RepositoryID: "R", Rules: []ClassicState{s}}
		exported, err := ExportClassicRules(o)
		if err != nil || exported[0].LockBranch == nil || *exported[0].LockBranch != value {
			t.Fatal(exported, err)
		}
		if changes, err := CompareClassicRules(o, exported); err != nil || len(changes) != 0 {
			t.Fatal(changes, err)
		}
		*exported[0].LockBranch = !value
		if *s.Rule.LockBranch != value {
			t.Fatal("export aliases acquired lock")
		}
		partial := ClassicRule{ID: s.Rule.ID, BranchNamePattern: "main"}
		if !ClassicMatches(s.Rule, partial) {
			t.Fatal("omitted lock acquired ownership")
		}
	}
}
