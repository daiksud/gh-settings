package main

import (
	"encoding/json"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateUsesPublishedReleaseAndFrozenSource(t *testing.T) {
	dir := newNotesTestRepository(t)
	initial := notesTestCommit(t, dir, "docs: initialize the repository")
	notesTestGit(t, dir, "tag", "v9.0.0", initial)
	previous := notesTestCommit(t, dir, "fix: already published correction")
	notesTestGit(t, dir, "tag", "-a", "v2.0.0", "-m", "annotated published release", previous)
	feature := notesTestCommit(t, dir, "feat(actions): support repository labels")
	notesTestGit(t, dir, "tag", "v999.0.0", feature)
	performance := notesTestCommit(t, dir, "perf(cli): reduce startup latency")
	notesTestGit(t, dir, "tag", "v99.0.0", performance)
	fix := notesTestCommit(t, dir, "fix(settings): preserve omitted fields")
	notesTestGit(t, dir, "tag", "v98.0.0", fix)
	source := notesTestCommit(t, dir, "ci(release): validate candidate artifacts")
	notesTestGit(t, dir, "tag", "v1.0.0", source)
	later := notesTestCommit(t, dir, "feat: an unreleased change after the frozen source")

	// Publication order can differ from tag version and release creation order.
	// The most recently published stable release is deliberately on page two
	// and has a version greater than the requested version.
	catalog := notesTestCatalog(t,
		[]notesTestRelease{
			{90, "v9.0.0", false, false, "2026-10-01T12:00:00Z"},
			{99, "v99.0.0", true, false, "2026-10-19T12:00:00Z"},
			{98, "v98.0.0", false, true, "2026-10-19T12:00:00Z"},
			{97, "v97.0.0-rc.1", false, false, "2026-10-19T12:00:00Z"},
			{96, "latest", false, false, "2026-10-19T12:00:00Z"},
			{95, "v01.0.0", false, false, "2026-10-19T12:00:00Z"},
			{100, "v1.0.0", false, false, "2026-10-20T12:00:00Z"},
		},
		[]notesTestRelease{{20, "v2.0.0", false, false, "2026-10-10T12:00:00Z"}},
	)
	notes, err := generate(dir, "owner/project", "v1.0.0", source, catalog)
	if err != nil {
		t.Fatal(err)
	}
	for heading, subjects := range map[string][]string{
		"## ✨ New Feature": {"support repository labels", "reduce startup latency"},
		"## 🐛 Bug Fix":     {"preserve omitted fields"},
		"## 🧹 Chore":       {"validate candidate artifacts"},
	} {
		section := notesTestSection(t, notes, heading)
		for _, subject := range subjects {
			if !strings.Contains(section, subject) {
				t.Errorf("%q missing from %q section:\n%s", subject, heading, section)
			}
		}
	}
	for _, sha := range []string{feature, performance, fix, source} {
		if count := strings.Count(notes, "https://github.com/owner/project/commit/"+sha); count != 1 {
			t.Errorf("commit %s appears %d times; want exactly once:\n%s", sha, count, notes)
		}
	}
	for _, excluded := range []string{
		"initialize the repository", "already published correction", "an unreleased change after the frozen source",
		"https://github.com/owner/project/commit/" + initial,
		"https://github.com/owner/project/commit/" + previous,
		"https://github.com/owner/project/commit/" + later,
		"## 🚨 BREAKING CHANGE", "feat(actions):", "perf(cli):", "fix(settings):", "ci(release):",
	} {
		if strings.Contains(notes, excluded) {
			t.Errorf("unexpected content %q:\n%s", excluded, notes)
		}
	}
	for _, wanted := range []string{
		"**actions**", "**cli**", "**settings**", "**release**",
		"https://github.com/owner/project/compare/" + previous + "..." + source,
	} {
		if !strings.Contains(notes, wanted) {
			t.Errorf("missing %q:\n%s", wanted, notes)
		}
	}
}

func TestGenerateClassifiesBreakingChangesAndPreservesMigration(t *testing.T) {
	dir := newNotesTestRepository(t)
	notesTestCommit(t, dir, "docs: initialize the repository")
	notesTestGit(t, dir, "tag", "v1.0.0")
	cases := []struct {
		message, subject, heading string
	}{
		{
			"feat(settings)!: rename the configuration section\n\n## Why\n\nMake the resource explicit.\n\nBREAKING CHANGE: Rename settings to repository.\nExisting declarations must use the new key.\n\nRun validation after migrating.\nRefs: #123",
			"rename the configuration section", "## 🚨 BREAKING CHANGE",
		},
		{
			"docs!: remove obsolete instructions",
			"remove obsolete instructions", "## 🚨 BREAKING CHANGE",
		},
		{
			"fix(cli): require an explicit target\n\nBREAKING-CHANGE: Add the repo option before applying.\nReviewed-by: fixture-reviewer",
			"require an explicit target", "## 🚨 BREAKING CHANGE",
		},
		{
			"BUILD(deps): upgrade the supported runtime\n\nRefs: #456\nBREAKING CHANGE: Install the supported runtime before upgrading.",
			"upgrade the supported runtime", "## 🚨 BREAKING CHANGE",
		},
		{
			"FeAt(ACTIONS): support new repository fields",
			"support new repository fields", "## ✨ New Feature",
		},
		{
			"PERF(cli): avoid repeated initialization",
			"avoid repeated initialization", "## ✨ New Feature",
		},
		{
			"FIX(cli): report missing context",
			"report missing context", "## 🐛 Bug Fix",
		},
		{
			"docs: explain the footer syntax\n\nThis paragraph discusses BREAKING CHANGE: markers without changing behavior.\n\n```text\nfeat!: a hypothetical example\n\nBREAKING CHANGE: Rename the example key.\n```\n\n> BREAKING CHANGE: A quoted example.\n\n    BREAKING CHANGE: An indented example.",
			"explain the footer syntax", "## 🧹 Chore",
		},
		{
			"docs: show a tilde fenced example\n\n~~~~text\nBREAKING-CHANGE: This is example code.\n~~~~",
			"show a tilde fenced example", "## 🧹 Chore",
		},
		{
			"docs: describe lowercase footer text\n\nbreaking change: This is not a Conventional Commits breaking marker.",
			"describe lowercase footer text", "## 🧹 Chore",
		},
		{
			"Merge pull request #18 from topic",
			"Merge pull request", "## 🧹 Chore",
		},
		{
			"maintenance: refresh contributor guidance",
			"refresh contributor guidance", "## 🧹 Chore",
		},
	}
	var commits []string
	for _, test := range cases {
		commits = append(commits, notesTestCommit(t, dir, test.message))
	}
	notes, err := generate(dir, "owner/project", "v2.0.0", commits[len(commits)-1], notesTestCatalog(t,
		[]notesTestRelease{{1, "v1.0.0", false, false, "2026-10-01T12:00:00Z"}},
	))
	if err != nil {
		t.Fatal(err)
	}
	for i, test := range cases {
		section := notesTestSection(t, notes, test.heading)
		if !strings.Contains(section, test.subject) || !strings.Contains(section, "/commit/"+commits[i]) {
			t.Errorf("commit %q missing from %q:\n%s", test.subject, test.heading, section)
		}
		if count := strings.Count(notes, "/commit/"+commits[i]); count != 1 {
			t.Errorf("commit %q appears %d times; want once", test.subject, count)
		}
	}
	breaking := notesTestSection(t, notes, "## 🚨 BREAKING CHANGE")
	for _, wanted := range []string{
		"Rename settings to repository.",
		"Existing declarations must use the new key.",
		"Run validation after migrating.",
		"Add the repo option before applying.",
		"Install the supported runtime before upgrading.",
	} {
		if !strings.Contains(breaking, wanted) {
			t.Errorf("missing migration detail %q:\n%s", wanted, breaking)
		}
	}
	for _, excluded := range []string{"Refs:", "Reviewed-by:", "fixture-reviewer", "Rename the example key", "A quoted example", "An indented example", "This is example code"} {
		if strings.Contains(notes, excluded) {
			t.Errorf("unrelated footer or example %q leaked into notes:\n%s", excluded, notes)
		}
	}
	last := -1
	for _, heading := range []string{"## 🚨 BREAKING CHANGE", "## ✨ New Feature", "## 🐛 Bug Fix", "## 🧹 Chore"} {
		index := strings.Index(notes, heading)
		if index <= last {
			t.Errorf("headings not in requested order:\n%s", notes)
		}
		last = index
	}
}

func TestGenerateIncludesMergedCommitsOnce(t *testing.T) {
	dir := newNotesTestRepository(t)
	previous := notesTestCommit(t, dir, "docs: previously released setup")
	notesTestGit(t, dir, "tag", "v1.0.0")
	notesTestGit(t, dir, "checkout", "-q", "-b", "topic")
	feature := notesTestCommit(t, dir, "feat: add topic functionality")
	notesTestGit(t, dir, "checkout", "-q", "main")
	fix := notesTestCommit(t, dir, "fix: repair the main branch")
	notesTestGit(t, dir, "merge", "--no-ff", "-m", "Merge topic", "topic")
	source := notesTestGit(t, dir, "rev-parse", "HEAD")
	notes, err := generate(dir, "owner/project", "v1.1.0", source, notesTestCatalog(t,
		[]notesTestRelease{{1, "v1.0.0", false, false, "2026-10-01T12:00:00Z"}},
	))
	if err != nil {
		t.Fatal(err)
	}
	for _, sha := range []string{feature, fix, source} {
		if count := strings.Count(notes, "/commit/"+sha); count != 1 {
			t.Errorf("merged history commit %s appears %d times; want once:\n%s", sha, count, notes)
		}
	}
	if strings.Contains(notes, "/commit/"+previous) {
		t.Errorf("previous release commit was included:\n%s", notes)
	}
	if !strings.Contains(notesTestSection(t, notes, "## ✨ New Feature"), "add topic functionality") ||
		!strings.Contains(notesTestSection(t, notes, "## 🐛 Bug Fix"), "repair the main branch") ||
		!strings.Contains(notesTestSection(t, notes, "## 🧹 Chore"), "Merge topic") {
		t.Fatalf("merged history was not classified correctly:\n%s", notes)
	}
}

func TestGenerateBreaksPublicationTimeTiesByReleaseID(t *testing.T) {
	dir := newNotesTestRepository(t)
	notesTestCommit(t, dir, "docs: original release")
	notesTestGit(t, dir, "tag", "v1.0.0")
	previous := notesTestCommit(t, dir, "feat: included in the later release")
	notesTestGit(t, dir, "tag", "v1.1.0")
	source := notesTestCommit(t, dir, "fix: included in the next release")
	earlier := notesTestRelease{1, "v1.0.0", false, false, "2026-10-01T12:00:00Z"}
	later := notesTestRelease{2, "v1.1.0", false, false, "2026-10-01T12:00:00Z"}
	for _, page := range [][]notesTestRelease{{earlier, later}, {later, earlier}} {
		notes, err := generate(dir, "owner/project", "v1.1.1", source, notesTestCatalog(t, page))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(notes, "included in the later release") ||
			!strings.Contains(notes, "https://github.com/owner/project/compare/"+previous+"..."+source) {
			t.Errorf("publication time tie selected the wrong release:\n%s", notes)
		}
	}
}

func TestGenerateFirstReleaseIncludesAllReachableHistory(t *testing.T) {
	dir := newNotesTestRepository(t)
	initial := notesTestCommit(t, dir, "feat: introduce the first capability")
	notesTestGit(t, dir, "tag", "v0.1.0")
	source := notesTestCommit(t, dir, "fix: correct the first capability")
	notesTestGit(t, dir, "tag", "v0.2.0")
	notes, err := generate(dir, "owner/project", "v1.0.0", source, notesTestCatalog(t,
		[]notesTestRelease{
			{1, "v0.1.0", true, false, "2026-10-01T12:00:00Z"},
			{2, "v0.2.0", false, true, "2026-10-02T12:00:00Z"},
		},
	))
	if err != nil {
		t.Fatal(err)
	}
	for _, sha := range []string{initial, source} {
		if strings.Count(notes, "/commit/"+sha) != 1 {
			t.Errorf("first release omitted reachable commit %s:\n%s", sha, notes)
		}
	}
	if !strings.Contains(notes, "https://github.com/owner/project/commits/"+source) || strings.Contains(notes, "/compare/") {
		t.Errorf("first release did not link to frozen-source history:\n%s", notes)
	}
	for _, emptyHeading := range []string{"## 🚨 BREAKING CHANGE", "## 🧹 Chore"} {
		if strings.Contains(notes, emptyHeading) {
			t.Errorf("empty section %q was rendered:\n%s", emptyHeading, notes)
		}
	}
}

func TestGenerateReportsAnEmptyReleaseRange(t *testing.T) {
	dir := newNotesTestRepository(t)
	source := notesTestCommit(t, dir, "feat: already released functionality")
	notesTestGit(t, dir, "tag", "v1.0.0")
	notes, err := generate(dir, "owner/project", "v1.0.1", source, notesTestCatalog(t,
		[]notesTestRelease{{1, "v1.0.0", false, false, "2026-10-01T12:00:00Z"}},
	))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(notes), "no changes") || strings.Contains(notes, "already released functionality") {
		t.Errorf("empty range did not produce a readable no-changes message:\n%s", notes)
	}
	for _, heading := range []string{"## 🚨 BREAKING CHANGE", "## ✨ New Feature", "## 🐛 Bug Fix", "## 🧹 Chore"} {
		if strings.Contains(notes, heading) {
			t.Errorf("empty range produced %q:\n%s", heading, notes)
		}
	}
	if !strings.Contains(notes, "https://github.com/owner/project/compare/"+source+"..."+source) {
		t.Errorf("empty range lacks its exact comparison:\n%s", notes)
	}
}

func TestGenerateRejectsUnavailableHistory(t *testing.T) {
	for _, scenario := range []string{"missing tag", "branch with the tag name", "unrelated published release", "missing source"} {
		t.Run(scenario, func(t *testing.T) {
			dir := newNotesTestRepository(t)
			previous := notesTestCommit(t, dir, "docs: initial release")
			notesTestGit(t, dir, "tag", "v1.0.0")
			source := notesTestCommit(t, dir, "feat: next release")
			switch scenario {
			case "branch with the tag name":
				notesTestGit(t, dir, "branch", "v1.1.0", previous)
			case "unrelated published release":
				notesTestGit(t, dir, "checkout", "--orphan", "unrelated")
				notesTestCommit(t, dir, "docs: unrelated release")
				notesTestGit(t, dir, "tag", "v1.1.0")
				notesTestGit(t, dir, "checkout", "-q", "main")
			case "missing source":
				notesTestGit(t, dir, "tag", "v1.1.0", previous)
				source = strings.Repeat("a", 40)
			}
			notes, err := generate(dir, "owner/project", "v1.2.0", source, notesTestCatalog(t,
				[]notesTestRelease{
					{1, "v1.0.0", false, false, "2026-10-01T12:00:00Z"},
					{2, "v1.1.0", false, false, "2026-10-10T12:00:00Z"},
				},
			))
			if err == nil || notes != "" {
				t.Fatalf("%s produced plausible release notes: notes=%q err=%v", scenario, notes, err)
			}
		})
	}
}

func TestGenerateRejectsShallowHistory(t *testing.T) {
	dir := newNotesTestRepository(t)
	notesTestCommit(t, dir, "feat: an omitted ancestor")
	source := notesTestCommit(t, dir, "fix: the visible shallow commit")
	shallow := filepath.Join(t.TempDir(), "shallow")
	origin := (&url.URL{Scheme: "file", Path: dir}).String()
	notesTestGit(t, dir, "clone", "--quiet", "--depth=1", "--no-tags", origin, shallow)
	if notesTestGit(t, shallow, "rev-parse", "--is-shallow-repository") != "true" {
		t.Fatal("fixture is not shallow")
	}
	notes, err := generate(shallow, "owner/project", "v1.0.0", source, strings.NewReader("[[]]"))
	if err == nil || notes != "" {
		t.Fatalf("shallow history produced incomplete release notes: notes=%q err=%v", notes, err)
	}
}

func TestGenerateRejectsUnreadableReleaseCatalog(t *testing.T) {
	dir := newNotesTestRepository(t)
	source := notesTestCommit(t, dir, "feat: a first release")
	for name, catalog := range map[string]io.Reader{
		"truncated pagination": strings.NewReader(`[[{"id":1,"tag_name":"v1.0.0"}],`),
		"unexpected shape":     strings.NewReader(`{"message":"API unavailable"}`),
		"read failure":         notesTestFailedReader{},
	} {
		t.Run(name, func(t *testing.T) {
			notes, err := generate(dir, "owner/project", "v1.1.0", source, catalog)
			if err == nil || notes != "" {
				t.Fatalf("invalid catalog produced plausible notes: notes=%q err=%v", notes, err)
			}
		})
	}
}

func TestGenerateEscapesCommitText(t *testing.T) {
	dir := newNotesTestRepository(t)
	source := notesTestCommit(t, dir, "feat(api*ops)!: show <img src=x> & [unsafe](https://example.invalid) *literal*\n\nBREAKING CHANGE: Replace <legacy> & [migration](https://example.invalid/migrate).")
	notes, err := generate(dir, "owner/project", "v1.0.0", source, strings.NewReader("[[]]"))
	if err != nil {
		t.Fatal(err)
	}
	for _, unsafe := range []string{"<img src=x>", "<legacy>", "[unsafe](https://example.invalid)", "[migration](https://example.invalid/migrate)", "**api*ops**"} {
		if strings.Contains(notes, unsafe) {
			t.Errorf("unescaped commit content %q:\n%s", unsafe, notes)
		}
	}
	for _, readable := range []string{"unsafe", "migration", "legacy", "literal", "/commit/" + source} {
		if !strings.Contains(notes, readable) {
			t.Errorf("escaping lost content %q:\n%s", readable, notes)
		}
	}
}

type notesTestRelease struct {
	ID          int    `json:"id"`
	TagName     string `json:"tag_name"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
	PublishedAt string `json:"published_at"`
}

func notesTestCatalog(t *testing.T, pages ...[]notesTestRelease) io.Reader {
	t.Helper()
	data, err := json.Marshal(pages)
	if err != nil {
		t.Fatal(err)
	}
	return strings.NewReader(string(data))
}

func newNotesTestRepository(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_COUNT", "0")
	dir := t.TempDir()
	notesTestGit(t, dir, "init", "--quiet", "--initial-branch=main")
	notesTestGit(t, dir, "config", "user.name", "Release Notes Test")
	notesTestGit(t, dir, "config", "user.email", "release-notes@example.invalid")
	notesTestGit(t, dir, "config", "commit.gpgsign", "false")
	notesTestGit(t, dir, "config", "tag.gpgsign", "false")
	return dir
}

func notesTestCommit(t *testing.T, dir, message string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "commit", "--quiet", "--allow-empty", "--cleanup=verbatim", "--file=-")
	cmd.Stdin = strings.NewReader(message + "\n")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create fixture commit: %v: %s", err, output)
	}
	return notesTestGit(t, dir, "rev-parse", "HEAD")
}

func notesTestGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func notesTestSection(t *testing.T, notes, heading string) string {
	t.Helper()
	start := strings.Index(notes, heading+"\n")
	if start < 0 {
		t.Fatalf("missing section %q:\n%s", heading, notes)
	}
	section := notes[start+len(heading)+1:]
	if end := strings.Index(section, "\n## "); end >= 0 {
		section = section[:end]
	}
	return section
}

type notesTestFailedReader struct{}

func (notesTestFailedReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}
