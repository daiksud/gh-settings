package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"testing"
)

func webhookAppObservation() settings.WebhookObservation {
	return settings.WebhookObservation{Target: settings.WebhookTarget{ID: 1, OwnerID: 2, FullName: "owner/current"}, Values: []settings.WebhookState{{ID: 7, CreatedAt: "stamp", Config: settings.WebhookConfig{PayloadURL: "https://private.example/omitted", ContentType: "application/json"}, Events: []string{"push"}}}}
}
func TestWebhooksLocalNoopAndSelection(t *testing.T) {
	for _, d := range []settings.Declaration{{}, {Webhooks: &[]settings.Webhook{}}} {
		load := func() (settings.Declaration, error) { return d, nil }
		resolve := func() (repository.Repository, error) {
			t.Fatal("local noop resolved target")
			return repository.Repository{}, nil
		}
		if _, e := Plan(load, resolve, Reader{}); e != nil {
			t.Fatal(e)
		}
		if _, e := Apply(load, resolve, Writer{}); e != nil {
			t.Fatal(e)
		}
	}
	want := "application/x-www-form-urlencoded"
	desired := []settings.Webhook{{ID: 7, ContentType: &want}}
	result, e := Plan(func() (settings.Declaration, error) { return settings.Declaration{Webhooks: &desired}, nil }, func() (repository.Repository, error) {
		return repository.Repository{Owner: "owner", Name: "current"}, nil
	}, Reader{Webhooks: func(_ repository.Repository, ids []int64) (settings.WebhookObservation, error) {
		if !reflect.DeepEqual(ids, []int64{7}) {
			t.Fatal(ids)
		}
		return webhookAppObservation(), nil
	}})
	if e != nil || result.Comparison.Managed != 1 || len(result.Comparison.Webhooks) != 1 || result.Comparison.Webhooks[0].Before.PayloadURL != nil {
		t.Fatal(e, result)
	}
}
func TestWebhookApplyHonestProgress(t *testing.T) {
	for _, mode := range []string{"noop", "write", "prewrite", "unknown", "after-noop", "after-write", "last-read"} {
		t.Run(mode, func(t *testing.T) {
			want := "application/x-www-form-urlencoded"
			d := []settings.Webhook{{ID: 7, ContentType: &want}, {ID: 8, ContentType: &want}}
			o := webhookAppObservation()
			two := o.Values[0].Clone()
			two.ID = 8
			o.Values = append(o.Values, two)
			calls := 0
			result := ApplyResult{}
			writer := Writer{ReadWebhooks: func(repository.Repository, []int64) (settings.WebhookObservation, error) {
				if mode == "last-read" {
					return settings.WebhookObservation{}, errors.New("unavailable")
				}
				return o, nil
			}, Webhook: func(_ repository.Repository, s settings.WebhookSelection) (bool, error) {
				calls++
				if s.Desired.PayloadURL != nil {
					t.Fatal("omitted control became intent")
				}
				switch mode {
				case "prewrite":
					return false, deployPreError{}
				case "unknown":
					return true, errors.New("uncertain")
				case "after-noop", "after-write":
					if calls == 2 {
						return false, deployPreError{}
					}
				}
				return mode == "write" || mode == "after-write", nil
			}}
			wrote, e := applyWebhooks(&result, repository.Repository{}, d, writer)
			if mode == "noop" || mode == "write" {
				if e != nil || calls != 2 || wrote != (mode == "write") {
					t.Fatal(e, calls, wrote)
				}
				return
			}
			if e == nil {
				t.Fatal("failure lost")
			}
			partial := mode == "after-write"
			if (result.Outcome == ApplyPartialFailure) != partial {
				t.Fatal(result.Outcome)
			}
			if mode == "unknown" && result.Operations[1].Status != OperationFailedUnknown {
				t.Fatal(result.Operations)
			}
			if mode == "last-read" && calls != 0 {
				t.Fatal("incomplete selection wrote")
			}
		})
	}
}
func TestWebhookExportRetentionAndDiagnosticFailure(t *testing.T) {
	for _, mode := range []string{"complete", "empty", "initial", "retained", "projection", "diagnostic"} {
		t.Run(mode, func(t *testing.T) {
			url := "https://prior.example"
			prior := []settings.Webhook{{ID: 7, PayloadURL: &url}}
			existing := settings.Declaration{}
			if mode == "retained" {
				existing.Webhooks = &prior
			}
			var saved settings.Declaration
			saveCount, infos := 0, 0
			sentinel := errors.New("output failed")
			read := Reader{Webhooks: func(_ repository.Repository, ids []int64) (settings.WebhookObservation, error) {
				if ids != nil {
					t.Fatal("not full inventory")
				}
				if mode == "initial" || mode == "retained" || mode == "diagnostic" {
					return settings.WebhookObservation{}, errors.New("unavailable")
				}
				o := webhookAppObservation()
				if mode == "empty" {
					o.Values = []settings.WebhookState{}
				}
				return o, nil
			}, Topics: func(repository.Repository) ([]string, error) { return []string{"later"}, nil }}
			check := ExportCheckout{Existing: existing, Repository: func() (repository.Repository, error) { return repository.Repository{}, nil }, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error { saved = d; saveCount++; return nil }, nil
			}, Info: func(path string, _ error) error {
				if path == settings.WebhooksPath {
					infos++
					if mode == "diagnostic" {
						return sentinel
					}
				}
				return nil
			}, Check: func(d settings.Declaration) error {
				if mode == "projection" && d.Webhooks != nil {
					return errors.New("projection unavailable")
				}
				return nil
			}}
			e := Export(check, read)
			if mode == "diagnostic" {
				if !errors.Is(e, sentinel) || saveCount != 0 {
					t.Fatal(e, saveCount)
				}
				return
			}
			if e != nil || saveCount != 1 || saved.Topics.Names == nil {
				t.Fatal(e, saveCount)
			}
			switch mode {
			case "complete":
				if saved.Webhooks == nil || len(*saved.Webhooks) != 1 || (*saved.Webhooks)[0].ContentType == nil {
					t.Fatal("incomplete export")
				}
			case "empty":
				if saved.Webhooks == nil || len(*saved.Webhooks) != 0 {
					t.Fatal("empty lost")
				}
			case "initial", "projection":
				if saved.Webhooks != nil || infos != 1 {
					t.Fatal("unavailable guessed")
				}
			case "retained":
				if !reflect.DeepEqual(saved.Webhooks, &prior) || infos != 1 {
					t.Fatal("whole prior lost")
				}
			}
		})
	}
}
