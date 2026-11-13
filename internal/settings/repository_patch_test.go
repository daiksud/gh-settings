package settings

import (
	"reflect"
	"testing"
)

func TestRepositoryPatchChangesPreservesControlOwnership(t *testing.T) {
	before := RepositoryDeclaration{Description: pagesPtr("same"), Homepage: pagesPtr("old"), HasIssues: pagesPtr(true), AllowMergeCommit: pagesPtr(true), MergeCommitTitle: pagesPtr("PR_TITLE"), MergeCommitMessage: pagesPtr("BLANK"), HasSponsorships: pagesPtr(false)}
	desired := RepositoryDeclaration{Description: pagesPtr("same"), Homepage: pagesPtr(""), HasIssues: pagesPtr(false), MergeCommitTitle: pagesPtr("PR_TITLE"), MergeCommitMessage: pagesPtr("PR_BODY"), HasSponsorships: pagesPtr(true), Archived: pagesPtr(true)}
	want := RepositoryDeclaration{Homepage: pagesPtr(""), HasIssues: pagesPtr(false), MergeCommitTitle: pagesPtr("PR_TITLE"), MergeCommitMessage: pagesPtr("PR_BODY")}
	got, err := RepositoryPatchChanges(before, desired)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("patch=%+v, err=%v", got, err)
	}
	*got.Homepage = "changed copy"
	if *desired.Homepage != "" || *before.Homepage != "old" {
		t.Fatal("patch aliases source intent")
	}
	before.Description = nil
	if got, err := RepositoryPatchChanges(before, desired); err == nil || !got.Empty() {
		t.Fatal("missing selected value returned partial patch", got, err)
	}
}

func TestRepositoryPatchMenusUseRawValuesAndKeepPartialOwnership(t *testing.T) {
	for _, menu := range []CommitMessageMenu{MergeCommitMenu, SquashCommitMenu} {
		for _, test := range []struct {
			name, rawTitle, rawMessage, title, message string
			partial, changed                           bool
		}{
			{"canonical match", "PR_TITLE", "BLANK", "PR_TITLE", "BLANK", false, false},
			{"one member differs", "PR_TITLE", "BLANK", "PR_TITLE", "PR_BODY", false, true},
			{"fallback projection is not persistence", "FUTURE_TITLE", "FUTURE_MESSAGE", "", "", false, true},
			{"partial native match", "PR_TITLE", "FUTURE_MESSAGE", "PR_TITLE", "", true, false},
			{"partial native change", "FUTURE_TITLE", "FUTURE_MESSAGE", "PR_TITLE", "", true, true},
		} {
			t.Run(string(menu)+"/"+test.name, func(t *testing.T) {
				before := RepositoryDeclaration{AllowMergeCommit: pagesPtr(true)}
				desired := RepositoryDeclaration{}
				if test.title == "" {
					test.title, test.message = "MERGE_MESSAGE", "PR_TITLE"
					if menu == SquashCommitMenu {
						test.title, test.message = "COMMIT_OR_PR_TITLE", "COMMIT_MESSAGES"
					}
				}
				if menu == MergeCommitMenu {
					before.MergeCommitTitle, before.MergeCommitMessage = &test.rawTitle, &test.rawMessage
					desired.MergeCommitTitle = &test.title
					if !test.partial {
						desired.MergeCommitMessage = &test.message
					}
				} else {
					before.SquashMergeCommitTitle, before.SquashMergeCommitMessage = &test.rawTitle, &test.rawMessage
					desired.SquashMergeCommitTitle = &test.title
					if !test.partial {
						desired.SquashMergeCommitMessage = &test.message
					}
				}
				got, err := RepositoryPatchChanges(before, desired)
				want := RepositoryDeclaration{}
				if test.changed {
					want = desired
				}
				if err != nil || !reflect.DeepEqual(got, want) {
					t.Fatalf("patch=%+v want=%+v err=%v", got, want, err)
				}
			})
		}
	}
}

func TestRepositoryPatchDependencyMatchesPlan(t *testing.T) {
	for _, mergeEnabled := range []bool{false, true} {
		for _, explicitEnable := range []bool{false, true} {
			state := MergeMethodsState{AllowMergeCommit: mergeEnabled, SquashMergeCommitTitle: "PR_TITLE", SquashMergeCommitMessage: "BLANK", MergeCommitTitle: "PR_TITLE", MergeCommitMessage: "BLANK"}
			desired := RepositoryDeclaration{MergeCommitTitle: pagesPtr("PR_TITLE"), MergeCommitMessage: pagesPtr("PR_BODY")}
			if explicitEnable {
				desired.AllowMergeCommit = pagesPtr(true)
			}
			_, planErr := CompareRepositoryObservation(NewRepositoryObservation(nil, &state, nil), desired)
			_, applyErr := RepositoryPatchChanges(RepositoryDeclaration{}.WithMergeMethods(state), desired)
			if (applyErr != nil) != (planErr != nil) || (applyErr != nil) != (!mergeEnabled && !explicitEnable) {
				t.Fatal("plan/apply dependency diverged", planErr, applyErr)
			}
		}
	}
	// Preserve the existing asymmetry: an omitted squash checkbox does not
	// acquire a new observed-enabled prerequisite.
	state := MergeMethodsState{AllowSquashMerge: false, SquashMergeCommitTitle: "PR_TITLE", SquashMergeCommitMessage: "BLANK"}
	desired := RepositoryDeclaration{SquashMergeCommitTitle: pagesPtr("PR_TITLE"), SquashMergeCommitMessage: pagesPtr("PR_BODY")}
	if _, err := RepositoryPatchChanges(RepositoryDeclaration{}.WithMergeMethods(state), desired); err != nil {
		t.Fatal(err)
	}
	if _, err := CompareRepositoryObservation(NewRepositoryObservation(nil, &state, nil), desired); err != nil {
		t.Fatal(err)
	}
}
