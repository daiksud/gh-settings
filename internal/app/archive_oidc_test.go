package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivedOIDCSelectedUnits(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	for _, mode := range []string{"default", "custom", "empty custom", "immutable false", "immutable true", "combined", "unselected immutable missing", "unselected template differs", "mode differs", "key order differs", "immutable differs", "immutable missing", "first missing", "fresh missing", "fresh differs", "reader unavailable", "reader error", "archive drift", "prune"} {
		t.Run(mode, func(t *testing.T) {
			yes, no := true, false
			keys := []string{"repo", "context"}
			d := settings.OIDCSubjectTemplate{UseDefaultTemplate: &yes}
			observed := settings.OIDCSubjectTemplate{UseDefaultTemplate: &yes, UseImmutableSubjectClaim: &no}
			switch mode {
			case "custom", "key order differs", "combined":
				d.UseDefaultTemplate, d.SubjectClaimTemplate = &no, &keys
				observed.UseDefaultTemplate, observed.SubjectClaimTemplate = &no, &keys
			case "empty custom":
				empty := []string{}
				d.UseDefaultTemplate, d.SubjectClaimTemplate = &no, &empty
				observed.UseDefaultTemplate, observed.SubjectClaimTemplate = &no, &empty
			case "immutable false", "unselected template differs", "immutable differs", "immutable missing":
				d.UseDefaultTemplate, d.UseImmutableSubjectClaim = nil, &no
			case "immutable true":
				d.UseDefaultTemplate, d.UseImmutableSubjectClaim = nil, &yes
				observed.UseImmutableSubjectClaim = &yes
			}
			if mode == "combined" {
				d.UseImmutableSubjectClaim = &no
			}
			if mode == "unselected immutable missing" {
				observed.UseImmutableSubjectClaim = nil
			}
			if mode == "unselected template differs" {
				observed.UseDefaultTemplate, observed.SubjectClaimTemplate = &no, &keys
			}
			if mode == "mode differs" {
				observed.UseDefaultTemplate, observed.SubjectClaimTemplate = &no, &keys
			}
			if mode == "key order differs" {
				reverse := []string{"context", "repo"}
				observed.SubjectClaimTemplate = &reverse
			}
			if mode == "immutable differs" {
				observed.UseImmutableSubjectClaim = &yes
			}
			if mode == "immutable missing" {
				observed.UseImmutableSubjectClaim = nil
			}
			o, err := settings.NewOIDCSubjectObservation(observed)
			if err != nil {
				t.Fatal(err)
			}
			observations := [2]settings.OIDCSubjectObservation{o, o}
			if mode == "first missing" {
				observations[0] = settings.OIDCSubjectObservation{}
			}
			if mode == "fresh missing" {
				observations[1] = settings.OIDCSubjectObservation{}
			}
			if mode == "fresh differs" {
				other := settings.OIDCSubjectTemplate{UseDefaultTemplate: &no, SubjectClaimTemplate: &keys}
				observations[1], err = settings.NewOIDCSubjectObservation(other)
				if err != nil {
					t.Fatal(err)
				}
			}
			declaration := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, OIDCSubject: &d}
			before := settings.CloneOIDCSubject(d)
			archiveReads, reads, writes := 0, 0, 0
			w := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					archiveReads++
					id := int64(1)
					if mode == "archive drift" && archiveReads == 2 {
						id = 2
					}
					return settings.ArchiveState{ID: id, FullName: "owner/current", Archived: true}, nil
				},
				ReadArchivedOIDCSubject: func(_ repository.Repository, state settings.ArchiveState) ([2]settings.OIDCSubjectObservation, error) {
					reads++
					if state.ID != 1 || !state.Archived {
						t.Fatal("wrong identity")
					}
					if mode == "reader error" {
						return observations, errors.New("denied")
					}
					return observations, nil
				},
				OIDCSubject: func(repository.Repository, settings.OIDCSubjectTemplate) (bool, error) { writes++; return false, nil },
				Archive:     func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
				CreatedID:   func(string, string, int64) error { writes++; return nil },
			}
			if mode == "reader unavailable" {
				w.ReadArchivedOIDCSubject = nil
			}
			var prune settings.PruneScopes
			if mode == "prune" {
				prune, err = settings.ParsePruneScopes("labels")
				if err != nil {
					t.Fatal(err)
				}
			}
			_, err = Apply(func() (settings.Declaration, error) { return declaration, nil }, func() (repository.Repository, error) { return repo, nil }, w, prune)
			want := mode == "default" || mode == "custom" || mode == "empty custom" || mode == "immutable false" || mode == "immutable true" || mode == "combined" || mode == "unselected immutable missing" || mode == "unselected template differs"
			if (err == nil) != want || writes != 0 || !reflect.DeepEqual(d, before) {
				t.Fatalf("err=%v reads=%d writes=%d", err, reads, writes)
			}
			if want && (reads != 1 || archiveReads != 2) {
				t.Fatalf("reads=%d archive=%d", reads, archiveReads)
			}
		})
	}
}

func TestArchivedOIDCAndWebhookMustBothMatch(t *testing.T) {
	for _, mode := range []string{"matching", "oidc differs", "webhook differs"} {
		t.Run(mode, func(t *testing.T) {
			yes, no, content := true, false, "application/json"
			d := settings.OIDCSubjectTemplate{UseImmutableSubjectClaim: &no}
			o := settings.OIDCSubjectTemplate{UseDefaultTemplate: &yes, UseImmutableSubjectClaim: &no}
			if mode == "oidc differs" {
				o.UseImmutableSubjectClaim = &yes
			}
			if mode == "webhook differs" {
				content = "application/x-www-form-urlencoded"
			}
			observed, err := settings.NewOIDCSubjectObservation(o)
			if err != nil {
				t.Fatal(err)
			}
			hooks := []settings.Webhook{{ID: 7, ContentType: &content}}
			declaration := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, OIDCSubject: &d, Webhooks: &hooks}
			repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			writes, hooksRead, oidcRead := 0, 0, 0
			w := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
				},
				ReadWebhooks: func(repository.Repository, []int64) (settings.WebhookObservation, error) {
					hooksRead++
					o := webhookAppObservation()
					o.Target.Archived = true
					return o, nil
				},
				ReadArchivedOIDCSubject: func(repository.Repository, settings.ArchiveState) ([2]settings.OIDCSubjectObservation, error) {
					oidcRead++
					return [2]settings.OIDCSubjectObservation{observed, observed}, nil
				},
				OIDCSubject: func(repository.Repository, settings.OIDCSubjectTemplate) (bool, error) { writes++; return false, nil },
				Webhook:     func(repository.Repository, settings.WebhookSelection) (bool, error) { writes++; return false, nil },
			}
			_, err = Apply(func() (settings.Declaration, error) { return declaration, nil }, func() (repository.Repository, error) { return repo, nil }, w)
			if (err == nil) != (mode == "matching") || writes != 0 || hooksRead == 0 || mode == "matching" && oidcRead != 1 {
				t.Fatalf("err=%v hooks=%d oidc=%d writes=%d", err, hooksRead, oidcRead, writes)
			}
		})
	}
}

func TestArchivedOIDCAndEnvironmentMustBothMatch(t *testing.T) {
	for _, mode := range []string{"matching", "oidc differs", "environment differs"} {
		t.Run(mode, func(t *testing.T) {
			yes, no, zero := true, false, 0
			d := settings.OIDCSubjectTemplate{UseImmutableSubjectClaim: &no}
			o := settings.OIDCSubjectTemplate{UseDefaultTemplate: &yes, UseImmutableSubjectClaim: &no}
			if mode == "oidc differs" {
				o.UseImmutableSubjectClaim = &yes
			}
			oidc, err := settings.NewOIDCSubjectObservation(o)
			if err != nil {
				t.Fatal(err)
			}
			envs, err := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{{Name: "production", WaitTimer: &zero}})
			if err != nil {
				t.Fatal(err)
			}
			timer := 0
			if mode == "environment differs" {
				timer = 1
			}
			reviewers := []settings.EnvironmentReviewer{}
			state := settings.EnvironmentState{ID: 71, CustomRules: []settings.EnvironmentCustomRule{}, Policy: settings.EnvironmentDeclaration{Name: "production", WaitTimer: &timer, PreventSelfReview: &no, CanAdminsBypass: &no, Reviewers: &reviewers, DeploymentBranchPolicyDeclared: true}}
			declaration := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, OIDCSubject: &d, Environments: &envs}
			repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			writes, environmentReads, oidcReads := 0, 0, 0
			w := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
				},
				ReadEnvironments: func(repository.Repository, []string) (settings.EnvironmentObservation, error) {
					environmentReads++
					return settings.NewEnvironmentObservation([]settings.EnvironmentState{state})
				},
				ReadArchivedOIDCSubject: func(repository.Repository, settings.ArchiveState) ([2]settings.OIDCSubjectObservation, error) {
					oidcReads++
					return [2]settings.OIDCSubjectObservation{oidc, oidc}, nil
				},
				OIDCSubject: func(repository.Repository, settings.OIDCSubjectTemplate) (bool, error) { writes++; return false, nil },
				Environment: func(repository.Repository, settings.EnvironmentDeclaration) (bool, error) {
					writes++
					return false, nil
				},
			}
			_, err = Apply(func() (settings.Declaration, error) { return declaration, nil }, func() (repository.Repository, error) { return repo, nil }, w)
			if (err == nil) != (mode == "matching") || writes != 0 || environmentReads == 0 || mode == "matching" && oidcReads != 1 {
				t.Fatalf("err=%v environment=%d oidc=%d writes=%d", err, environmentReads, oidcReads, writes)
			}
		})
	}
}
