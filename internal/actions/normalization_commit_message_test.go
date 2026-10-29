package actions_test

import (
	"bytes"
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/cli"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestNormalizedCommitMenuReportRetainsOnePairedDecision(t *testing.T) {
	for _, test := range []struct {
		menu                        settings.CommitMessageMenu
		path, title, message, field string
	}{
		{settings.SquashCommitMenu, "repository.settings.general.pull_requests.squash_merge.default_commit_message", "COMMIT_OR_PR_TITLE", "BLANK", "repository.squash_merge_commit_title"},
		{settings.MergeCommitMenu, "repository.settings.general.pull_requests.merge_commit.default_commit_message", "MERGE_MESSAGE", "BLANK", "repository.merge_commit_title"},
	} {
		normalization := settings.CommitMessageNormalization{Path: test.path, RawTitle: test.title, RawMessage: test.message, Fallback: uimodel.DefaultMessage}
		decision, err := settings.NewNormalizedCommitMessageChange(test.menu, normalization, uimodel.DefaultMessage)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "plan.json")
		var stdout, stderr bytes.Buffer
		status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
			return app.PlanResult{Target: repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 2, CommitMessages: []settings.NormalizedCommitMessageChange{decision}}}, nil
		})
		data, err := os.ReadFile(path)
		if err != nil || status != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
			t.Fatal("report write failed", status, err, stdout.String(), stderr.String())
		}
		var report actions.PlanReport
		if err := json.Unmarshal(data, &report); err != nil {
			t.Fatal(err)
		}
		if report.Outcome != "changed" || report.Managed != 2 || len(report.Changes) != 0 || len(report.CommitMessages) != 1 || report.CommitMessages[0].Menu != test.menu || report.CommitMessages[0].After != "default_message" || report.CommitMessages[0].Normalization != normalization {
			t.Fatal("normalized menu became no-difference or lost paired provenance", report)
		}
		if _, err := actions.RenderSummary(report, summaryMetadata()); err != nil {
			t.Fatal("closed paired decision failed reconstruction", err)
		}
		mixed := report
		mixed.Managed = 3
		mixed.Changes = []actions.PlanChange{{Field: "repository.description", Before: "before", After: "after"}}
		if _, err := actions.RenderSummary(mixed, summaryMetadata()); err != nil {
			t.Fatal("unrelated native scalar was treated as menu overlap", err)
		}
		for _, failure := range []string{"count", "duplicate", "overlap", "missing provenance", "wrong path", "supported raw", "menu", "desired"} {
			invalid := report
			invalid.CommitMessages = append([]actions.CommitMessagePlanChange{}, report.CommitMessages...)
			switch failure {
			case "count":
				invalid.Managed = 1
			case "duplicate":
				invalid.CommitMessages = append(invalid.CommitMessages, invalid.CommitMessages[0])
				invalid.Managed = 4
			case "overlap":
				invalid.Changes = []actions.PlanChange{{Field: test.field, Before: test.title, After: "PR_TITLE"}}
				invalid.Managed = 3
			case "missing provenance":
				invalid.CommitMessages[0].Normalization = settings.CommitMessageNormalization{}
			case "wrong path":
				invalid.CommitMessages[0].Normalization.Path = "unmanaged"
			case "supported raw":
				invalid.CommitMessages[0].Normalization.RawTitle, invalid.CommitMessages[0].Normalization.RawMessage = "PR_TITLE", "BLANK"
			case "menu":
				invalid.CommitMessages[0].Menu = settings.CommitMessageMenu("future_menu")
			case "desired":
				invalid.CommitMessages[0].After = uimodel.CommitMessageChoice("future_choice")
			}
			if _, err := actions.RenderSummary(invalid, summaryMetadata()); err == nil {
				t.Fatal("inconsistent paired report accepted", failure)
			}
		}
	}
}

func TestNormalizedCommitMenuPresentationShowsOneSafeWarning(t *testing.T) {
	rawTitle, rawMessage := "<script>PAIR-CANARY</script>\n::warning::untrusted", "FUTURE&\nMESSAGE"
	for _, test := range []struct {
		menu               settings.CommitMessageMenu
		path, nativePrefix string
	}{
		{settings.SquashCommitMenu, "repository.settings.general.pull_requests.squash_merge.default_commit_message", "repository.squash_merge_commit_"},
		{settings.MergeCommitMenu, "repository.settings.general.pull_requests.merge_commit.default_commit_message", "repository.merge_commit_"},
	} {
		for _, desired := range []uimodel.CommitMessageChoice{uimodel.DefaultMessage, uimodel.PullRequestTitle} {
			normalization := settings.CommitMessageNormalization{Path: test.path, RawTitle: rawTitle, RawMessage: rawMessage, Fallback: uimodel.DefaultMessage}
			decision, err := settings.NewNormalizedCommitMessageChange(test.menu, normalization, desired)
			if err != nil {
				t.Fatal(err)
			}
			output := string(cli.RenderComparison(settings.Comparison{Managed: 2, CommitMessages: []settings.NormalizedCommitMessageChange{decision}}))
			if strings.Count(output, "Warning:") != 1 || strings.Count(output, "write required") != 1 || !strings.Contains(output, test.path) || !strings.Contains(output, strconv.Quote(rawTitle)) || !strings.Contains(output, strconv.Quote(rawMessage)) || !strings.Contains(output, `"default_message"`) || !strings.Contains(output, strconv.Quote(string(desired))) || strings.Contains(output, test.nativePrefix) || strings.Contains(output, "No differences") || strings.Contains(output, "\n::warning::") {
				t.Fatal("paired menu warning was lost, duplicated or unsafe", output)
			}
			report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 2, Changes: []actions.PlanChange{}, CommitMessages: []actions.CommitMessagePlanChange{{Menu: test.menu, After: desired, Normalization: normalization}}}
			summary, err := actions.RenderSummary(report, summaryMetadata())
			if err != nil || !strings.Contains(html.UnescapeString(summary), strings.TrimSuffix(output, "\n")) || strings.Contains(summary, "<script>") || !strings.Contains(summary, "&lt;script&gt;") {
				t.Fatal("paired Summary lost safe UI warning", summary, err)
			}
			data, _ := json.Marshal(report)
			path, privatePath := filepath.Join(t.TempDir(), "plan.json"), filepath.Join(t.TempDir(), "summary.md")
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("SETTINGS_TARGET", "owner/current")
			var stdout, stderr bytes.Buffer
			status := actions.Run([]string{"summary", path, privatePath, "status-only"}, &stdout, &stderr, "dev", nil)
			private, err := os.ReadFile(privatePath)
			if err != nil || status != 0 || stderr.Len() != 0 || stdout.String() != "plan-outcome=changed\n" || !strings.Contains(string(private), "Outcome: changed") || strings.Contains(string(private), "PAIR-CANARY") || strings.Contains(string(private), test.path) || strings.Contains(string(private), "default_message") {
				t.Fatal("status-only output disclosed paired policy", status, string(private), stdout.String(), stderr.String(), err)
			}
		}
	}
}
