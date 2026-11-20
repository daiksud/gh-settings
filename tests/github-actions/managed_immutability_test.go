package githubactions

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/workspace"
)

// Test-only projection of the pinned operational declaration, not a product
// importer or alias. Remove when a compatible immutable release is adopted.
func managedImmutabilityCandidate(t *testing.T, data []byte) []byte {
	t.Helper()
	if operationalAction != "daiksud/gh-settings@b074b1de2e7cbc7f623cc0e31bcab71d681d166d" {
		t.Fatal("reassess the published-package key boundary")
	}
	if _, err := workspace.DecodeConfig(data); err == nil || !strings.Contains(err.Error(), "immutable_release") {
		t.Fatalf("expected obsolete key rejection: %v", err)
	}
	old, next := []byte("        immutable_release: true\n"), []byte("        enable_release_immutability: true\n")
	if bytes.Count(data, old) != 1 || bytes.Contains(data, []byte("enable_release_immutability:")) {
		t.Fatal("unexpected operational release declaration")
	}
	return bytes.Replace(data, old, next, 1)
}

func TestManagedImmutabilityCandidatePreservesPolicy(t *testing.T) {
	original, err := os.ReadFile(filepath.Join(root(), ".github/settings.yml"))
	if err != nil {
		t.Fatal(err)
	}
	candidate := managedImmutabilityCandidate(t, original)
	if !bytes.Equal(bytes.Replace(candidate, []byte("enable_release_immutability:"), []byte("immutable_release:"), 1), original) {
		t.Fatal("additional operational bytes changed")
	}
	d := readManagedDeclaration(t)
	if d.Security.ImmutableReleases == nil || !*d.Security.ImmutableReleases {
		t.Fatal("retain observed release immutability")
	}
}
