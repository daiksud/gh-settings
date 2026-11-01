package main

import (
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
		message           string
		major, deprecated bool
	}{
		{"fix: correct behavior", false, false},
		{"feat(settings): expose a field", false, false},
		{"perf: reduce runtime work", false, false},
		{"refactor!: replace the API", true, false},
		{"fix: correction\n\nBREAKING CHANGE: Migrate A.", true, false},
		{"fix: correction\n\nBREAKING-CHANGE: Migrate B.", true, false},
		{"chore: notice\n\nDEPRECATED: Migrate before removal.", false, true},
		{"feat!: replace behavior\n\nDEPRECATED: Retire the old form.", true, true},
		{"fix: correction\r\n\r\nBREAKING CHANGE: Migrate CRLF.", true, false},
		{"fix: correction\n\nRefs #12\nBREAKING CHANGE: Migrate after a trailer.", true, false},
		{"docs: prose\n\nMarkers can be discussed inline.\nBREAKING CHANGE: This is body text.", false, false},
		{"docs: prose\n\nMarkers can be discussed inline.\nDEPRECATED: This is body text.", false, false},
		{"docs: examples\n\nBREAKING CHANGE #12\nDEPRECATED #13", false, false},
		{"docs: example\n\n~~~text\nBREAKING CHANGE: Example only.\n~~~", false, false},
		{"docs: example\n\n````text\nDEPRECATED: Example only.\n```\nBREAKING-CHANGE: Still inside the fence.\n````", false, false},
	} {
		t.Run(strings.ReplaceAll(tt.message, "\n", " "), func(t *testing.T) {
			c := parseChange(strings.Repeat("a", 40), tt.message)
			if c.major != tt.major || c.deprecated != tt.deprecated {
				t.Fatalf("unexpected release-note classification: %+v", c)
			}
		})
	}
}

func TestCommittedNotesDoNotNeedGitHubOrReleaseConfiguration(t *testing.T) {
	f := newFixture(t, "feat: expose a supported field")
	tools := filepath.Join(f.root, "tools")
	called := filepath.Join(f.root, "unexpected-gh-call")
	write(t, filepath.Join(tools, "gh"), "#!/bin/sh\nprintf 'called\\n' > \"$GH_CALL_RECORD\"\nexit 99\n")
	if err := os.Chmod(filepath.Join(tools, "gh"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tools+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GH_CALL_RECORD", called)
	t.Setenv("PREVIOUS_TAG", "v1.2.3")
	t.Setenv("PREVIOUS_SHA", f.base)
	catalog := filepath.Join(f.root, "releases.json")
	write(t, catalog, f.catalog)
	var notes strings.Builder
	if err := run(f.root, []string{"owner/repo", "v1.3.0", f.source, catalog}, &notes); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(notes.String(), "### ✨ New Feature") || !strings.Contains(notes.String(), "expose a supported field") {
		t.Fatal("missing committed feature", notes.String())
	}
	if _, err := os.Stat(filepath.Join(f.root, ".github/release.yml")); !os.IsNotExist(err) {
		t.Fatal("fixture unexpectedly has a release configuration", err)
	}
	if _, err := os.Stat(called); !os.IsNotExist(err) {
		t.Fatal("note generation requested GitHub metadata", err)
	}
}

func TestCommittedNotesCategorizeEachChangeOnce(t *testing.T) {
	f := newFixture(t,
		"ci: repair the build pipeline",
		"perf(settings): reduce plan requests",
		"fix: preserve explicit false",
		"feat(settings): expose a supported field",
		"feat: retire an alias\n\nDEPRECATED: Use the current form.",
		"fix!: remove a legacy value\n\nDEPRECATED: Use the new value.",
		"perf: replace the output format\n\nBREAKING-CHANGE: Update consumers.",
		"build: refresh tools",
		"legacy maintenance summary",
		"revert: remove an unsuccessful change",
	)
	commits := strings.Fields(mustGit(t, f.root, "rev-list", "--reverse", f.base+".."+f.source))
	item := func(subject string, index int) string {
		return fmt.Sprintf("- [%s](https://github.com/owner/repo/commit/%s)\n", subject, commits[index])
	}
	want := "## What's Changed\n\n### 💥 BREAKING CHANGE\n\n" +
		item("remove a legacy value", 5) +
		item("replace the output format", 6) +
		"\n### ⚠️ DEPRECATED\n\n" + item("retire an alias", 4) +
		"\n### ✨ New Feature\n\n" + item("expose a supported field", 3) +
		"\n### 🐛 Bug Fix\n\n" + item("preserve explicit false", 2) +
		"\n### ⚡ Performance\n\n" + item("reduce plan requests", 1) +
		"\n### 🛠️ Maintenance\n\n" +
		item("repair the build pipeline", 0) +
		item("refresh tools", 7) +
		item("legacy maintenance summary", 8) +
		item("remove an unsuccessful change", 9)
	notes, err := generate(f.root, "owner/repo", "v1.3.0", f.source, strings.NewReader(f.catalog), "v1.2.3", f.base)
	if err != nil {
		t.Fatal(err)
	}
	changes := strings.SplitN(notes, "\n## ⚠️ Migration guidance", 2)[0]
	if changes != want {
		t.Fatalf("unexpected categorized changes\nwant:\n%s\ngot:\n%s", want, changes)
	}
	if strings.Count(notes, "No migration footer was recorded; review the linked change before upgrading.") != 1 {
		t.Fatal("header-only breaking change lost its guidance", notes)
	}
}

func TestGenerationUsesFixedBoundaryAndKeepsEscapedMigration(t *testing.T) {
	f := newFixture(t, "feat!: replace <old> [API] & `mode` (#12)\n\nBREAKING CHANGE: Migrate <old> to [new](destination).\n\nPreserve *all* data.\n\nRefs #12")
	write(t, filepath.Join(f.root, "later.txt"), "later")
	mustGit(t, f.root, "add", ".")
	mustGit(t, f.root, "commit", "--quiet", "-m", "feat!: excluded later change")
	notes, err := generate(f.root, "owner/repo", "v2.0.0", f.source, strings.NewReader(f.catalog), "v1.2.3", f.base)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"- [replace &lt;old&gt; \\[API\\] &amp; \\`mode\\` (\\#12)](https://github.com/owner/repo/commit/" + f.source + ")",
		"## ⚠️ Migration guidance",
		"Migrate &lt;old&gt; to \\[new\\](destination).",
		"Preserve \\*all\\* data.",
		f.base + "..." + f.source,
	} {
		if !strings.Contains(notes, want) {
			t.Fatal("missing", want, notes)
		}
	}
	if strings.Contains(notes, "excluded later") || strings.Contains(notes, "### ✨ New Feature") || strings.Contains(notes, "Refs #12") {
		t.Fatal("wrong category, range, or footer boundary", notes)
	}
}

func TestHistoryKeepsOnlySubstantiveFirstParentChanges(t *testing.T) {
	f := newFixture(t, "feat: reviewed product behavior")
	mainBranch := mustGit(t, f.root, "branch", "--show-current")
	mustGit(t, f.root, "commit", "--quiet", "--allow-empty", "-m", "feat!: empty marker")
	mustGit(t, f.root, "checkout", "--quiet", "-b", "topic")
	write(t, filepath.Join(f.root, "topic.txt"), "topic change\n")
	mustGit(t, f.root, "add", "topic.txt")
	mustGit(t, f.root, "commit", "--quiet", "-m", "feat: intermediate topic work")
	mustGit(t, f.root, "checkout", "--quiet", mainBranch)
	mustGit(t, f.root, "merge", "--no-ff", "--no-edit", "-m", "fix: reviewed main-line change", "topic")
	f.source = mustGit(t, f.root, "rev-parse", "HEAD")
	catalog, err := readCatalog(strings.NewReader(f.catalog))
	if err != nil {
		t.Fatal(err)
	}
	interval, err := history(f.root, f.source, "", catalog)
	if err != nil {
		t.Fatal(err)
	}
	var subjects []string
	for _, c := range interval.changes {
		subjects = append(subjects, c.subject)
	}
	if want := []string{"reviewed product behavior", "reviewed main-line change"}; !reflect.DeepEqual(subjects, want) {
		t.Fatalf("unexpected main-line changes: got %v, want %v", subjects, want)
	}
}

func TestGenerationRejectsInvalidInputsAndChangedBoundary(t *testing.T) {
	f := newFixture(t, "fix: correction")
	for _, catalog := range []string{"not JSON", "{}", "[]", "[null]", `[[{"id":1}]]`, f.catalog + "{}"} {
		if _, err := generate(f.root, "owner/repo", "v1.2.4", f.source, strings.NewReader(catalog), "v1.2.3", f.base); err == nil {
			t.Fatal("invalid catalog accepted", catalog)
		}
	}
	for _, input := range []struct{ repository, version, source string }{
		{"../owner/repo", "v1.2.4", f.source},
		{"owner/repo", "latest", f.source},
		{"owner/repo", "v1.2.4", "HEAD"},
		{"owner/repo", "v1.2.4", strings.Repeat("f", 40)},
	} {
		if _, err := generate(f.root, input.repository, input.version, input.source, strings.NewReader(f.catalog), "v1.2.3", f.base); err == nil {
			t.Fatal("invalid generation identity accepted", input)
		}
	}
	if _, err := generate(f.root, "owner/repo", "v1.2.4", f.source, strings.NewReader(f.catalog), "v1.2.2", f.base); err == nil {
		t.Fatal("changed boundary accepted")
	}
	mustGit(t, f.root, "tag", "v1.2.4", f.base)
	if _, err := generate(f.root, "owner/repo", "v1.2.4", f.source, strings.NewReader(f.catalog), "v1.2.3", f.base); err == nil {
		t.Fatal("conflicting tag accepted")
	}
	if tags := mustGit(t, f.root, "tag", "--list"); tags != "v1.2.3\nv1.2.4" {
		t.Fatal("generation changed release tags", tags)
	}
}

func TestSameTagWithChangedPredecessorFails(t *testing.T) {
	f := newFixture(t, "fix: correction")
	if _, err := generate(f.root, "owner/repo", "v1.2.4", f.source, strings.NewReader(f.catalog), "v1.2.3", strings.Repeat("f", 40)); err == nil {
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
