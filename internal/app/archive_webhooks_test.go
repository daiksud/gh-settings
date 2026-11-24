package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivedExistingWebhookReadOnlyMatching(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	for _, mode := range []string{"matching", "unselected changes", "event order", "config differs", "active differs", "events differ", "missing", "incomplete", "reader nil", "first read error", "fresh read error", "created drift", "id drift", "owner drift", "repository id drift", "repository name drift", "archive drift", "archive final drift", "create", "prune"} {
		t.Run(mode, func(t *testing.T) {
			yes, active, content := true, false, "application/json"
			events := []string{"push", "issues"}
			webhooks := []settings.Webhook{{ID: 7, ContentType: &content, Active: &active, Events: &events}}
			if mode == "create" {
				url := "https://example.com/hook"
				webhooks[0].ID, webhooks[0].PayloadURL = 0, &url
			}
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, Webhooks: &webhooks}
			before := webhooks[0].Clone()
			archiveReads, reads, writes := 0, 0, 0
			w := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					archiveReads++
					id := int64(1)
					if mode == "archive final drift" && archiveReads == 2 {
						id = 2
					}
					return settings.ArchiveState{ID: id, FullName: "owner/current", Archived: true}, nil
				},
				ReadWebhooks: func(_ repository.Repository, ids []int64) (settings.WebhookObservation, error) {
					reads++
					if !reflect.DeepEqual(ids, []int64{7}) {
						t.Fatalf("unexpected IDs %v", ids)
					}
					if mode == "first read error" || mode == "fresh read error" && reads == 2 {
						return settings.WebhookObservation{}, errors.New("unavailable")
					}
					o := webhookAppObservation()
					o.Complete = false // Selected-ID reads need not claim a complete inventory.
					o.Target.Archived = true
					o.Values[0].Events = []string{"issues", "push"}
					if reads == 2 {
						switch mode {
						case "unselected changes":
							o.Values[0].Config.PayloadURL = "https://other.example/hook"
							o.Values[0].Config.InsecureSSL = true
						case "event order":
							o.Values[0].Events = []string{"push", "issues"}
						case "config differs":
							o.Values[0].Config.ContentType = "application/x-www-form-urlencoded"
						case "active differs":
							o.Values[0].Active = true
						case "events differ":
							o.Values[0].Events = []string{"push"}
						case "missing":
							o.Values = []settings.WebhookState{}
						case "incomplete":
							o.Values[0].Events = nil
						case "created drift":
							o.Values[0].CreatedAt = "another"
						case "id drift":
							o.Values[0].ID = 8
						case "owner drift":
							o.Target.OwnerID = 3
						case "repository id drift":
							o.Target.ID = 2
						case "repository name drift":
							o.Target.FullName = "owner/other"
						case "archive drift":
							o.Target.Archived = false
						}
					}
					return o, nil
				},
				Archive:   func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
				Webhook:   func(repository.Repository, settings.WebhookSelection) (bool, error) { writes++; return false, nil },
				CreatedID: func(string, string, int64) error { writes++; return nil },
			}
			if mode == "reader nil" {
				w.ReadWebhooks = nil
			}
			var prune settings.PruneScopes
			if mode == "prune" {
				var err error
				prune, err = settings.ParsePruneScopes("labels")
				if err != nil {
					t.Fatal(err)
				}
			}
			result, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, w, prune)
			wantSuccess := mode == "matching" || mode == "unselected changes" || mode == "event order"
			if (err == nil) != wantSuccess || writes != 0 {
				t.Fatalf("result=%v err=%v writes=%d", result.Outcome, err, writes)
			}
			if !reflect.DeepEqual(webhooks[0].Clone(), before) {
				t.Fatal("declaration changed")
			}
			if wantSuccess && (reads != 2 || archiveReads != 2) {
				t.Fatalf("reads: webhook=%d archive=%d", reads, archiveReads)
			}
			if (mode == "create" || mode == "reader nil" || mode == "prune") && reads != 0 {
				t.Fatalf("unexpected reads: %d", reads)
			}
		})
	}
}

func TestArchivedWebhookAndRulesetMustBothMatch(t *testing.T) {
	for _, mode := range []string{"matching", "webhook differs", "ruleset differs"} {
		t.Run(mode, func(t *testing.T) {
			state := appRulesetState(t)
			policy := state.Policy
			id := int64(7)
			policy.ID = &id
			if mode == "ruleset differs" {
				value := "disabled"
				policy.Enforcement = &value
			}
			policies, err := settings.NewRulesetDeclarations([]settings.RulesetDeclaration{policy})
			if err != nil {
				t.Fatal(err)
			}
			archived, content := true, "application/json"
			if mode == "webhook differs" {
				content = "application/x-www-form-urlencoded"
			}
			hooks := []settings.Webhook{{ID: 7, ContentType: &content}}
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived}, Webhooks: &hooks, Rulesets: &policies}
			repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			ruleReads, hookReads, writes := 0, 0, 0
			w := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
				},
				ReadRulesets: func(repository.Repository) (settings.RulesetObservation, error) {
					ruleReads++
					return settings.NewRulesetObservation([]settings.RulesetState{state}, nil)
				},
				ReadWebhooks: func(repository.Repository, []int64) (settings.WebhookObservation, error) {
					hookReads++
					o := webhookAppObservation()
					o.Target.Archived = true
					return o, nil
				},
				Webhook:   func(repository.Repository, settings.WebhookSelection) (bool, error) { writes++; return false, nil },
				Ruleset:   func(repository.Repository, settings.RulesetChange) (int64, error) { writes++; return 0, nil },
				CreatedID: func(string, string, int64) error { writes++; return nil },
			}
			_, err = Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, w)
			if (err == nil) != (mode == "matching") || writes != 0 || ruleReads != 1 || (mode == "matching" && hookReads != 2) {
				t.Fatalf("err=%v rules=%d webhooks=%d writes=%d", err, ruleReads, hookReads, writes)
			}
		})
	}
}

func TestArchivedWebhookMultipleSelectedIDs(t *testing.T) {
	for _, mode := range []string{"matching", "second missing", "second differs", "second identity drift"} {
		t.Run(mode, func(t *testing.T) {
			yes, content := true, "application/json"
			hooks := []settings.Webhook{{ID: 7, ContentType: &content}, {ID: 8, ContentType: &content}}
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, Webhooks: &hooks}
			repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			reads, writes := 0, 0
			w := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
				},
				ReadWebhooks: func(_ repository.Repository, ids []int64) (settings.WebhookObservation, error) {
					reads++
					if !reflect.DeepEqual(ids, []int64{7, 8}) {
						t.Fatalf("ids=%v", ids)
					}
					o := webhookAppObservation()
					o.Target.Archived = true
					second := o.Values[0].Clone()
					second.ID = 8
					if reads == 2 {
						switch mode {
						case "second missing":
							return o, nil
						case "second differs":
							second.Config.ContentType = "application/x-www-form-urlencoded"
						case "second identity drift":
							second.CreatedAt = "other"
						}
					}
					o.Values = append(o.Values, second)
					return o, nil
				},
				Webhook:   func(repository.Repository, settings.WebhookSelection) (bool, error) { writes++; return false, nil },
				CreatedID: func(string, string, int64) error { writes++; return nil },
			}
			_, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, w)
			if (err == nil) != (mode == "matching") || writes != 0 || reads != 2 {
				t.Fatalf("err=%v reads=%d writes=%d", err, reads, writes)
			}
		})
	}
}
