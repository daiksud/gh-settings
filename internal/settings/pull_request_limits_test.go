package settings

import "testing"

func TestPullRequestLimitsComparison(t *testing.T) {
	yes, no := true, false
	maximum := 1000
	o, err := NewPullRequestLimitsObservation(PullRequestLimitsValues{false, 1, true})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		d                PullRequestLimitsDeclaration
		managed, changes int
	}{
		{PullRequestLimitsDeclaration{}, 0, 0}, {PullRequestLimitsDeclaration{Enabled: &no}, 1, 0}, {PullRequestLimitsDeclaration{Enabled: &yes, Maximum: &maximum, IncludeDrafts: &no}, 3, 3},
	} {
		m, c, e := ComparePullRequestLimits(o, tc.d)
		if e != nil || m != tc.managed || len(c) != tc.changes {
			t.Fatal(m, c, e)
		}
	}
	if _, _, err := ComparePullRequestLimits(PullRequestLimitsObservation{}, PullRequestLimitsDeclaration{Enabled: &no}); err == nil {
		t.Fatal("unacquired accepted")
	}
	for _, v := range []int{-1, 0, 1001} {
		if (PullRequestLimitsDeclaration{Maximum: &v}).Validate() == nil {
			t.Fatal(v)
		}
	}
	for _, field := range []string{PullRequestLimitEnabledPath, PullRequestLimitMaximumPath, PullRequestLimitDraftsPath, "unknown"} {
		if _, err := NewPullRequestLimitsChange(field, "false", true); err == nil {
			t.Fatal(field)
		}
	}
}
