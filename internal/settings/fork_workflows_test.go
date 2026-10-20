package settings

import "testing"

func forkBool(v bool) *bool { return &v }
func TestForkPolicyObservationsAndExplicitPrerequisites(t *testing.T) {
	if _, err := (ForkApprovalObservation{}).Value(); err == nil {
		t.Fatal("zero approval acquired")
	}
	if _, err := (PrivateForkObservation{}).Values(); err == nil {
		t.Fatal("zero private acquired")
	}
	for _, v := range []string{"first_time_contributors_new_to_github", "first_time_contributors", "all_external_contributors"} {
		o, err := NewForkApprovalObservation(v)
		if err != nil {
			t.Fatal(err)
		}
		m, c, err := CompareForkApproval(o, ActionsPolicyDeclaration{ForkApprovalPolicy: &v})
		if err != nil || m != 1 || len(c) != 0 {
			t.Fatal(m, c, err)
		}
	}
	for _, v := range []string{"", "never", "ALL_EXTERNAL_CONTRIBUTORS"} {
		if _, err := NewForkApprovalObservation(v); err == nil {
			t.Fatal(v)
		}
	}
	o := NewPrivateForkObservation([4]bool{true, false, true, false})
	for _, test := range []struct {
		d                PrivateForkDeclaration
		managed, changes int
		valid            bool
	}{
		{PrivateForkDeclaration{}, 0, 0, true},
		{PrivateForkDeclaration{WriteTokens: forkBool(false)}, 1, 0, true},
		{PrivateForkDeclaration{Secrets: forkBool(false)}, 0, 0, false},
		{PrivateForkDeclaration{Run: forkBool(true), Secrets: forkBool(false)}, 2, 1, true},
		{PrivateForkDeclaration{Run: forkBool(false)}, 1, 1, true},
	} {
		m, c, err := ComparePrivateFork(o, test.d)
		if (err == nil) != test.valid || test.valid && (m != test.managed || len(c) != test.changes) {
			t.Fatal(test, m, c, err)
		}
	}
	d := PrivateForkDeclaration{Run: forkBool(true)}
	if _, _, err := ComparePrivateFork(NewPrivateForkObservation([4]bool{false, true, true, false}), d); err == nil {
		t.Fatal("latent credential settings implicitly activated")
	}
	d.WriteTokens = forkBool(false)
	d.Secrets = forkBool(false)
	d.Approval = forkBool(true)
	if m, c, err := ComparePrivateFork(NewPrivateForkObservation([4]bool{false, true, true, false}), d); err != nil || m != 4 || len(c) != 4 {
		t.Fatal(m, c, err)
	}
	round, err := o.Declaration()
	if err != nil {
		t.Fatal(err)
	}
	if m, c, err := ComparePrivateFork(o, round); err != nil || m != 4 || len(c) != 0 {
		t.Fatal(m, c, err)
	}
	whole := ActionsPolicyDeclaration{PrivateFork: round}
	if whole.Empty() || !whole.ExecutionEmpty() || !whole.WorkflowPermissionsEmpty() || !whole.RetentionEmpty() {
		t.Fatal("family leaked", whole)
	}
}
func TestForkComparisonsUseClosedNativeValues(t *testing.T) {
	fields := []string{"actions.fork_pr_approval_policy"}
	for _, k := range PrivateForkKeys() {
		fields = append(fields, "actions.private_fork_workflows."+k)
	}
	for i, f := range fields {
		var b, a any = false, true
		if i == 0 {
			b, a = "first_time_contributors", "all_external_contributors"
		}
		if _, err := NewActionsPolicyChange(f, b, a); err != nil {
			t.Fatal(f, err)
		}
		for _, bad := range []any{nil, 1, "false", []string{}} {
			if _, err := NewActionsPolicyChange(f, b, bad); err == nil {
				t.Fatal(f, bad)
			}
		}
		if _, err := NewActionsPolicyChange(f, nil, a); err == nil {
			t.Fatal("missing before", f)
		}
	}
}
