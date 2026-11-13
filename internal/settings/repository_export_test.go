package settings

import (
	"reflect"
	"testing"
)

func TestRepositoryExportGroupsAreCompleteAndDisjoint(t *testing.T) {
	want := [][]string{
		{"repository.description", "repository.homepage", "repository.default_branch", "repository.has_issues", "repository.has_projects", "repository.has_wiki", "repository.has_discussions", "repository.web_commit_signoff_required", "repository.is_template"},
		{"repository.allow_merge_commit", "repository.allow_squash_merge", "repository.allow_rebase_merge", "repository.delete_branch_on_merge", "repository.allow_auto_merge", "repository.allow_update_branch", "repository.squash_merge_commit_title", "repository.squash_merge_commit_message", "repository.merge_commit_title", "repository.merge_commit_message"},
		{"repository.has_pull_requests", "repository.pull_request_creation_policy"},
		{"repository.issue_creation_policy"},
		{"repository.has_sponsorships"},
	}
	groups := RepositoryExportGroups()
	if len(groups) != len(want) {
		t.Fatalf("groups=%v", groups)
	}
	var all []FieldID
	for i, group := range groups {
		fields := group.Fields()
		var paths []string
		for _, field := range fields {
			paths = append(paths, field.Path())
		}
		if !reflect.DeepEqual(paths, want[i]) {
			t.Fatalf("group %d fields=%v, want %v", i, paths, want[i])
		}
		base, merge, pr, issue, sponsor, err := RepositoryObservationGroups(fields)
		selected := []bool{base, merge, pr, issue, sponsor}
		for j, enabled := range selected {
			if err != nil || enabled != (i == j) {
				t.Fatalf("group %d selects=%v, err=%v", i, selected, err)
			}
		}
		all = append(all, fields...)
		fields[0] = FieldID{}
		if group.Fields()[0].Path() != want[i][0] {
			t.Fatal("caller mutated group membership")
		}
	}
	if !reflect.DeepEqual(all, AllRepositoryFields()) {
		t.Fatal("export partition does not cover exactly the strict scalar observation")
	}
}

func TestRepositoryExportProjectionOwnsOnlyCompleteRequestedGroup(t *testing.T) {
	observation := NewRepositoryObservation(&RepositoryState{DefaultBranch: "main"}, &MergeMethodsState{}, &PRAccessState{}).
		WithIssueCreationPolicy(IssueCreationPolicyState{}).WithSponsorships(SponsorshipsState{})
	archived, retained := true, "retained"
	prior := RepositoryDeclaration{Archived: &archived, Description: &retained}
	for _, group := range RepositoryExportGroups() {
		t.Run(group.Fields()[0].Path(), func(t *testing.T) {
			isolated, err := observation.ExportDeclaration(group, RepositoryDeclaration{})
			if err != nil || !reflect.DeepEqual(DeclaredRepositoryFields(isolated), group.Fields()) {
				t.Fatalf("projection imported missing or unrelated values: %+v, %v", isolated, err)
			}
			merged, err := observation.ExportDeclaration(group, prior)
			if err != nil || merged.Archived == nil || !*merged.Archived {
				t.Fatalf("projection lost lifecycle intent: %+v, %v", merged, err)
			}
			if group == RepositoryExportBase {
				if *merged.Description != "" || merged.HasIssues == nil || *merged.HasIssues {
					t.Fatal("explicit empty/false did not replace prior intent")
				}
				*merged.Description = "caller change"
				if *observation.values.Description != "" || retained != "retained" {
					t.Fatal("projection aliases observed or previous values")
				}
			} else if merged.Description != prior.Description {
				t.Fatal("unrelated prior intent changed")
			}
			missing := observation
			switch group {
			case RepositoryExportBase:
				missing.values.Homepage = nil
			case RepositoryExportMergeMethods:
				missing.values.MergeCommitMessage = nil
			case RepositoryExportPullRequests:
				missing.values.PullRequestCreationPolicy = nil
			case RepositoryExportIssuePolicy:
				missing.values.IssueCreationPolicy = nil
			case RepositoryExportSponsorships:
				missing.values.HasSponsorships = nil
			}
			got, err := missing.ExportDeclaration(group, prior)
			if err == nil || !reflect.DeepEqual(got, RepositoryDeclaration{}) || retained != "retained" {
				t.Fatalf("incomplete group exposed partial projection: %+v, %v", got, err)
			}
			if _, err := missing.Declaration(); err == nil {
				t.Fatal("export relaxed the strict all-fields projection")
			}
		})
	}
}
