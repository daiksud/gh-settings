package workspace

import (
	"reflect"
	"testing"
)

func TestIssueCreationPolicyConfigTypesAndOmission(t *testing.T) {
	for _, value := range []string{"ALL", "COLLABORATORS_ONLY"} {
		d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {has_issues: false, issue_creation_policy: " + value + "}}\n"))
		if err != nil {
			t.Fatal(err)
		}
		if d.Repository.IssueCreationPolicy == nil || *d.Repository.IssueCreationPolicy != value || d.Repository.HasIssues == nil || *d.Repository.HasIssues || !d.Repository.PRAccessEmpty() {
			t.Fatal(d)
		}
		encoded, err := EncodeConfig(d)
		if err != nil {
			t.Fatal(err)
		}
		round, err := DecodeConfig(encoded)
		if err != nil || !reflect.DeepEqual(round, d) {
			t.Fatal(string(encoded), err)
		}
	}
	for _, value := range []string{"null", "''", "all", "UNKNOWN", "false", "0", "[]", "{}"} {
		if _, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {issue_creation_policy: " + value + "}}\n")); err == nil {
			t.Fatal("invalid policy accepted", value)
		}
	}
	d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {issue_creation_policy: ALL}}\n"))
	if err != nil || d.Repository.HasIssues != nil || !d.Repository.PRAccessEmpty() {
		t.Fatal("companion setting invented", d, err)
	}
}
