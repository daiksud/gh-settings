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
func managedPublishedCandidate(t *testing.T, data []byte) []byte {
	t.Helper()
	if operationalAction != "daiksud/gh-settings@b074b1de2e7cbc7f623cc0e31bcab71d681d166d" {
		t.Fatal("reassess the published-package key boundary")
	}
	if _, err := workspace.DecodeConfig(data); err == nil || !(strings.Contains(err.Error(), `key "immutable_release"`) || strings.Contains(err.Error(), `key "squash_merge"`) || strings.Contains(err.Error(), `key "query_suite"`)) {
		t.Fatalf("expected obsolete key rejection: %v", err)
	}
	replacements := [][2]string{
		{"        immutable_release: true\n", "        enable_release_immutability: true\n"},
		{"        squash_merge:\n", "        squash_merging:\n"},
		{"          query_suite: extended\n", "          query_suites: extended\n"},
	}
	candidate := append([]byte(nil), data...)
	for _, pair := range replacements {
		if bytes.Count(candidate, []byte(pair[0])) != 1 || bytes.Contains(candidate, []byte(pair[1])) {
			t.Fatal("unexpected operational declaration at published-package boundary")
		}
		candidate = bytes.Replace(candidate, []byte(pair[0]), []byte(pair[1]), 1)
	}
	return candidate
}

func TestManagedImmutabilityCandidatePreservesPolicy(t *testing.T) {
	original, err := os.ReadFile(filepath.Join(root(), ".github/settings.yml"))
	if err != nil {
		t.Fatal(err)
	}
	candidate := managedPublishedCandidate(t, original)
	restored := bytes.Replace(candidate, []byte("enable_release_immutability:"), []byte("immutable_release:"), 1)
	restored = bytes.Replace(restored, []byte("        squash_merging:\n"), []byte("        squash_merge:\n"), 1)
	restored = bytes.Replace(restored, []byte("          query_suites: extended\n"), []byte("          query_suite: extended\n"), 1)
	if !bytes.Equal(restored, original) {
		t.Fatal("additional operational bytes changed")
	}
	d := readManagedDeclaration(t)
	if d.Security.ImmutableReleases == nil || !*d.Security.ImmutableReleases || d.Security.CodeQLConfiguration.QuerySuite == nil || *d.Security.CodeQLConfiguration.QuerySuite != "extended" {
		t.Fatal("retain observed release immutability")
	}
}
