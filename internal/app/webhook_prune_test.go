package app

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func appWebhookPrune(t *testing.T, archived bool, ids ...int64) settings.WebhookPruneObservation {
	t.Helper()
	states := []settings.WebhookPruneState{}
	for _, id := range ids {
		states = append(states, settings.WebhookPruneState{State: settings.WebhookState{ID: id, CreatedAt: "2026-11-14T00:00:00Z", Config: settings.WebhookConfig{PayloadURL: "https://private.test/webhook", ContentType: "application/json"}, Events: []string{"push"}}, UpdatedAt: "2026-11-14T01:00:00Z"})
	}
	o, err := settings.NewWebhookPruneObservation(settings.WebhookTarget{ID: 1, OwnerID: 2, FullName: "owner/current", Archived: archived}, states)
	if err != nil {
		t.Fatal(err)
	}
	return o
}
func TestWebhookPruneGlobalPreflightStopsUnrelatedWrites(t *testing.T) {
	for _, mode := range []string{"read-error", "missing-evidence", "missing-retained-id", "archived", "missing-adapter"} {
		t.Run(mode, func(t *testing.T) {
			description := "changed"
			unarchive := false
			hooks := []settings.Webhook{}
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &description, Archived: &unarchive}, Webhooks: &hooks}
			if mode == "missing-retained-id" {
				mime := "application/json"
				hooks = append(hooks, settings.Webhook{ID: 9, ContentType: &mime})
			}
			if mode == "archived" {
				d.Repository.Archived = nil
			}
			writer := Writer{ReadWebhookPrune: func(repository.Repository) (settings.WebhookPruneObservation, error) {
				if mode == "read-error" {
					return settings.WebhookPruneObservation{}, errors.New("unavailable")
				}
				if mode == "missing-evidence" {
					return settings.WebhookPruneObservation{}, nil
				}
				return appWebhookPrune(t, mode == "archived", 7), nil
			}, DeleteWebhook: func(repository.Repository, settings.WebhookPruneObservation, int64) (bool, error) {
				t.Fatal("delete before preflight")
				return false, nil
			}, Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) {
				t.Fatal("unrelated write before preflight")
				return false, nil
			}, ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
				t.Fatal("unarchive acquired before global preflight")
				return settings.ArchiveState{}, nil
			}}
			if mode == "missing-adapter" {
				writer.ReadWebhookPrune = nil
			}
			prune, _ := settings.ParsePruneScopes("webhooks")
			result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, writer, prune)
			if err == nil || result.Outcome != ApplyFailed || len(result.Operations) != 0 || !strings.Contains(err.Error(), "before any writes") {
				t.Fatal(result, err)
			}
		})
	}
}
func TestWebhookDeleteFixedCandidatesAndHonestProgress(t *testing.T) {
	for _, mode := range []string{"success", "first-read", "first-unknown", "second-read", "second-unknown", "missing-attempt"} {
		t.Run(mode, func(t *testing.T) {
			o, err := settings.PrepareWebhookPrune(appWebhookPrune(t, false, 7, 8), []settings.Webhook{})
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			result := ApplyResult{}
			writer := Writer{DeleteWebhook: func(_ repository.Repository, expected settings.WebhookPruneObservation, id int64) (bool, error) {
				calls++
				if id != int64(6+calls) {
					t.Fatal("candidate changed", id)
				}
				if calls == 2 {
					if _, found := expected.Find(7); found {
						t.Fatal("earlier deletion remained in expected inventory")
					}
				}
				if mode == "first-read" || mode == "second-read" && calls == 2 {
					return false, deployPreError{}
				}
				if mode == "first-unknown" || mode == "second-unknown" && calls == 2 {
					return true, errors.New("uncertain")
				}
				return mode != "missing-attempt", nil
			}}
			wrote, err := applyWebhookDeletes(&result, repository.Repository{}, o, writer)
			if mode == "success" {
				if err != nil || calls != 2 || !wrote {
					t.Fatal(wrote, calls, err)
				}
				return
			}
			if err == nil {
				t.Fatal("failure lost")
			}
			prior := strings.HasPrefix(mode, "second")
			if wrote != prior || (result.Outcome == ApplyPartialFailure) != prior {
				t.Fatal("incorrect partial progress", wrote, result)
			}
			last := result.Operations[len(result.Operations)-1].Status
			if (last == OperationFailedRead) != (strings.HasSuffix(mode, "read") || mode == "missing-attempt") {
				t.Fatal("incorrect failure status", last)
			}
		})
	}
}

func TestWebhookPruneComposesVerifiedWritesBeforeFixedDelete(t *testing.T) {
	for _, mode := range []string{"success", "writeback-failure", "neighbor-drift", "noop-drift"} {
		t.Run(mode, func(t *testing.T) {
			makeState := func(id int64, url string) settings.WebhookPruneState {
				return settings.WebhookPruneState{State: settings.WebhookState{ID: id, CreatedAt: "2026-11-14T00:00:00Z", Config: settings.WebhookConfig{PayloadURL: url, ContentType: "application/json"}, Active: true, Events: []string{"push"}}, UpdatedAt: "2026-11-14T01:00:00Z"}
			}
			target := settings.WebhookTarget{ID: 1, OwnerID: 2, FullName: "owner/current"}
			states := []settings.WebhookPruneState{makeState(7, "https://example.test/retained"), makeState(8, "https://example.test/candidate")}
			read := func() settings.WebhookPruneObservation {
				o, e := settings.NewWebhookPruneObservation(target, states)
				if e != nil {
					t.Fatal(e)
				}
				return o
			}
			changedType := "application/x-www-form-urlencoded"
			url := "https://example.test/new"
			mime := "application/json"
			active := true
			events := []string{"push"}
			desired := []settings.Webhook{{ID: 7, ContentType: &changedType}, {PayloadURL: &url, ContentType: &mime, Active: &active, Events: &events}}
			if mode == "noop-drift" {
				unchanged := "application/json"
				desired[0].ContentType = &unchanged
				desired = desired[:1]
			}
			frozen, e := settings.PrepareWebhookPrune(read(), desired)
			if e != nil {
				t.Fatal(e)
			}
			var calls []string
			writer := Writer{
				ReadWebhookPrune: func(repository.Repository) (settings.WebhookPruneObservation, error) { return read(), nil },
				Webhook: func(_ repository.Repository, selected settings.WebhookSelection) (bool, error) {
					calls = append(calls, "update")
					if selected.Expected == nil || !selected.Expected.Complete {
						t.Fatal("missing complete expected inventory")
					}
					if mode == "noop-drift" {
						states[0].UpdatedAt = "2026-11-14T02:00:00Z"
						return false, nil
					}
					states[0].State.Config.ContentType = changedType
					states[0].UpdatedAt = "2026-11-14T02:00:00Z"
					if mode == "neighbor-drift" {
						states[1].UpdatedAt = "2026-11-14T02:00:00Z"
					}
					return true, nil
				},
				CreateWebhook: func(_ repository.Repository, expected settings.WebhookObservation, d settings.Webhook) (settings.WebhookCreateResult, error) {
					calls = append(calls, "create")
					created := makeState(9, url)
					states = append(states, created)
					after, err := expected.WithCreated(created.State)
					if err != nil {
						t.Fatal(err)
					}
					return settings.WebhookCreateResult{Attempted: true, RecoveryID: 9, Observation: after}, nil
				},
				CreatedID: func(collection, name string, id int64) error {
					calls = append(calls, "record")
					if collection != "webhooks" || name != url || id != 9 {
						t.Fatal(collection, name, id)
					}
					if mode == "writeback-failure" {
						return errors.New("local failure")
					}
					return nil
				},
				DeleteWebhook: func(_ repository.Repository, expected settings.WebhookPruneObservation, id int64) (bool, error) {
					calls = append(calls, "delete")
					if id != 8 {
						t.Fatal("candidate widened", id)
					}
					if _, found := expected.Find(9); !found {
						t.Fatal("created ID not retained")
					}
					if _, found := expected.Find(7); !found {
						t.Fatal("retained ID missing")
					}
					return true, nil
				},
			}
			result := ApplyResult{}
			wrote, err := applyWebhooks(&result, repository.Repository{}, desired, writer, &frozen)
			want := []string{"update", "create", "record", "delete"}
			switch mode {
			case "writeback-failure":
				want = []string{"update", "create", "record"}
			case "neighbor-drift", "noop-drift":
				want = []string{"update"}
			}
			if !reflect.DeepEqual(calls, want) || (err == nil) != (mode == "success") || wrote != (mode != "noop-drift") {
				t.Fatal(calls, wrote, err, result)
			}
		})
	}
}
