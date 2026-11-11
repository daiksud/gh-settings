package workspace

import (
	"os"
	"strings"
	"testing"
)

func TestCreatedIDActionsPolicyWriteback(t *testing.T) {
	input := "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    actions:\n      policies:\n        - policy_name: main # keep comment\n          enforcement_status: disabled\n          restrict_events: {allowed_events: [push]}\n        - policy_name: later\n          enforcement_status: active\n          restrict_events: {allowed_events: [workflow_dispatch]}\n"
	root, path := writebackFixture(t, input)
	record := createdIDRecorder(root, []byte(input))
	if err := record("actions_policies", "main", 31); err != nil {
		t.Fatal(err)
	}
	if err := record("actions_policies", "later", 32); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeConfig(data)
	if err != nil {
		t.Fatal(err)
	}
	policies := *got.Settings.Actions.Policies
	if *policies[0].ID != 31 || *policies[1].ID != 32 || *policies[0].EnforcementStatus != "disabled" || !strings.Contains(string(data), "# keep comment") {
		t.Fatalf("writeback changed policy intent: %s", data)
	}
}

func TestCreatedIDActionsPolicyWritebackGuardsPreserveSnapshot(t *testing.T) {
	for _, test := range []struct {
		name, input string
		edit        bool
	}{
		{"existing ID", "repository: {settings: {actions: {policies: [{policy_name: main, id: 7}]}}}\n", false},
		{"missing object", "repository: {settings: {actions: {policies: [{policy_name: other}]}}}\n", false},
		{"obsolete name alias", "repository: {settings: {actions: {policies: [{name: main}]}}}\n", false},
		{"ambiguous name", "repository: {settings: {actions: {policies: [{policy_name: main}, {policy_name: main}]}}}\n", false},
		{"aliased collection", "value: &items [{policy_name: main}]\nrepository: {settings: {actions: {policies: *items}}}\n", false},
		{"aliased entry", "value: &item {policy_name: main}\nrepository: {settings: {actions: {policies: [*item]}}}\n", false},
		{"aliased parent", "value: &actions {policies: [{policy_name: main}]}\nrepository: {settings: {actions: *actions}}\n", false},
		{"merged entry", "repository: {settings: {actions: {policies: [{policy_name: main, <<: {enforcement_status: active}}]}}}\n", false},
		{"concurrent edit", "repository: {settings: {actions: {policies: [{policy_name: main}]}}}\n", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, path := writebackFixture(t, test.input)
			record := createdIDRecorder(root, []byte(test.input))
			want := test.input
			if test.edit {
				want += "# user edit\n"
				if err := os.WriteFile(path, []byte(want), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := record("actions_policies", "main", 31); err == nil {
				t.Fatal("unsafe policy writeback succeeded")
			}
			if data, err := os.ReadFile(path); err != nil || string(data) != want {
				t.Fatalf("failure changed snapshot: %s, %v", data, err)
			}
		})
	}
}
