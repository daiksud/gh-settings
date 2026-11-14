package settings

import (
	"strings"
	"testing"
)

func runnerDesired(id int64, names ...string) RunnerLabels {
	if names == nil {
		names = []string{}
	}
	return RunnerLabels{ID: id, CustomLabels: &names}
}
func TestRunnerLabelsOwnershipAndIdentity(t *testing.T) {
	for _, v := range []RunnerLabels{runnerDesired(1), runnerDesired(23, "GPU", "K"), runnerDesired(2, " leading ")} {
		if e := ValidateRunnerLabels([]RunnerLabels{v}); e != nil {
			t.Fatal(e)
		}
	}
	for _, values := range [][]RunnerLabels{{runnerDesired(0)}, {runnerDesired(-1)}, {runnerDesired(1), runnerDesired(1)}, {{ID: 1}}, {runnerDesired(1, "")}, {runnerDesired(1, " \t")}, {runnerDesired(1, "a\nb")}, {runnerDesired(1, "gpu", "GPU")}, {runnerDesired(1, "K", "K")}, {runnerDesired(1, strings.Split(strings.Repeat("x,", 100), ",")...)}} {
		if ValidateRunnerLabels(values) == nil {
			t.Fatal("accepted", values)
		}
	}
	a, b := runnerDesired(1, "GPU", "K"), runnerDesired(1, "k", "gpu")
	if !RunnerLabelsMatch(a, b) {
		t.Fatal("case/order mismatch")
	}
	o, e := NewRunnerObservation([]RunnerState{{Value: a, ReadOnlyLabels: []string{"self-hosted"}}, {Value: runnerDesired(2)}})
	if e != nil {
		t.Fatal(e)
	}
	managed, changes, e := CompareRunnerLabels(o, []RunnerLabels{b})
	if e != nil || managed != 1 || len(changes) != 0 {
		t.Fatal(managed, changes, e)
	}
	if _, _, e = CompareRunnerLabels(o, []RunnerLabels{runnerDesired(3)}); e == nil {
		t.Fatal("missing runner accepted")
	}
	if PrepareRunnerLabels(o, []RunnerLabels{runnerDesired(1, "SELF-HOSTED")}) == nil {
		t.Fatal("readonly collision accepted")
	}
	clone := CloneRunnerLabels([]RunnerLabels{a})
	(*clone[0].CustomLabels)[0] = "changed"
	v, _ := o.Values()
	if (*v[0].CustomLabels)[0] != "GPU" {
		t.Fatal("observation mutated")
	}
	if _, e = (RunnerObservation{}).Values(); e == nil {
		t.Fatal("unknown exported empty")
	}
	if n, c, e := CompareRunnerLabels(RunnerObservation{}, []RunnerLabels{}); e != nil || n != 0 || len(c) != 0 {
		t.Fatal(n, c, e)
	}
}
