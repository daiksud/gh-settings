package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyCurrentValidatesAndNoopsLocally(t *testing.T) {
	dir := t.TempDir()
	if output, err := exec.Command("git", "init", "--quiet", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	if err := os.Mkdir(filepath.Join(dir, ".github"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir) // No origin: invalid input and no-op must finish before origin/authentication.
	for _, test := range []struct{ input, want string }{
		{"settings: {}\n", "missing schemaVersion"},
		{repositoryYAML("has_issues: [\n"), "decode YAML"},
		{repositoryYAML("description: null\n"), "must be a string"},
		{repositoryYAML("unknown: false\n"), "unknown setting"},
		{"schemaVersion: 1\n", ""},
		{"schemaVersion: 1\nsettings: {}\n", ""},
		{"schemaVersion: 1\nsettings:\n  repository: {}\n", ""},
	} {
		if err := os.WriteFile(filepath.Join(dir, settingsPath), []byte(test.input), 0600); err != nil {
			t.Fatal(err)
		}
		data, err := applyCurrent()
		if test.want == "" {
			if err != nil || string(data) != "No managed repository settings.\n" {
				t.Fatalf("output=%q error=%v", data, err)
			}
		} else if len(data) != 0 || err == nil || !strings.Contains(err.Error(), "validate "+settingsPath) || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("output=%q error=%v", data, err)
		}
	}
}

func TestApplyOutputFailure(t *testing.T) {
	for _, message := range []string{"Applied repository settings.\n", "No managed repository settings.\n"} {
		var stderr bytes.Buffer
		code := run([]string{"apply"}, brokenWriter{}, &stderr, commandOperations{apply: func() ([]byte, error) { return []byte(message), nil }})
		if code != 1 || stderr.String() != "gh settings: write output: output unavailable\n" {
			t.Fatalf("status=%d stderr=%q", code, stderr.String())
		}
	}
}
