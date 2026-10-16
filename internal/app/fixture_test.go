package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/daiksud/gh-settings/internal/workspace"
)

func appCheckout(t *testing.T, input string, origin bool) string {
	t.Helper()
	root := t.TempDir()
	if output, err := exec.Command("git", "init", "--quiet", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	if origin {
		if output, err := exec.Command("git", "-C", root, "remote", "add", "origin", "https://github.com/owner/current.git").CombinedOutput(); err != nil {
			t.Fatalf("git origin: %v: %s", err, output)
		}
	}
	if err := os.Mkdir(filepath.Join(root, ".github"), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, workspace.SettingsPath)
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	return path
}
