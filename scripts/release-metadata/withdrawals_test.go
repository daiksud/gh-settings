package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseWithdrawalsAndClarifications(t *testing.T) {
	f := newFixture(t, "feat!: expose <temporary> mode\n\nBREAKING CHANGE: Switch to temporary mode.", "feat: retain useful mode")
	commits := strings.Fields(mustGit(t, f.root, "rev-list", "--reverse", f.base+".."+f.source))
	write(t, filepath.Join(f.root, "change.txt"), "withdraw temporary mode")
	mustGit(t, f.root, "add", ".")
	mustGit(t, f.root, "commit", "--quiet", "-m", "build(release): clarify final behavior\n\nRelease-note: Temporary mode was removed before publication.\nExisting <old> mode users must review [the change].\n\nRelease-withdraws: "+commits[0]+" Rejected in the final source.\nDo not switch to *temporary* mode.\n\nRefs #12")
	source := mustGit(t, f.root, "rev-parse", "HEAD")
	// A later removal cannot change the already frozen interval.
	before, err := generate(f.root, "owner/repo", "v1.3.0", f.source, strings.NewReader(f.catalog), "", "")
	if err != nil || !strings.Contains(before, "### 💥 BREAKING CHANGE") || strings.Contains(before, "Withdrawn") {
		t.Fatal("later withdrawal entered frozen interval", before, err)
	}
	notes, err := generate(f.root, "owner/repo", "v1.3.0", source, strings.NewReader(f.catalog), "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"## Release clarifications\n\n- [Clarification](https://github.com/owner/repo/commit/" + source + ")",
		"  > Existing &lt;old&gt; mode users must review \\[the change\\].",
		"### ↩️ Withdrawn before release\n\n- [expose &lt;temporary&gt; mode](https://github.com/owner/repo/commit/" + commits[0] + ")",
		"Withdrawn by [this change](https://github.com/owner/repo/commit/" + source + "); not available in this release.",
		"  > Rejected in the final source.\n  > Do not switch to \\*temporary\\* mode.",
		"### ✨ New Feature\n\n- [retain useful mode]",
		"### 🛠️ Maintenance\n\n- [clarify final behavior]",
	} {
		if !strings.Contains(notes, want) {
			t.Fatalf("missing %q in notes:\n%s", want, notes)
		}
	}
	if strings.Contains(notes, "Migration guidance") || strings.Contains(notes, "Switch to temporary mode") || strings.Contains(notes, "Refs #12") || strings.Count(notes, "- [expose") != 1 {
		t.Fatal("withdrawn entry advertised or obsolete migration leaked", notes)
	}
}

func TestInvalidReleaseAnnotationsFailBeforeOutput(t *testing.T) {
	for _, value := range []string{
		"Release-note: ",
		"Release-withdraws: short reason",
		"Release-withdraws: " + strings.Repeat("f", 40) + " unknown target",
		"Release-withdraws: BASE already published",
		"Release-withdraws: TARGET",
		"Release-withdraws: TARGET   ",
		"Release-withdraws: TARGET reason\nRelease-withdraws: TARGET duplicate",
	} {
		t.Run(value, func(t *testing.T) {
			f := newFixture(t, "feat: temporary mode")
			value := strings.NewReplacer("BASE", f.base, "TARGET", f.source).Replace(value)
			write(t, filepath.Join(f.root, "change.txt"), "removal")
			mustGit(t, f.root, "add", ".")
			mustGit(t, f.root, "commit", "--quiet", "--cleanup=verbatim", "-m", "build: annotate release\n\n"+value)
			catalog := filepath.Join(f.root, "catalog.json")
			write(t, catalog, f.catalog)
			var output strings.Builder
			err := run(f.root, []string{"owner/repo", "v1.3.0", mustGit(t, f.root, "rev-parse", "HEAD"), catalog}, &output)
			if err == nil || output.Len() != 0 {
				t.Fatal("invalid annotation produced output", err, output.String())
			}
		})
	}
}

func TestReleaseAnnotationsOnlyReadFormalFooters(t *testing.T) {
	for _, body := range []string{
		"Prose introduction\nRelease-note: not a footer\nRelease-withdraws: ignored",
		"```text\nRelease-note: not a footer\nRelease-withdraws: ignored\n```",
		"~~~text\nRelease-note: not a footer\n~~~",
		"> Release-note: quoted\n> Release-withdraws: ignored",
		"Release-note #12\nRelease-withdraws #13",
	} {
		c := parseChange(strings.Repeat("a", 40), "docs: examples\n\n"+body)
		if len(c.notes) != 0 || len(c.withdrawals) != 0 {
			t.Fatalf("example became annotation: %+v", c)
		}
	}
	c := parseChange(strings.Repeat("a", 40), "build: notes\r\n\r\nRelease-note: first\r\ncontinued\r\nRelease-note: second\r\nBREAKING CHANGE: real migration")
	if len(c.notes) != 2 || c.notes[0] != "first\ncontinued" || !c.major || c.breaking[0] != "real migration" {
		t.Fatalf("formal multiline repeated footers lost: %+v", c)
	}
}

func TestWithdrawalReferencesAreEarlierUniqueAndUnchained(t *testing.T) {
	a, b, c := strings.Repeat("a", 40), strings.Repeat("b", 40), strings.Repeat("c", 40)
	for _, changes := range [][]change{
		{{sha: a, withdrawals: []string{a + " self"}}},
		{{sha: a, withdrawals: []string{b + " forward"}}, {sha: b}},
		{{sha: a}, {sha: b, withdrawals: []string{a + " reason"}}, {sha: c, withdrawals: []string{a + " duplicate"}}},
		{{sha: a}, {sha: b, withdrawals: []string{a + " reason"}}, {sha: c, withdrawals: []string{b + " chain"}}},
	} {
		if err := applyWithdrawals(changes); err == nil {
			t.Fatal("ambiguous withdrawal accepted", changes)
		}
	}
}
