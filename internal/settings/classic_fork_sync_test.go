package settings

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func forkModelState(fork, locked, sync bool) ClassicState {
	s := classicModelState(7, "main")
	s.IsFork, s.Rule.LockBranch, s.Companions.LockAllowsFetchAndMerge = fork, &locked, sync
	s.Rule.AllowForkSyncing = ClassicForkSyncProjection(fork, locked, sync)
	return s
}

func TestClassicForkSyncProjectionAndOwnership(t *testing.T) {
	for _, fork := range []bool{false, true} {
		for _, locked := range []bool{false, true} {
			for _, sync := range []bool{false, true} {
				s := forkModelState(fork, locked, sync)
				o := ClassicObservation{RepositoryID: "R", IsFork: fork, Rules: []ClassicState{s}}
				exported, err := ExportClassicRules(o)
				if err != nil || len(exported) != 1 {
					t.Fatal(exported, err)
				}
				if (exported[0].AllowForkSyncing != nil) != (fork && locked) {
					t.Fatal(exported)
				}
				if fork && locked {
					if *exported[0].AllowForkSyncing != sync {
						t.Fatal(exported)
					}
					o.Rules[0].Rule.AllowForkSyncing = nil
					if o.Validate() == nil {
						t.Fatal("missing locked-fork projection accepted")
					}
					o.Rules[0] = s
				}
				yes := true
				d := ClassicRule{ID: s.Rule.ID, BranchNamePattern: "main", LockBranch: &yes, AllowForkSyncing: &sync}
				changes, err := CompareClassicRules(o, []ClassicRule{d})
				if (err != nil) != !fork {
					t.Fatal(fork, locked, sync, changes, err)
				}
				if fork && len(changes) != map[bool]int{true: 0, false: 1}[locked] {
					t.Fatal(changes)
				}
				if fork && err == nil && ValidateClassicChanges(changes) != nil {
					t.Fatal(changes)
				}
			}
		}
	}
}

func TestClassicForkSyncOmissionPreservationAndPrune(t *testing.T) {
	s := forkModelState(true, true, true)
	no, yes := false, true
	d := ClassicRule{ID: s.Rule.ID, BranchNamePattern: "main", LockBranch: &no}
	after := cloneClassicState(s)
	after.Rule.LockBranch, after.Rule.AllowForkSyncing = &no, nil
	if err := VerifyClassicPreservation(s, after, d); err != nil {
		t.Fatal(err)
	}
	after.Companions.LockAllowsFetchAndMerge = false
	if VerifyClassicPreservation(s, after, d) == nil {
		t.Fatal("unlock erased omitted raw sync")
	}
	after.Rule.LockBranch, after.Rule.AllowForkSyncing = &yes, &no
	d.LockBranch, d.AllowForkSyncing = &yes, &no
	if err := VerifyClassicPreservation(s, after, d); err != nil {
		t.Fatal(err)
	}
	o := ClassicObservation{RepositoryID: "R", IsFork: true, Archived: true, Rules: []ClassicState{s}}
	prune, err := PrepareClassicPrune(o, nil)
	if err != nil || prune.ValidateApply(false) == nil || prune.ValidateApply(true) != nil {
		t.Fatal(prune, err)
	}
	prune = prune.AfterUnarchive()
	o.Archived = false
	if err = prune.Recheck(o); err != nil {
		t.Fatal(err)
	}
	candidates, _ := prune.Candidates()
	if candidates[0].VerifyTarget("R", false, false) == nil {
		t.Fatal("fork detachment accepted after delete")
	}
	o.IsFork, o.Rules[0].IsFork, o.Rules[0].Rule.AllowForkSyncing = false, false, nil
	if prune.Recheck(o) == nil {
		t.Fatal("hidden target drift accepted")
	}
	copy := CloneClassicRule(s.Rule)
	*copy.AllowForkSyncing = false
	if !*s.Rule.AllowForkSyncing {
		t.Fatal("projection aliases")
	}
}

func TestClassicForkSyncClosedReport(t *testing.T) {
	s := forkModelState(true, true, false)
	yes := true
	d := ClassicRule{ID: s.Rule.ID, BranchNamePattern: "main", LockBranch: &yes, AllowForkSyncing: &yes}
	change := ClassicChange{Before: &s.Rule, After: &d}
	data, _ := json.Marshal(change)
	var decoded ClassicChange
	if err := json.Unmarshal(data, &decoded); err != nil || ValidateClassicChanges([]ClassicChange{decoded}) != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(change, decoded) {
		t.Fatal(decoded)
	}
	for _, pair := range [][2]string{{`"allow_fork_syncing":true`, `"allow_fork_syncing":null`}, {`"allow_fork_syncing":true`, `"Allow_fork_syncing":true`}, {`"allow_fork_syncing":true`, `"allow_fork_syncing":"true"`}, {`"lock_branch":true`, `"lock_branch":false`}} {
		bad := strings.ReplaceAll(string(data), pair[0], pair[1])
		err := json.Unmarshal([]byte(bad), &decoded)
		if err == nil {
			err = ValidateClassicChanges([]ClassicChange{decoded})
		}
		if err == nil {
			t.Fatal("invalid fork report accepted", bad)
		}
	}
}
