package workspace

import (
	"reflect"
	"strings"
	"testing"
)

func TestEnvironmentConfigDefaultsAndRoundTrip(t *testing.T) {
	for _, input := range []string{
		"schemaVersion: 1",
		"schemaVersion: 1\nsettings: {environments: []}",
		"schemaVersion: 1\nsettings: {environments: [{name: production}]}",
		"schemaVersion: 1\nsettings: {environments: [{name: 'preview/a%20', wait_timer: 30, prevent_self_review: true, reviewers: [{type: User, id: 7}], deployment_branch_policy: {protected_branches: true, custom_branch_policies: false}}]}",
		"schemaVersion: 1\nsettings: {environments: [{name: production, deployment_branch_policy: null}]}",
	} {
		d, err := DecodeConfig([]byte(input))
		if err != nil {
			t.Fatal(input, err)
		}
		encoded, err := EncodeConfig(d)
		if err != nil {
			t.Fatal(err)
		}
		round, err := DecodeConfig(encoded)
		if err != nil || !reflect.DeepEqual(d, round) {
			t.Fatalf("roundtrip %s: %v", encoded, err)
		}
		if (d.Environments != nil) != strings.Contains(input, "environments") {
			t.Fatal("lost presence")
		}
		if strings.Contains(input, "[{name: production}]") {
			p := d.Environments.Values()[0]
			if *p.WaitTimer != 0 || *p.PreventSelfReview || len(*p.Reviewers) != 0 || p.DeploymentBranchPolicy != nil {
				t.Fatal("whole-object defaults not applied", p)
			}
		}
	}
}

func TestEnvironmentConfigRejectsIDsNullAndUnsupportedPolicies(t *testing.T) {
	for _, value := range []string{
		"null", "{}", "[null]", "[{}]", "[{name: 7}]", "[{name: null}]", "[{name: a, id: 7}]",
		"[{name: a, wait_timer: null}]", "[{name: a, wait_timer: '0'}]", "[{name: a, wait_timer: 0.5}]",
		"[{name: a, prevent_self_review: null}]", "[{name: a, prevent_self_review: 'false'}]",
		"[{name: a, reviewers: null}]", "[{name: a, reviewers: [null]}]", "[{name: a, reviewers: [{type: User, id: '7'}]}]",
		"[{name: a, reviewers: [{type: User, id: 7, login: ignored}]}]",
		"[{name: a, deployment_branch_policy: {protected_branches: true}}]",
		"[{name: a, deployment_branch_policy: {protected_branches: null, custom_branch_policies: false}}]",
		"[{name: a, deployment_branch_policy: {protected_branches: false, custom_branch_policies: true}}]",
		"[{name: Production}, {name: production}]", "[{name: a, name: b}]", "&x [*x]",
	} {
		if _, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {environments: " + value + "}\n")); err == nil {
			t.Fatal("accepted", value)
		}
	}
}
