package app

import (
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivedWorkflowPermissionsSelectedUnits(t *testing.T) {
	for _, scenario := range []string{"both", "permission only", "approval only", "permission differs", "approval differs", "normalized selected", "normalized omitted", "unavailable", "denied", "missing reader", "identity drift", "prune", "unsupported"} {
		t.Run(scenario, func(t *testing.T) {
			yes, no, permission := true, false, "read"
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, Actions: settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: &permission, CanApprovePullRequestReviews: &no}}
			switch scenario {
			case "permission only":
				d.Actions.CanApprovePullRequestReviews = nil
			case "approval only", "normalized omitted":
				d.Actions.DefaultWorkflowPermissions = nil
			case "unsupported":
				d.Actions.Enabled = &yes
				if archivedMatchingScope(d, settings.PruneScopes{}) {
					t.Fatal("unsupported execution policy passed the archived scope guard")
				}
			}
			observed := settings.WorkflowPermissionsState{DefaultWorkflowPermissions: "read", CanApprovePullRequestReviews: false}
			switch scenario {
			case "permission differs":
				observed.DefaultWorkflowPermissions = "write"
			case "approval differs":
				observed.CanApprovePullRequestReviews = true
			case "normalized selected", "normalized omitted":
				observed = github.NormalizeWorkflowPermissions(settings.WorkflowPermissionsState{DefaultWorkflowPermissions: "future"})
			case "unavailable":
				observed = settings.WorkflowPermissionsState{}
			}
			repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			archiveReads, reads, writes := 0, 0, 0
			w := Writer{ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
				archiveReads++
				id := int64(1)
				if scenario == "identity drift" && archiveReads == 2 {
					id = 2
				}
				return settings.ArchiveState{ID: id, FullName: "owner/current", Archived: true}, nil
			}, ReadWorkflowPermissions: func(repository.Repository) (settings.WorkflowPermissionsState, error) {
				reads++
				if scenario == "denied" {
					return settings.WorkflowPermissionsState{}, errors.New("denied")
				}
				return observed, nil
			}, WorkflowPermissions: func(repository.Repository, settings.ActionsPolicyDeclaration) (bool, error) {
				writes++
				return false, nil
			}, Archive: func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil }, CreatedID: func(string, string, int64) error { writes++; return nil }, BindCollaboratorID: func(string, int64) error { writes++; return nil }}
			if scenario == "missing reader" {
				w.ReadWorkflowPermissions = nil
			}
			var prune settings.PruneScopes
			if scenario == "prune" {
				var err error
				prune, err = settings.ParsePruneScopes("labels")
				if err != nil {
					t.Fatal(err)
				}
			}
			_, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, w, prune)
			want := scenario == "both" || scenario == "permission only" || scenario == "approval only" || scenario == "normalized omitted"
			if (err == nil) != want || writes != 0 {
				t.Fatalf("err=%v reads=%d writes=%d", err, reads, writes)
			}
			if want && (reads != 1 || archiveReads != 2) {
				t.Fatalf("reads=%d archive=%d", reads, archiveReads)
			}
		})
	}
}

func TestArchivedWorkflowPermissionsAndOIDCMustBothMatch(t *testing.T) {
	for _, scenario := range []string{"matching", "workflow differs", "OIDC differs"} {
		t.Run(scenario, func(t *testing.T) {
			yes, no, permission := true, false, "read"
			immutable := no
			if scenario == "OIDC differs" {
				immutable = yes
			}
			oidc, err := settings.NewOIDCSubjectObservation(settings.OIDCSubjectTemplate{UseDefaultTemplate: &yes, UseImmutableSubjectClaim: &immutable})
			if err != nil {
				t.Fatal(err)
			}
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, OIDCSubject: &settings.OIDCSubjectTemplate{UseImmutableSubjectClaim: &no}, Actions: settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: &permission}}
			repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			writes, oidcReads, workflowReads := 0, 0, 0
			w := Writer{ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
				return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
			}, ReadArchivedOIDCSubject: func(repository.Repository, settings.ArchiveState) ([2]settings.OIDCSubjectObservation, error) {
				oidcReads++
				return [2]settings.OIDCSubjectObservation{oidc, oidc}, nil
			}, ReadWorkflowPermissions: func(repository.Repository) (settings.WorkflowPermissionsState, error) {
				workflowReads++
				value := "read"
				if scenario == "workflow differs" {
					value = "write"
				}
				return settings.WorkflowPermissionsState{DefaultWorkflowPermissions: value}, nil
			}, OIDCSubject: func(repository.Repository, settings.OIDCSubjectTemplate) (bool, error) { writes++; return false, nil }, WorkflowPermissions: func(repository.Repository, settings.ActionsPolicyDeclaration) (bool, error) {
				writes++
				return false, nil
			}}
			_, err = Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, w)
			if (err == nil) != (scenario == "matching") || writes != 0 || oidcReads != 1 || scenario == "matching" && workflowReads != 1 {
				t.Fatalf("err=%v writes=%d reads=%d/%d", err, writes, oidcReads, workflowReads)
			}
		})
	}
}
