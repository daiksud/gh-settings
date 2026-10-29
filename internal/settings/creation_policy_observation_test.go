package settings

import "testing"

func TestRepositoryObservationOwnsCreationPolicyProvenance(t *testing.T) {
	prRecord := ObservationNormalization{Path: "repository.settings.general.features.pull_request_permissions.creation_allowed_by", RawValue: "FUTURE_PR", Fallback: "all_users"}
	issueRecord := ObservationNormalization{Path: "repository.settings.general.features.issue_permissions.creation_allowed_by", RawValue: "FUTURE_ISSUE", Fallback: "all_users"}
	pr := PRAccessState{HasPullRequests: true, PullRequestCreationPolicy: "ALL", Normalization: prRecord}
	issue := IssueCreationPolicyState{Policy: "ALL", Normalization: issueRecord}
	observed := NewRepositoryObservation(&RepositoryState{DefaultBranch: "main"}, &MergeMethodsState{MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE", SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES"}, &pr).WithIssueCreationPolicy(issue).WithSponsorships(SponsorshipsState{})
	pr.Normalization.RawValue = "mutated source"
	issue.Normalization.RawValue = "mutated source"
	records := observed.CreationPolicyNormalizations()
	if len(records) != 2 || records[0] != prRecord || records[1] != issueRecord {
		t.Fatal("lost scalar provenance or unstable field order", records)
	}
	records[0].RawValue = "mutated caller"
	if again := observed.CreationPolicyNormalizations(); len(again) != 2 || again[0] != prRecord || again[1] != issueRecord {
		t.Fatal("returned records alias observation", again)
	}
	declaration, err := observed.Declaration()
	if err != nil || declaration.PullRequestCreationPolicy == nil || *declaration.PullRequestCreationPolicy != "ALL" || declaration.IssueCreationPolicy == nil || *declaration.IssueCreationPolicy != "ALL" {
		t.Fatal("wire evidence leaked into effective intent", declaration, err)
	}
	err = VisitDeclaredRepositoryFields(declaration, func(field, value string) error {
		if value == "FUTURE_PR" || value == "FUTURE_ISSUE" {
			t.Fatal("raw evidence became declared field", field, value)
		}
		return nil
	}, func(string, bool) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreationPolicyProvenanceDistinguishesEmptyAndUnacquired(t *testing.T) {
	for _, observation := range []RepositoryObservation{NewRepositoryObservation(nil, nil, nil), NewRepositoryObservation(nil, nil, &PRAccessState{PullRequestCreationPolicy: "ALL"}).WithIssueCreationPolicy(IssueCreationPolicyState{Policy: "ALL"})} {
		if records := observation.CreationPolicyNormalizations(); len(records) != 0 {
			t.Fatal("unacquired/canonical policy acquired provenance", records)
		}
	}
	empty := ObservationNormalization{Path: "repository.settings.general.features.issue_permissions.creation_allowed_by", RawValue: "", Fallback: "all_users"}
	observed := NewRepositoryObservation(nil, nil, nil).WithIssueCreationPolicy(IssueCreationPolicyState{Policy: "ALL", Normalization: empty})
	if records := observed.CreationPolicyNormalizations(); len(records) != 1 || records[0] != empty {
		t.Fatal("empty acquired token was omitted", records)
	}
}
