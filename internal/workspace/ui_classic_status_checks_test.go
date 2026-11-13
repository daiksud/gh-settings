package workspace

import (
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/github"
)

func TestUIClassicStatusChecksLiteral(t *testing.T) {
	input := "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    branch_protection_rules:\n      - branch_name_pattern: main\n        require_status_checks_to_pass_before_merging:\n          enabled: true\n          require_branches_to_be_up_to_date_before_merging: false\n          status_checks:\n            - name: screening\n              accept_updates_from: any\n"
	model, err := DecodeConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeConfig(model)
	if err != nil || !strings.Contains(string(data), "require_branches_to_be_up_to_date_before_merging: false") || !strings.Contains(string(data), "accept_updates_from: any") {
		t.Fatal("status-check union did not roundtrip", err, string(data))
	}
}

func TestUIClassicStatusChecksStrictUnion(t *testing.T) {
	prefix := "version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: [{branch_name_pattern: main, require_status_checks_to_pass_before_merging: "
	for _, value := range []string{"null", "{enabled: 'true'}", "{enabled: true}", "{enabled: false, status_checks: []}", "{enabled: true, require_branches_to_be_up_to_date_before_merging: false, status_checks: null}", "{enabled: true, require_branches_to_be_up_to_date_before_merging: false, status_checks: [{name: x, accept_updates_from: 123}]}", "{enabled: true, require_branches_to_be_up_to_date_before_merging: false, status_checks: [{name: x, app_id: any}]}", "{requires_status_checks: true}"} {
		model, err := DecodeConfig([]byte(prefix + value + "}]}}\n"))
		if err == nil {
			_, err = github.DeclarationFromUI(model)
		}
		if err == nil {
			t.Fatal("invalid status group accepted", value)
		}
	}
	for _, value := range []string{"{}", "{enabled: false}", "{enabled: true, require_branches_to_be_up_to_date_before_merging: false, status_checks: []}"} {
		model, err := DecodeConfig([]byte(prefix + value + "}]}}\n"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := github.DeclarationFromUI(model); err != nil {
			t.Fatal(err)
		}
	}
}
