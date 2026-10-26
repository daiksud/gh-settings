package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type fixture struct{ root, base, source, catalog string }

func mustGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	out, err := git(root, args...)
	if err != nil {
		t.Fatal(args, err)
	}
	return strings.TrimSpace(out)
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func newFixture(t *testing.T, messages ...string) fixture {
	return newFixtureVersion(t, "v1.2.3", messages...)
}

func newFixtureVersion(t *testing.T, version string, messages ...string) fixture {
	t.Helper()
	f := fixture{root: t.TempDir()}
	for name, path := range map[string]string{
		"HOME":            filepath.Join(f.root, "home"),
		"XDG_CONFIG_HOME": filepath.Join(f.root, "xdg"),
		"GH_CONFIG_DIR":   filepath.Join(f.root, "gh"),
		"TMPDIR":          filepath.Join(f.root, "tmp"),
	} {
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
		t.Setenv(name, path)
	}
	mustGit(t, f.root, "init", "--quiet")
	mustGit(t, f.root, "config", "user.name", "Fixture")
	mustGit(t, f.root, "config", "user.email", "fixture@example.invalid")
	mustGit(t, f.root, "config", "commit.gpgsign", "false")
	write(t, filepath.Join(f.root, "action/cli-version"), version+"\n")
	write(t, filepath.Join(f.root, ".github/release.yml"), "changelog:\n  categories:\n    - title: Chore\n      labels: ['*']\n")
	mustGit(t, f.root, "add", ".")
	mustGit(t, f.root, "commit", "--quiet", "-m", "chore: baseline")
	f.base = mustGit(t, f.root, "rev-parse", "HEAD")
	mustGit(t, f.root, "-c", "tag.gpgsign=false", "tag", version)
	for i, message := range messages {
		write(t, filepath.Join(f.root, "change.txt"), fmt.Sprintf("change %d\n", i))
		mustGit(t, f.root, "add", "change.txt")
		mustGit(t, f.root, "commit", "--quiet", "-m", message)
	}
	f.source = mustGit(t, f.root, "rev-parse", "HEAD")
	f.catalog = fmt.Sprintf(`[[{"id":1,"tag_name":%q,"draft":false,"prerelease":false,"published_at":"2026-10-01T00:00:00Z"}]]`, version)
	return f
}

func TestReleaseNotesRecognizeCompatibilityFooters(t *testing.T) {
	for _, tt := range []struct {
		message      string
		major, minor bool
	}{
		{"fix: correct behavior", false, false},
		{"feat(settings): expose a field", false, true},
		{"perf: reduce runtime work", false, true},
		{"refactor!: replace the API", true, false},
		{"fix: correction\n\nBREAKING CHANGE: Migrate A.", true, false},
		{"fix: correction\n\nBREAKING-CHANGE: Migrate B.", true, false},
		{"chore: notice\n\nDEPRECATED: Migrate before removal.", false, true},
		{{"docs: example\n\n~~~text\nBREAKING CHANGE: Example only.\n~~~", false, false},
	} {
		t.Run(strings.ReplaceAll(tt.message, "\n", " "), func(t *testing.T) {
			c := parseChange(strings.Repeat("a", 40), tt.message)
			if c.major != tt.major || c.minor != tt.minor {
				t.Fatalf("unexpected release-note classification: %+v", c)
			}
		})
	}
}

func TestNativeGenerationUsesFixedBoundaryAndKeepsMigration(t *testing.T) {
	f := newFixture(t, "feat!: replace <old> behavior\n\nBREAKING CHANGE: Migrate <old> to new.\n\nPreserve all data.\n\nRefs #12")
	write(t, filepath.Join(f.root, "later.txt"), "later")
	mustGit(t, f.root, "add", ".")
	mustGit(t, f.root, "commit", "--quiet", "-m", "feat!: excluded later change")
	calls := 0
	gh := func(args ...string) ([]byte, error) {
		calls++
		want := []string{"api", "-X", "POST", "repos/owner/repo/releases/generate-notes", "-f", "tag_name=v2.0.0", "-f", "target_commitish=" + f.source, "-f", "configuration_file_path=.github/release.yml", "-f", "previous_tag_name=v1.2.3"}
		if !reflect.DeepEqual(args, want) {
			t.Fatalf("unexpected generation: %v", args)
		}
		return []byte(`{"body":"## GitHub-owned changes\n* Reviewed PR #12 by @author\n"}`), nil
	}
	notes, err := generate(f.root, "owner/repo", "v2.0.0", f.source, strings.NewReader(f.catalog), "v1.2.3", f.base, gh)
	if err != nil || calls != 1 {
		t.Fatal(err, calls)
	}
	for _, want := range []string{"## GitHub-owned changes", "Reviewed PR #12", "## ⚠️ Migration guidance", "Migrate &lt;old&gt; to new.", "Preserve all data.", f.base + "..." + f.source} {
		if !strings.Contains(notes, want) {
			t.Fatal("missing", want, notes)
		}
	}
	if strings.Contains(notes, "excluded later") || strings.Contains(notes, "## ✨ New Feature") || strings.Contains(notes, "Refs #12") {
		t.Fatal("custom classification or wrong range", notes)
	}
}

func TestGenerationFailuresNeverCreateReleaseState(t *testing.T) {
	f := newFixture(t, "fix: correction")
	for _, response := range []string{"{}", `{"body":null}`, `{"body":""}`, `{"body":42}`, "not JSON"} {
		calls := 0
		gh := func(args ...string) ([]byte, error) {
			calls++
			if len(args) < 4 || args[3] != "repos/owner/repo/releases/generate-notes" {
				t.Fatal("unexpected operation", args)
			}
			return []byte(response), nil
		}
		if _, err := generate(f.root, "owner/repo", "v1.2.4", f.source, strings.NewReader(f.catalog), "v1.2.3", f.base, gh); err == nil || calls != 1 {
			t.Fatal(response, err, calls)
		}
	}
	noCall := func(...string) ([]byte, error) { t.Fatal("invalid boundary reached GitHub"); return nil, nil }
	if _, err := generate(f.root, "owner/repo", "v1.2.4", f.source, strings.NewReader(f.catalog), "v1.2.2", f.base, noCall); err == nil {
		t.Fatal("changed boundary accepted")
	}
	mustGit(t, f.root, "tag", "v1.2.4", f.base)
	if _, err := generate(f.root, "owner/repo", "v1.2.4", f.source, strings.NewReader(f.catalog), "v1.2.3", f.base, noCall); err == nil {
		t.Fatal("conflicting tag accepted")
	}
	failure := func(...string) ([]byte, error) { return nil, errors.New("unavailable") }
	if _, err := generate(f.root, "owner/repo", "v1.2.5", f.source, strings.NewReader(f.catalog), "v1.2.3", f.base, failure); err == nil {
		t.Fatal("generation failure accepted")
	}
}

func TestSameTagWithChangedPredecessorFails(t *testing.T) {
	f := newFixture(t, "fix: correction")
	noCall := func(...string) ([]byte, error) { t.Fatal("changed predecessor reached notes API"); return nil, nil }
	if _, err := generate(f.root, "owner/repo", "v1.2.4", f.source, strings.NewReader(f.catalog), "v1.2.3", strings.Repeat("f", 40), noCall); err == nil {
		t.Fatal("changed predecessor accepted")
	}
}

func TestUnrelatedPublishedHistoryIsNotSilentlySkipped(t *testing.T) {
	f := newFixture(t, "feat: useful change")
	tree := mustGit(t, f.root, "rev-parse", f.base+"^{tree}")
	unrelated := mustGit(t, f.root, "commit-tree", tree, "-m", "chore: unrelated history")
	mustGit(t, f.root, "tag", "v2.0.0", unrelated)
	catalog := strings.ReplaceAll(f.catalog, "v1.2.3", "v2.0.0")
	releases, err := readCatalog(strings.NewReader(catalog))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := history(f.root, f.source, "", releases); err == nil {
		t.Fatal("unrelated predecessor accepted")
	}
}
