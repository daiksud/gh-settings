package workspace

import (
	"github.com/daiksud/gh-settings/internal/github"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestClassicProtectionLiteralCodec(t *testing.T) {
	const prefix = "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n"
	input := prefix + "    branch_protection_rules:\n      - id: 7\n        branch_name_pattern: 'Release/**/*'\n        require_conversation_resolution_before_merging: false\n        require_signed_commits: true\n        require_linear_history: false\n        allow_deletions: true\n"
	ui, err := DecodeConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	d, err := github.DeclarationFromUI(ui)
	if err != nil {
		t.Fatal(err)
	}
	p := (*d.BranchProtectionRules)[0]
	if *p.ID != 7 || p.BranchNamePattern != "Release/**/*" || *p.RequireConversationResolutionBeforeMerging || !*p.RequireSignedCommits || *p.RequireLinearHistory || !*p.AllowDeletions {
		t.Fatal(p)
	}
	data, err := EncodeConfig(ui)
	if err != nil {
		t.Fatal(err)
	}
	again, err := DecodeConfig(data)
	if err != nil || !reflect.DeepEqual(ui, again) {
		t.Fatal(string(data), err)
	}
	for _, item := range []string{"{branch_name_pattern: null}", "{branch_name_pattern: true}", "{branch_name_pattern: ''}", "{branch_name_pattern: '   '}", "{branch_name_pattern: main, id: 0}", "{branch_name_pattern: main, id: 1.5}", "{branch_name_pattern: main, allow_deletions: null}", "{branch_name_pattern: main, allow_deletions: 'false'}", "{branch_name_pattern: main, is_admin_enforced: true}", "{branch_name_pattern: main, pattern: main}", "{branch_name_pattern: main, delete: true}", "{branch_name_pattern: main, require_signed_commits: 1}"} {
		ui, err := DecodeConfig([]byte(prefix + "    branch_protection_rules: [" + item + "]\n"))
		if err == nil {
			_, err = github.DeclarationFromUI(ui)
		}
		if err == nil {
			t.Fatal("accepted", item)
		}
	}
	for _, member := range []string{"{}", "{branch_protection_rules: []}"} {
		ui, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: " + member + "}\n"))
		if err != nil || (ui.Settings.BranchProtectionRules != nil) != strings.Contains(member, "[]") {
			t.Fatal(ui, err)
		}
	}
}

func TestClassicCreatedIDRecorderUsesPatternAndSnapshot(t *testing.T) {
	root := t.TempDir()
	input := []byte("version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: [{branch_name_pattern: 'z/**'}, {branch_name_pattern: 'z/*'}]}}\n")
	if err := os.Mkdir(filepath.Join(root, ".github"), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, SettingsPath)
	if err := os.WriteFile(path, input, 0600); err != nil {
		t.Fatal(err)
	}
	record := createdIDRecorder(root, input)
	if err := record("branch_protection_rules", "z/**", 17); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	ui, err := DecodeConfig(data)
	if err != nil || *(*ui.Settings.BranchProtectionRules)[0].ID != 17 || (*ui.Settings.BranchProtectionRules)[1].ID != nil {
		t.Fatal(string(data), err)
	}
	edited := append(data, []byte("# concurrent edit\n")...)
	if err := os.WriteFile(path, edited, 0600); err != nil {
		t.Fatal(err)
	}
	if err := record("branch_protection_rules", "z/*", 18); err == nil {
		t.Fatal("overwrote concurrent edit")
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(edited) {
		t.Fatal(string(after))
	}
}
