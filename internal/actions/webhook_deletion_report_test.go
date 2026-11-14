package actions_test

import (
	"bytes"
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestWebhookDeletionReportUnionAndStatusPrivacy(t *testing.T) {
	deletedURL, mime, active, events := "https://deleted-canary.example/hook", "application/json", false, []string{"push", "future_event"}
	newURL := "https://created-canary.example/hook"
	change := settings.WebhookChange{Before: &settings.Webhook{ID: 9007199254740993, PayloadURL: &deletedURL, ContentType: &mime, Active: &active, Events: &events}}
	create := settings.WebhookChange{After: &settings.Webhook{PayloadURL: &newURL, ContentType: &mime, Active: &active, Events: &events}}
	path := filepath.Join(t.TempDir(), "report.json")
	var out, diag bytes.Buffer
	run := func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, Webhooks: []settings.WebhookChange{create, change}}}, nil
	}
	if status := actions.Run([]string{"plan", path}, &out, &diag, "dev", run); status != 0 {
		t.Fatal(diag.String())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var report actions.PlanReport
	if err = json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	rendered, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil || !strings.Contains(html.UnescapeString(rendered), deletedURL) || !strings.Contains(rendered, "9007199254740993") || !strings.Contains(rendered, "irreversible") || !strings.Contains(rendered, "opaque Secret") || !strings.Contains(rendered, "automatic ping") {
		t.Fatal(rendered, err)
	}
	for _, private := range []string{"created_at", "updated_at", "insecure_ssl", "secret"} {
		if strings.Contains(string(raw), private) {
			t.Fatal("private evidence", string(raw))
		}
	}
	t.Setenv("SETTINGS_TARGET", "owner/current")
	summary := filepath.Join(t.TempDir(), "summary.md")
	if status := actions.Run([]string{"summary", path, summary, "status-only"}, &out, &diag, "dev", nil); status != 0 {
		t.Fatal(diag.String())
	}
	hidden, err := os.ReadFile(summary)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"deleted-canary", "created-canary", "future_event", "9007199254740993"} {
		if strings.Contains(string(hidden)+out.String()+diag.String(), value) {
			t.Fatal("status-only leaked", value)
		}
	}
	for _, bad := range []string{
		`{"before":{"id":7,"payload_url":"https://example.test","content_type":"application/json","active":false}}`,
		`{"before":{"id":7,"payload_url":"https://example.test","content_type":"application/json","active":false,"events":[],"secret":"private"}}`,
		`{"before":{"id":7,"payload_url":"https://example.test","content_type":"application/json","active":false,"events":[]},"after":null}`,
		`{"before":{"id":7,"payload_url":"https://example.test","content_type":"application/json","active":false,"events":[],"id":8}}`,
		`{"before":{"payload_url":"https://example.test","content_type":"application/json","active":false,"events":[]}}`,
	} {
		var c settings.WebhookChange
		if json.Unmarshal([]byte(bad), &c) == nil {
			t.Fatal("accepted malformed deletion", bad)
		}
	}
	report.Webhooks = append(report.Webhooks, change)
	if _, err = actions.RenderSummary(report, summaryMetadata()); err == nil {
		t.Fatal("duplicate deletion accepted")
	}
}

func TestWebhookDeletionReportRejectsCreationCollision(t *testing.T) {
	url, mime, active, events := "https://hooks.example/old", "application/json", true, []string{"push"}
	before := settings.Webhook{ID: 7, PayloadURL: &url, ContentType: &mime, Active: &active, Events: &events}
	after := before.Clone()
	after.ID = 0
	if err := settings.ValidateWebhookChanges([]settings.WebhookChange{{Before: &before}, {After: &after}}); err == nil {
		t.Fatal("creation cannot reuse pending deletion URL")
	}
}
