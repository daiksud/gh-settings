package main

import (
	"bytes"
	"encoding/json"
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
	t.Helper()
	f := fixture{root: t.TempDir()}
	mustGit(t, f.root, "init", "--quiet")
	mustGit(t, f.root, "config", "user.name", "Fixture")
	mustGit(t, f.root, "config", "user.email", "fixture@example.invalid")
	mustGit(t, f.root, "config", "commit.gpgsign", "false")
	write(t, filepath.Join(f.root, "action/cli-version"), "v1.2.3\n")
	write(t, filepath.Join(f.root, ".github/release.yml"), "changelog:\n  categories:\n    - title: Chore\n      labels: ['*']\n")
	mustGit(t, f.root, "add", ".")
	mustGit(t, f.root, "commit", "--quiet", "-m", "chore: baseline")
	f.base = mustGit(t, f.root, "rev-parse", "HEAD")
	mustGit(t, f.root, "-c", "tag.gpgsign=false", "tag", "v1.2.3")
	for i, message := range messages {
		write(t, filepath.Join(f.root, "change.txt"), fmt.Sprintf("change %d\n", i))
		mustGit(t, f.root, "add", "change.txt")
		mustGit(t, f.root, "commit", "--quiet", "-m", message)
	}
	f.source = mustGit(t, f.root, "rev-parse", "HEAD")
	f.catalog = `[[{"id":1,"tag_name":"v1.2.3","draft":false,"prerelease":false,"published_at":"2026-10-01T00:00:00Z"}]]`
	return f
}

func TestVersionTiersAndCompatibilityFooters(t *testing.T) {
	for _, tt := range []struct {
		message      string
		major, minor bool
	}{
		{"fix: correct behavior", false, false},
		{"feat(settings): expose a field", false, true},
		{"perf: reduce runtime work", false, true},
		{"ci: accelerate tests", false, false},
		{"build(release): prepare next version", false, false},
		{"revert: remove a feature", false, false},
		{"refactor!: replace the API", true, false},
		{"fix: correction\n\nBREAKING CHANGE: Migrate A.\n\nSecond paragraph.\n\nRefs #123", true, false},
		{"fix: correction\n\nBREAKING-CHANGE: Historical marker.", true, false},
		{"chore: notice\n\nDEPRECATED: Migrate before removal.", false, true},
		{"docs: example\n\n```text\nBREAKING CHANGE: Not a real footer.\nDEPRECATED: Not a real footer.\n```", false, false},
		{"docs: example\n\n~~~text\nBREAKING CHANGE: Not a real footer.\n~~~~\n\nBREAKING CHANGE: Actual footer.", true, false},
		{"legacy maintenance", false, false},
	} {
		t.Run(strings.ReplaceAll(tt.message, "\n", " "), func(t *testing.T) {
			c := parseChange(strings.Repeat("a", 40), tt.message)
			if c.major != tt.major || c.minor != tt.minor {
				t.Fatalf("%+v", c)
			}
		})
	}
	c := parseChange(strings.Repeat("a", 40), "fix: correction\n\nBREAKING CHANGE: Migrate A.\n\nSecond paragraph.\n\nRefs #123")
	if !reflect.DeepEqual(c.breaking, []string{"Migrate A.\n\nSecond paragraph."}) {
		t.Fatal(c.breaking)
	}
	for _, tt := range []struct {
		version string
		tier    int
		want    string
	}{
		{"v0.17.0", 2, "v1.0.0"}, {"v0.17.0", 1, "v0.18.0"}, {"v0.17.0", 0, "v0.17.1"},
		{"v1.2.3", 2, "v2.0.0"}, {"v1.2.3", 1, "v1.3.0"}, {"v1.2.3", 0, "v1.2.4"},
	} {
		got, err := bump(tt.version, tt.tier)
		if err != nil || got != tt.want {
			t.Fatalf("%s %d: %s %v", tt.version, tt.tier, got, err)
		}
	}
	for _, invalid := range []string{"v01.2.3", "v1.2.3-rc.1", "v1.2.3+build", "1.2.3", "v1.2.18446744073709551615", "v1.2.18446744073709551616"} {
		if _, err := bump(invalid, 0); err == nil {
			t.Fatal("accepted", invalid)
		}
	}
}

func TestProposalUsesEntireFrozenPublishedInterval(t *testing.T) {
	f := newFixture(t, "feat: earlier feature", "fix: later correction")
	mustGit(t, f.root, "tag", "v99.0.0")
	write(t, filepath.Join(f.root, "later.txt"), "later")
	mustGit(t, f.root, "add", ".")
	mustGit(t, f.root, "commit", "--quiet", "-m", "feat!: not part of the frozen source")
	p, err := propose(f.root, f.source, strings.NewReader(f.catalog), false)
	if err != nil || p.version != "v1.3.0" || p.previousTag != "v1.2.3" {
		t.Fatalf("%+v %v", p, err)
	}
	// Published order, not the numerically highest version, chooses the boundary.
	mustGit(t, f.root, "-c", "tag.gpgsign=false", "tag", "-a", "v1.1.9", f.base, "-m", "annotated")
	catalog := strings.TrimSuffix(f.catalog, "]]") + `,{"id":2,"tag_name":"v99.0.0","draft":true,"prerelease":false},{"id":3,"tag_name":"v1.1.9","draft":false,"prerelease":false,"published_at":"2026-10-02T00:00:00Z"}]]`
	p, err = propose(f.root, f.source, strings.NewReader(catalog), false)
	if err != nil || p.version != "v1.2.0" || p.previousTag != "v1.1.9" {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestProposalPrecedenceAndVersionPreparation(t *testing.T) {
	for _, tt := range []struct {
		messages []string
		want     string
	}{
		{[]string{"fix: correct behavior", "perf: reduce work"}, "v1.3.0"},
		{[]string{"feat: new behavior", "refactor!: incompatible behavior"}, "v2.0.0"},
		{[]string{"ci: faster validation", "docs: explain usage"}, "v1.2.4"},
		{[]string{"feat: add capability", "revert: revert capability"}, "v1.3.0"},
		{[]string{"fix: maintenance\n\nDEPRECATED: Retire this API later."}, "v1.3.0"},
	} {
		f := newFixture(t, tt.messages...)
		p, err := propose(f.root, f.source, strings.NewReader(f.catalog), false)
		if err != nil || p.version != tt.want {
			t.Fatalf("%+v %v", p, err)
		}
	}
	f := newFixture(t)
	write(t, filepath.Join(f.root, "action/cli-version"), "v1.2.4\n")
	mustGit(t, f.root, "add", ".")
	mustGit(t, f.root, "commit", "--quiet", "-m", "build(release): prepare next version")
	if _, err := propose(f.root, mustGit(t, f.root, "rev-parse", "HEAD"), strings.NewReader(f.catalog), false); err == nil {
		t.Fatal("version-only release accepted")
	}
	if _, err := propose(f.root, f.base, strings.NewReader(f.catalog), false); err == nil {
		t.Fatal("empty interval accepted")
	}
}

func TestCatalogAndHistoryFailures(t *testing.T) {
	f := newFixture(t, "fix: corrected behavior")
	for _, catalog := range []string{"null", "[]", "[null]", "{}", "[[]] true", `[[{"id":1,"tag_name":"v1.2.3"}]]`, `[[{"id":1,"tag_name":"v1.2.3","draft":false,"prerelease":false}]]`, strings.ReplaceAll(f.catalog, "false", "null"), strings.TrimSuffix(f.catalog, "]]") + "," + strings.TrimPrefix(f.catalog, "[[")} {
		if _, err := propose(f.root, f.source, strings.NewReader(catalog), false); err == nil {
			t.Fatal("accepted catalog", catalog)
		}
	}
	if _, err := propose(f.root, f.source, strings.NewReader("[[]]"), false); err == nil {
		t.Fatal("silently invented an initial release")
	}
	if _, err := propose(f.root, f.source, strings.NewReader(strings.ReplaceAll(f.catalog, "v1.2.3", "v1.2.2")), false); err == nil {
		t.Fatal("missing predecessor accepted")
	}
	if _, err := propose(f.root, "main", strings.NewReader(f.catalog), false); err == nil {
		t.Fatal("mutable source accepted")
	}
	write(t, filepath.Join(f.root, ".git/shallow"), f.base+"\n")
	if _, err := propose(f.root, f.source, strings.NewReader(f.catalog), false); err == nil {
		t.Fatal("shallow history accepted")
	}
}

func TestPrepareResolveAndRetryWithoutRemoteMutation(t *testing.T) {
	f := newFixture(t, "feat: new capability")
	calls := 0
	gh := func(args ...string) ([]byte, error) {
		calls++
		want := []string{"api", "repos/owner/repo/releases?per_page=100", "--paginate", "--slurp"}
		if !reflect.DeepEqual(args, want) {
			t.Fatalf("unexpected remote operation: %v", args)
		}
		return []byte(f.catalog), nil
	}
	var output bytes.Buffer
	if err := run(f.root, []string{"resolve", "owner/repo", f.source}, &output, gh); err == nil || output.Len() != 0 {
		t.Fatal("unprepared version resolved")
	}
	if err := run(f.root, []string{"prepare", "owner/repo"}, &output, gh); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(f.root, "action/cli-version"))
	if err != nil || string(data) != "v1.3.0\n" {
		t.Fatal(string(data), err)
	}
	if mustGit(t, f.root, "tag", "--list") != "v1.2.3" {
		t.Fatal("preparation created a tag")
	}
	mustGit(t, f.root, "add", "action/cli-version")
	mustGit(t, f.root, "commit", "--quiet", "-m", "build(release): prepare next version")
	source := mustGit(t, f.root, "rev-parse", "HEAD")
	output.Reset()
	if err := run(f.root, []string{"resolve", "owner/repo", source}, &output, gh); err != nil {
		t.Fatal(err)
	}
	if output.String() != "version=v1.3.0\nprevious-tag=v1.2.3\nprevious-sha="+f.base+"\nsource-sha="+source+"\n" {
		t.Fatal(output.String())
	}
	if calls != 3 {
		t.Fatal(calls)
	}
	// A matching draft or published release is a retry, not another version bump.
	mustGit(t, f.root, "tag", "v1.3.0")
	for _, draft := range []bool{true, false} {
		marker := fmt.Sprintf(`<!-- gh-settings-release: {"version":"v1.3.0","commit":%q,"macos_sha256":%q,"linux_sha256":%q,"checksums_sha256":%q} -->`+"\nStored notes", source, strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64))
		record := map[string]any{"id": 2, "tag_name": "v1.3.0", "draft": draft, "prerelease": false, "published_at": "2026-10-02T00:00:00Z", "body": marker}
		encoded, _ := json.Marshal(record)
		catalog := strings.TrimSuffix(f.catalog, "]]") + "," + string(encoded) + "]]"
		p, err := propose(f.root, source, strings.NewReader(catalog), true)
		if err != nil || !p.reuse || p.version != "v1.3.0" {
			t.Fatalf("retry: %+v %v", p, err)
		}
		bad := strings.ReplaceAll(catalog, strings.Repeat("a", 64), "invalid")
		p, err = propose(f.root, source, strings.NewReader(bad), true)
		if err == nil && p.reuse {
			t.Fatal("invalid manifest reused")
		}
	}
}

func TestPreparePreservesDirtyOrUnsafeFilesAndAPIFailure(t *testing.T) {
	f := newFixture(t, "fix: correction")
	write(t, filepath.Join(f.root, "action/cli-version"), "user work\n")
	noAPI := func(...string) ([]byte, error) { t.Fatal("dirty work reached API"); return nil, nil }
	var output bytes.Buffer
	if err := prepare(f.root, "owner/repo", &output, noAPI); err == nil {
		t.Fatal("dirty work overwritten")
	}
	data, _ := os.ReadFile(filepath.Join(f.root, "action/cli-version"))
	if string(data) != "user work\n" {
		t.Fatal("dirty work lost")
	}
	mustGit(t, f.root, "checkout", "--", "action/cli-version")
	failure := func(...string) ([]byte, error) { return nil, errors.New("fixture unavailable") }
	if err := run(f.root, []string{"resolve", "owner/repo", f.source}, &output, failure); err == nil || output.Len() != 0 {
		t.Fatal("failed API emitted metadata")
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
	if _, err := propose(f.root, f.source, strings.NewReader(catalog), false); err == nil {
		t.Fatal("unrelated predecessor accepted")
	}
}
