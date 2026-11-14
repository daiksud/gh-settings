package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"testing"
)

func webhookAppCreate(url string) settings.Webhook {
	mime, active, events := "application/json", false, []string{}
	return settings.Webhook{PayloadURL: &url, ContentType: &mime, Active: &active, Events: &events}
}
func TestWebhookCreationProgressExpectedInventoryAndRecording(t *testing.T) {
	for _, mode := range []string{"mixed", "reversed", "two-creates", "guard", "unavailable", "prewrite", "uncertain", "bad-confirm", "local-failure", "late-failure"} {
		t.Run(mode, func(t *testing.T) {
			o := webhookAppObservation()
			want := "application/x-www-form-urlencoded"
			update := settings.Webhook{ID: 7, ContentType: &want}
			first := webhookAppCreate("https://new.example/one")
			second := webhookAppCreate("https://new.example/two")
			desired := []settings.Webhook{update, first}
			if mode == "reversed" {
				desired = []settings.Webhook{first, update}
			}
			if mode == "two-creates" || mode == "late-failure" {
				desired = []settings.Webhook{first, second}
			}
			if mode == "prewrite" || mode == "uncertain" || mode == "bad-confirm" || mode == "local-failure" {
				desired = []settings.Webhook{first}
			}
			if mode == "guard" {
				bad := true
				desired[0].Active = &bad
			}
			calls := []string{}
			created, recorded := 0, 0
			writer := Writer{ReadWebhooks: func(_ repository.Repository, ids []int64) (settings.WebhookObservation, error) {
				if ids != nil {
					t.Fatal("creation did not request full inventory")
				}
				if mode == "unavailable" {
					return settings.WebhookObservation{}, errors.New("unavailable")
				}
				return o, nil
			}, Webhook: func(_ repository.Repository, s settings.WebhookSelection) (bool, error) {
				calls = append(calls, "update")
				if s.Expected == nil || s.Expected.ValidateComplete() != nil {
					t.Fatal("mixed update lost inventory guard")
				}
				if mode == "reversed" && len(s.Expected.Values) != 2 {
					t.Fatal("own create missing from update expectation")
				}
				return true, nil
			}, CreateWebhook: func(_ repository.Repository, expected settings.WebhookObservation, d settings.Webhook) (settings.WebhookCreateResult, error) {
				calls = append(calls, "create")
				created++
				if mode == "mixed" && expected.Values[0].Config.ContentType != want {
					t.Fatal("own config update lost")
				}
				if created == 2 && len(expected.Values) != 2 {
					t.Fatal("own creation lost")
				}
				if mode == "prewrite" || mode == "late-failure" && created == 2 {
					return settings.WebhookCreateResult{}, deployPreError{}
				}
				id := int64(70 + created)
				result := settings.WebhookCreateResult{Attempted: true, RecoveryID: id}
				if mode == "uncertain" {
					return result, errors.New("unconfirmed")
				}
				s := settings.WebhookState{ID: id, CreatedAt: "new", Config: settings.WebhookConfig{PayloadURL: *d.PayloadURL, ContentType: *d.ContentType}, Active: *d.Active, Events: *d.Events}
				result.Observation, _ = expected.WithCreated(s)
				if mode == "bad-confirm" {
					result.Observation.Values[0].Config.PayloadURL = "https://outside.example"
				}
				return result, nil
			}, CreatedID: func(collection, name string, id int64) error {
				recorded++
				calls = append(calls, "record")
				if collection != "webhooks" || name == "" || id < 71 {
					t.Fatal("wrong local binding")
				}
				if mode == "local-failure" {
					return errors.New("file changed")
				}
				return nil
			}}
			result := ApplyResult{}
			wrote, err := applyWebhooks(&result, repository.Repository{}, desired, writer)
			success := mode == "mixed" || mode == "reversed" || mode == "two-creates"
			if (err == nil) != success {
				t.Fatal(err)
			}
			if mode == "guard" || mode == "unavailable" {
				if len(calls) != 0 || wrote {
					t.Fatal("preflight wrote")
				}
				return
			}
			if success {
				expectedCalls := []string{"update", "create", "record"}
				if mode == "reversed" {
					expectedCalls = []string{"create", "record", "update"}
				}
				if mode == "two-creates" {
					expectedCalls = []string{"create", "record", "create", "record"}
				}
				if !wrote || !reflect.DeepEqual(calls, expectedCalls) || len(result.CreatedIDs) != created {
					t.Fatal(calls, result)
				}
			} else {
				confirmed := mode == "local-failure" || mode == "late-failure"
				if wrote != confirmed || (result.Outcome == ApplyPartialFailure) != confirmed {
					t.Fatal(wrote, result.Outcome)
				}
				wantRecorded := 0
				if confirmed {
					wantRecorded = 1
				}
				if recorded != wantRecorded {
					t.Fatal("unverified ID recorded")
				}
				wantRecovery := 1
				if mode == "prewrite" {
					wantRecovery = 0
				}
				if len(result.CreatedIDs) != wantRecovery {
					t.Fatal("recovery evidence lost or invented")
				}
			}
		})
	}
}
func TestWebhookParentOnlyGuardsRemainSelectedAndNoop(t *testing.T) {
	active, events := false, []string{"push"}
	desired := []settings.Webhook{{ID: 7, Active: &active, Events: &events}}
	reads, writes := 0, 0
	result, err := Plan(func() (settings.Declaration, error) { return settings.Declaration{Webhooks: &desired}, nil }, func() (repository.Repository, error) { return repository.Repository{}, nil }, Reader{Webhooks: func(_ repository.Repository, ids []int64) (settings.WebhookObservation, error) {
		reads++
		if !reflect.DeepEqual(ids, []int64{7}) {
			t.Fatal(ids)
		}
		return webhookAppObservation(), nil
	}})
	if err != nil || reads != 1 || len(result.Comparison.Webhooks) != 0 {
		t.Fatal(err, result)
	}
	applied := ApplyResult{}
	_, err = applyWebhooks(&applied, repository.Repository{}, desired, Writer{ReadWebhooks: func(_ repository.Repository, ids []int64) (settings.WebhookObservation, error) {
		if !reflect.DeepEqual(ids, []int64{7}) {
			t.Fatal(ids)
		}
		return webhookAppObservation(), nil
	}, Webhook: func(_ repository.Repository, s settings.WebhookSelection) (bool, error) {
		if s.Expected != nil {
			t.Fatal("bound-only requires inventory")
		}
		writes++
		return false, nil
	}})
	if err != nil || writes != 1 || applied.Operations[1].Status != OperationUnchanged {
		t.Fatal(err, applied)
	}
}
