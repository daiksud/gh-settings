package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCommandReadsOnlySettingsFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".github"), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, settingsPath)
	input := []byte(repositoryYAML("description: ''\nhas_issues: false\n"))
	if err := os.WriteFile(path, input, 0600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("git", "init", "--quiet", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Join(dir, "nested")) // No origin or GitHub authentication is available.
	var stdout, stderr bytes.Buffer
	code := run([]string{"validate"}, &stdout, &stderr, commandOperations{export: func() ([]byte, error) {
		t.Fatal("validate must not call export or GitHub")
		return nil, nil
	}, validate: validateCurrent})
	if code != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	unchanged, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(unchanged, input) {
		t.Fatalf("settings changed: %q, error=%v", unchanged, err)
	}
}

func TestValidateCommandMissingFile(t *testing.T) {
	dir := t.TempDir()
	if output, err := exec.Command("git", "init", "--quiet", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	if code := run([]string{"validate"}, &stdout, &stderr, commandOperations{validate: validateCurrent}); code == 0 || stdout.Len() != 0 || !strings.Contains(stderr.String(), settingsPath) {
		t.Fatalf("status=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestValidateCommandPreservesTrailingSpaceInGitRoot(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "repo ")
	if err := os.MkdirAll(filepath.Join(dir, ".github"), 0700); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("git", "init", "--quiet", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	if err := os.WriteFile(filepath.Join(dir, settingsPath), []byte(repositoryYAML("has_issues: false\n")), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	if code := run([]string{"validate"}, &stdout, &stderr, commandOperations{validate: validateCurrent}); code != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
