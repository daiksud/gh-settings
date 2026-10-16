package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanOutputFailure(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{"plan"}, brokenWriter{}, &stderr, commandOperations{plan: func() ([]byte, error) { return []byte("No managed repository settings.\n"), nil }})
	if code != 1 || stderr.String() != "gh settings: write output: output unavailable\n" {
		t.Fatalf("status=%d stderr=%q", code, stderr.String())
	}
}

func TestPlanCurrentValidatesFirst(t *testing.T) {
	dir := t.TempDir()
	if output, err := exec.Command("git", "init", "--quiet", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	if err := os.Mkdir(filepath.Join(dir, ".github"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir) // No origin or authentication: validation must win over either error.
	for _, test := range []struct{ input, want string }{
		{"settings: {}\n", "missing schemaVersion"},
		{repositoryYAML("has_issues: [\n"), "decode YAML"},
	} {
		if err := os.WriteFile(filepath.Join(dir, settingsPath), []byte(test.input), 0600); err != nil {
			t.Fatal(err)
		}
		data, err := planCurrent()
		if len(data) != 0 || err == nil || !strings.Contains(err.Error(), "validate "+settingsPath) || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("output=%q error=%v", data, err)
		}
	}
}
