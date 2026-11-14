package settings

import "testing"

func TestWebhookPruneScopeSelection(t *testing.T) {
	empty := []Webhook{}
	for _, option := range []string{"webhooks", "labels,webhooks", "all"} {
		prune, err := ParsePruneScopes(option)
		if err != nil || !prune.Has("webhooks") || !prune.Manages(Declaration{Webhooks: &empty}) || prune.Manages(Declaration{}) {
			t.Fatalf("webhook explicit prune selection %q: %v", option, err)
		}
	}
}

func webhookPruneState(id int64) WebhookPruneState {
	return WebhookPruneState{State: WebhookState{ID: id, CreatedAt: "2026-11-14T00:00:00Z", Config: WebhookConfig{PayloadURL: "https://example.test/hook", ContentType: "application/json"}, Active: true, Events: []string{"push", "issues"}}, UpdatedAt: "2026-11-14T01:00:00Z"}
}
func webhookPruneObservation(t *testing.T, values ...WebhookPruneState) WebhookPruneObservation {
	t.Helper()
	if values == nil {
		values = []WebhookPruneState{}
	}
	o, err := NewWebhookPruneObservation(WebhookTarget{ID: 1, OwnerID: 2, FullName: "owner/current"}, values)
	if err != nil {
		t.Fatal(err)
	}
	return o
}
func TestWebhookPruneFixedSetAndPrivateEvidence(t *testing.T) {
	original := webhookPruneObservation(t, webhookPruneState(8), webhookPruneState(7))
	mime := "application/json"
	prepared, err := PrepareWebhookPrune(original, []Webhook{{ID: 7, ContentType: &mime}})
	if err != nil {
		t.Fatal(err)
	}
	ids, err := prepared.Candidates()
	if err != nil || len(ids) != 1 || ids[0] != 8 {
		t.Fatal(ids, err)
	}
	ids[0] = 7
	if err := prepared.VerifyDelete(7, original); err == nil {
		t.Fatal("retained identity became a candidate")
	}
	if err := prepared.VerifyDelete(8, original); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"addition", "removal", "timestamp", "url", "mime", "ssl", "active", "events", "created", "owner", "repository", "archive"} {
		t.Run(mode, func(t *testing.T) {
			fresh := original.Clone()
			switch mode {
			case "addition":
				fresh.states = append(fresh.states, webhookPruneState(9))
			case "removal":
				fresh.states = fresh.states[:1]
			case "timestamp":
				fresh.states[1].UpdatedAt = "2026-11-14T02:00:00Z"
			case "url":
				fresh.states[1].State.Config.PayloadURL = "https://changed.test"
			case "mime":
				fresh.states[1].State.Config.ContentType = "application/x-www-form-urlencoded"
			case "ssl":
				fresh.states[1].State.Config.InsecureSSL = true
			case "active":
				fresh.states[1].State.Active = false
			case "events":
				fresh.states[1].State.Events = []string{"different"}
			case "created":
				fresh.states[1].State.CreatedAt = "2026-11-13T00:00:00Z"
			case "owner":
				fresh.target.OwnerID++
			case "repository":
				fresh.target.ID++
			case "archive":
				fresh.target.Archived = true
			}
			if err := prepared.VerifyDelete(8, fresh); err == nil {
				t.Fatal("retained or candidate drift accepted")
			}
		})
	}
	fresh := original.Clone()
	fresh.states[0].State.Events = []string{"issues", "push"}
	if err := prepared.VerifyDelete(8, fresh); err != nil {
		t.Fatal("event order changed meaning", err)
	}
	remaining, err := prepared.Without(8)
	if err != nil || remaining.Verify(webhookPruneObservation(t, webhookPruneState(7))) != nil {
		t.Fatal(err)
	}
	if err := remaining.Verify(original); err == nil {
		t.Fatal("absence was not required")
	}
	if _, err := remaining.Without(8); err == nil {
		t.Fatal("deleted ID reused")
	}
}
func TestWebhookPruneOwnEffectsOverlayOnlyWrittenObject(t *testing.T) {
	o, err := PrepareWebhookPrune(webhookPruneObservation(t, webhookPruneState(7), webhookPruneState(8)), []Webhook{{ID: 7, ContentType: webhookPtr("application/json")}})
	if err != nil {
		t.Fatal(err)
	}
	want := Webhook{ID: 7, ContentType: webhookPtr("application/x-www-form-urlencoded")}
	fresh := o.Clone()
	fresh.states[0].State.Config.ContentType = *want.ContentType
	fresh.states[0].UpdatedAt = "2026-11-14T02:00:00Z"
	updated, err := o.WithUpdated(want, fresh)
	if err != nil || updated.Verify(fresh) != nil {
		t.Fatal(err)
	}
	fresh.states[1].UpdatedAt = "2026-11-14T03:00:00Z"
	if _, err := o.WithUpdated(want, fresh); err == nil {
		t.Fatal("neighbor timestamp adopted")
	}
	created := webhookPruneState(9)
	fresh = updated.Clone()
	fresh.states = append(fresh.states, created)
	withNew, err := updated.WithCreated(created.State, fresh)
	if err != nil {
		t.Fatal(err)
	}
	ids, _ := withNew.Candidates()
	if len(ids) != 1 || ids[0] != 8 {
		t.Fatal("creation widened candidates", ids)
	}
	if err := withNew.VerifyDelete(9, fresh); err == nil {
		t.Fatal("created object selected for delete")
	}
	fresh.states[0].UpdatedAt = "2026-11-14T03:00:00Z"
	if _, err := updated.WithCreated(created.State, fresh); err == nil {
		t.Fatal("creation adopted neighbor drift")
	}
	archived := o.Clone()
	archived.target.Archived = true
	if archived.ValidateApply(false) == nil || archived.ValidateApply(true) != nil {
		t.Fatal("archive gate")
	}
	if archived.Verify(o) == nil || archived.AfterUnarchive().Verify(o) != nil {
		t.Fatal("only own unarchive permitted")
	}
}
func TestWebhookPruneRejectsMissingEvidence(t *testing.T) {
	for _, value := range []string{"", "PRIVATE_NOT_A_TIMESTAMP"} {
		s := webhookPruneState(7)
		s.UpdatedAt = value
		if _, err := NewWebhookPruneObservation(WebhookTarget{ID: 1, OwnerID: 2, FullName: "owner/current"}, []WebhookPruneState{s}); err == nil {
			t.Fatal("invalid timestamp accepted")
		}
	}
	if _, err := (WebhookPruneObservation{}).Candidates(); err == nil {
		t.Fatal("zero snapshot accepted")
	}
	if _, err := PrepareWebhookPrune(webhookPruneObservation(t), []Webhook{{ID: 7, ContentType: webhookPtr("application/json")}}); err == nil {
		t.Fatal("missing retained ID accepted")
	}
}
