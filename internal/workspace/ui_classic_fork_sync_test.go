package workspace

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/github"
)

func TestClassicForkSyncLiteralCodec(t *testing.T) {
	for _, value := range []string{"true", "false"} {
		data := []byte("version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: [{id: 7, branch_name_pattern: main, lock_branch: true, allow_fork_syncing: " + value + "}]}}\n")
		ui, err := DecodeConfig(data)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = github.DeclarationFromUI(ui); err != nil {
			t.Fatal(err)
		}
		encoded, err := EncodeConfig(ui)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeConfig(encoded)
		if err != nil || !reflect.DeepEqual(ui, again) {
			t.Fatal(string(encoded), err)
		}
	}
}

func TestClassicForkSyncRejectsInvalidDependencies(t *testing.T) {
	for _, fields := range []string{"allow_fork_syncing: true", "allow_fork_syncing: false", "lock_branch: false, allow_fork_syncing: false", "lock_branch: false, allow_fork_syncing: true", "lock_branch: true, allow_fork_syncing: null", "lock_branch: true, allow_fork_syncing: 'true'", "lock_branch: true, Allow_fork_syncing: true"} {
		ui, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: [{branch_name_pattern: main, " + fields + "}]}}\n"))
		if err == nil {
			_, err = github.DeclarationFromUI(ui)
		}
		if err == nil {
			t.Fatal("invalid fork declaration accepted", fields)
		}
	}
}
