package workspace

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestDefaultBranchDeclaration(t *testing.T) {
	for _, branch := range []string{"main", "release/v2", "@", "feature/日本語"} {
		d, err := DecodeConfig([]byte(fmt.Sprintf("schemaVersion: 1\nsettings: {repository: {default_branch: %q}}", branch)))
		if err != nil || d.Empty() || d.Repository.BaseEmpty() || d.Repository.DefaultBranch == nil || *d.Repository.DefaultBranch != branch {
			t.Fatalf("branch=%s declaration=%+v error=%v", branch, d, err)
		}
		data, err := EncodeConfig(d)
		if err != nil {
			t.Fatal(err)
		}
		round, err := DecodeConfig(data)
		if err != nil || !reflect.DeepEqual(round, d) {
			t.Fatalf("round=%+v error=%v", round, err)
		}
	}
	for _, value := range []string{"null", "", "''", "false", "0", "[]", "{}", "'a..b'", "'HEAD'", "'a b'", "'@{-1}'"} {
		if d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {default_branch: " + value + "}}")); err == nil || !d.Empty() {
			t.Fatalf("accepted %q: %+v %v", value, d, err)
		}
	}
	d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {has_issues: false}}"))
	if err != nil || d.Repository.DefaultBranch != nil {
		t.Fatalf("omission=%+v error=%v", d, err)
	}
	data, err := EncodeConfig(d)
	if err != nil || strings.Contains(string(data), "default_branch") {
		t.Fatalf("omitted branch introduced: %s %v", data, err)
	}
}
