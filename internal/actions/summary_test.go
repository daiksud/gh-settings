package actions_test

import (
	"bytes"
	"encoding/json"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/actions"
)

func TestNoDifferenceSummary(t *testing.T) {
	content, err := actions.RenderSummary(actions.PlanReport{Outcome: "no-difference", Target: "owner/current", Managed: 5, Changes: []actions.PlanChange{}}, summaryMetadata())
	if err != nil || !strings.Contains(content, "No differences for managed settings.") || strings.Contains(content, "Observed before") {
		t.Fatalf("summary=%q error=%v", content, err)
	}
	if !strings.Contains(content, "Original actor") || !strings.Contains(content, "can overwrite intervening remote changes") || !strings.Contains(content, "does not prove publication") {
		t.Fatalf("review context missing: %s", content)
	}
}

func TestSummaryInputAndStorageFailures(t *testing.T) {
	for _, test := range []struct {
		name, report            string
		missingActor, directory bool
	}{
		{"missing report", "", false, false},
		{"malformed JSON", "{", false, false},
		{"missing count", `{"outcome":"unmanaged","changes":[]}`, false, false},
		{"null count", `{"outcome":"unmanaged","managed":null,"changes":[]}`, false, false},
		{"invalid state", `{"outcome":"unknown","managed":0,"changes":[]}`, false, false},
		{"missing actor", `{"outcome":"unmanaged","managed":0,"changes":[]}`, true, false},
		{"directory", `{"outcome":"unmanaged","managed":0,"changes":[]}`, false, true},
		{"failed plan and directory", `{"outcome":"invalid-config","managed":0,"changes":[],"diagnostic":"configuration validation failed"}`, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			setSummaryEnvironment(t)
			if test.missingActor {
				t.Setenv("SETTINGS_ACTOR", "")
			}
			reportPath := filepath.Join(t.TempDir(), "report.json")
			if test.report != "" {
				if err := os.WriteFile(reportPath, []byte(test.report), 0600); err != nil {
					t.Fatal(err)
				}
			}
			summaryPath := filepath.Join(t.TempDir(), "summary.md")
			const prior = "existing Summary\n"
			if test.directory {
				if err := os.Mkdir(summaryPath, 0700); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(summaryPath, []byte(prior), 0600); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			status := actions.Run([]string{"summary", reportPath, summaryPath}, &stdout, &stderr, "fixture", nil)
			if status != 1 || stdout.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
			if !test.directory {
				content, err := os.ReadFile(summaryPath)
				if err != nil || string(content) != prior {
					t.Fatalf("prior Summary changed: %q error=%v", content, err)
				}
			}
		})
	}
}

func TestSummaryRequiresNativeChanges(t *testing.T) {
	for _, test := range []struct {
		name   string
		change actions.PlanChange
	}{
		{"empty field", actions.PlanChange{Before: "before", After: "after"}},
		{"numeric pair", actions.PlanChange{Field: "repository.description", Before: float64(1), After: float64(2)}},
		{"mismatched pair", actions.PlanChange{Field: "repository.description", Before: "before", After: false}},
		{"null before", actions.PlanChange{Field: "repository.description", After: "after"}},
		{"description booleans", actions.PlanChange{Field: "repository.description", Before: true, After: false}},
		{"issues strings", actions.PlanChange{Field: "repository.has_issues", Before: "yes", After: "no"}},
		{"unknown field", actions.PlanChange{Field: "unknown", Before: "before", After: "after"}},
		{"equal strings", actions.PlanChange{Field: "repository.description", Before: "same", After: "same"}},
		{"equal booleans", actions.PlanChange{Field: "repository.has_issues", Before: false, After: false}},
	} {
		t.Run(test.name, func(t *testing.T) {
			report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{test.change}}
			content, err := actions.RenderSummary(report, summaryMetadata())
			if err == nil || content != "" {
				t.Fatalf("malformed native change accepted: content=%t error=%v", content != "", err)
			}
		})
	}
}

func TestSummaryRequiresValidReportState(t *testing.T) {
	empty := []actions.PlanChange{}
	change := []actions.PlanChange{{Field: "repository.description", Before: "before", After: "after"}}
	for _, test := range []struct {
		name   string
		report actions.PlanReport
	}{
		{"unknown outcome", actions.PlanReport{Outcome: "unknown", Changes: empty}},
		{"missing changes", actions.PlanReport{Outcome: "unmanaged"}},
		{"negative count", actions.PlanReport{Outcome: "unmanaged", Managed: -1, Changes: empty}},
		{"excess count", actions.PlanReport{Outcome: "no-difference", Target: "owner/current", Managed: 26, Changes: empty}},
		{"changed without changes", actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: empty}},
		{"no difference with changes", actions.PlanReport{Outcome: "no-difference", Target: "owner/current", Managed: 1, Changes: change}},
		{"unmanaged positive count", actions.PlanReport{Outcome: "unmanaged", Managed: 1, Changes: empty}},
		{"failure without diagnostic", actions.PlanReport{Outcome: "invalid-config", Changes: empty}},
		{"wrong diagnostic category", actions.PlanReport{Outcome: "invalid-config", Diagnostic: "repository read failed", Changes: empty}},
		{"arbitrary diagnostic", actions.PlanReport{Outcome: "target-failure", Diagnostic: "arbitrary failure text", Changes: empty}},
		{"read failure without target", actions.PlanReport{Outcome: "read-failure", Diagnostic: "repository read failed", Changes: empty}},
		{"target mismatch", actions.PlanReport{Outcome: "no-difference", Target: "other/repo", Managed: 1, Changes: empty}},
		{"duplicate field", actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 2, Changes: []actions.PlanChange{{Field: "repository.description", Before: "before", After: "after"}, {Field: "repository.description", Before: "before", After: "conflicting"}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			content, err := actions.RenderSummary(test.report, summaryMetadata())
			if err == nil || content != "" {
				t.Fatalf("invalid report state accepted: content=%t error=%v", content != "", err)
			}
		})
	}
}

func TestSummaryRequiresStructuredMetadata(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*actions.RunMetadata)
	}{
		{"workflow owner", func(m *actions.RunMetadata) { m.WorkflowRef = "/workflows/path.yml@refs/heads/main" }},
		{"workflow path", func(m *actions.RunMetadata) { m.WorkflowRef = "owner/workflows/@refs/heads/main" }},
		{"workflow ref", func(m *actions.RunMetadata) { m.WorkflowRef = "owner/workflows/path.yml@refs/" }},
		{"target repository", func(m *actions.RunMetadata) { m.Target = "owner" }},
		{"Action repository", func(m *actions.RunMetadata) { m.ActionRepository = "owner/repo/extra" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata := summaryMetadata()
			test.change(&metadata)
			content, err := actions.RenderSummary(actions.PlanReport{Outcome: "unmanaged", Changes: []actions.PlanChange{}}, metadata)
			if err == nil || content != "" {
				t.Fatalf("malformed metadata structure accepted: content=%t error=%v", content != "", err)
			}
		})
	}
}

func TestSummaryRequiresPositiveRunNumbers(t *testing.T) {
	for _, test := range []struct{ name, runID, attempt string }{
		{"nonnumeric run ID", "not-a-run", "1"},
		{"zero attempt", "123", "0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata := summaryMetadata()
			metadata.RunID, metadata.RunAttempt = test.runID, test.attempt
			content, err := actions.RenderSummary(actions.PlanReport{Outcome: "unmanaged", Changes: []actions.PlanChange{}}, metadata)
			if err == nil || content != "" {
				t.Fatalf("malformed run number accepted: content=%t error=%v", content != "", err)
			}
		})
	}
}

func TestSummaryRejectsMalformedHexIdentities(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*actions.RunMetadata)
	}{
		{"short workflow revision", func(m *actions.RunMetadata) { m.WorkflowSHA = "a" }},
		{"nonhex config revision", func(m *actions.RunMetadata) { m.ConfigSHA = strings.Repeat("g", 40) }},
		{"short CLI digest", func(m *actions.RunMetadata) { m.CLISHA256 = strings.Repeat("c", 63) }},
		{"nonhex CLI digest", func(m *actions.RunMetadata) { m.CLISHA256 = strings.Repeat("g", 64) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata := summaryMetadata()
			test.change(&metadata)
			content, err := actions.RenderSummary(actions.PlanReport{Outcome: "unmanaged", Changes: []actions.PlanChange{}}, metadata)
			if err == nil || content != "" {
				t.Fatalf("malformed identity accepted: content=%t error=%v", content != "", err)
			}
		})
	}
}

func TestSummaryRequiresMetadata(t *testing.T) {
	for _, test := range []struct {
		name  string
		clear func(*actions.RunMetadata)
	}{
		{"actor", func(m *actions.RunMetadata) { m.Actor = "" }},
		{"triggering actor", func(m *actions.RunMetadata) { m.TriggeringActor = "" }},
		{"event", func(m *actions.RunMetadata) { m.Event = "" }},
		{"run ID", func(m *actions.RunMetadata) { m.RunID = "" }},
		{"run attempt", func(m *actions.RunMetadata) { m.RunAttempt = "" }},
		{"workflow ref", func(m *actions.RunMetadata) { m.WorkflowRef = "" }},
		{"workflow SHA", func(m *actions.RunMetadata) { m.WorkflowSHA = "" }},
		{"config SHA", func(m *actions.RunMetadata) { m.ConfigSHA = "" }},
		{"target", func(m *actions.RunMetadata) { m.Target = "" }},
		{"CLI version", func(m *actions.RunMetadata) { m.CLIVersion = "" }},
		{"CLI SHA-256", func(m *actions.RunMetadata) { m.CLISHA256 = "" }},
		{"incomplete Action repository", func(m *actions.RunMetadata) { m.ActionRepository = "" }},
		{"incomplete Action ref", func(m *actions.RunMetadata) { m.ActionRef = "" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata := summaryMetadata()
			test.clear(&metadata)
			content, err := actions.RenderSummary(actions.PlanReport{Outcome: "unmanaged", Changes: []actions.PlanChange{}}, metadata)
			if err == nil || content != "" {
				t.Fatalf("missing metadata accepted: content=%t error=%v", content != "", err)
			}
		})
	}
}

func TestLocalActionSummary(t *testing.T) {
	metadata := summaryMetadata()
	metadata.ActionRepository, metadata.ActionRef = "", ""
	content, err := actions.RenderSummary(actions.PlanReport{Outcome: "unmanaged", Changes: []actions.PlanChange{}}, metadata)
	if err != nil || !strings.Contains(content, "Local checkout") || !strings.Contains(content, "Unavailable (local action)") || strings.Contains(content, "action-owner/action-repo") {
		t.Fatalf("local Action identity=%q error=%v", content, err)
	}
	if !strings.Contains(content, metadata.ConfigSHA) || !strings.Contains(content, metadata.WorkflowSHA) {
		t.Fatalf("audit identities missing: %s", content)
	}
}

func TestSummaryWorkflowLinkIdentity(t *testing.T) {
	for _, filename := range []string{"review@prod.yml", "review#prod.yaml", "review?prod.yml", "review%23prod.yaml"} {
		t.Run(filename, func(t *testing.T) {
			metadata := summaryMetadata()
			path := ".github/workflows/" + filename
			metadata.WorkflowRef = "workflow-owner/workflows/" + path + "@refs/heads/release@candidate"
			content, err := actions.RenderSummary(actions.PlanReport{Outcome: "no-difference", Target: metadata.Target, Managed: 1, Changes: []actions.PlanChange{}}, metadata)
			if err != nil {
				t.Fatal(err)
			}
			foundWorkflow, foundConfig := false, false
			for _, match := range regexp.MustCompile(`href=["']([^"']+)["']`).FindAllStringSubmatch(content, -1) {
				link, err := url.Parse(html.UnescapeString(match[1]))
				if err != nil || link.Scheme != "https" || link.Host != "github.com" || link.RawQuery != "" || link.Fragment != "" {
					t.Fatalf("unsafe or malformed link %q: %v", match[1], err)
				}
				foundWorkflow = foundWorkflow || link.Path == "/workflow-owner/workflows/blob/"+metadata.WorkflowSHA+"/"+path
				foundConfig = foundConfig || link.Path == "/owner/current/blob/"+metadata.ConfigSHA+"/.github/settings.yml"
			}
			if !foundWorkflow || !foundConfig || !strings.Contains(html.UnescapeString(content), metadata.WorkflowRef) {
				t.Fatalf("workflow/config identity lost for %q: %s", filename, content)
			}
		})
	}
}

func TestSummaryQuotesMultilineData(t *testing.T) {
	metadata := summaryMetadata()
	attack := "\"quotes\"\r\n\r\n</td><script>bad</script>\n\n## Forged approval\n\n[Approve](https://example.invalid)\n::notice::approved &copy;"
	metadata.Actor = attack
	value := "\"quote\"\n</pre>\n```\n## Forged values\n<script>bad</script>\n::notice::approved &copy;"
	report := actions.PlanReport{Outcome: "changed", Target: metadata.Target, Managed: 1, Changes: []actions.PlanChange{{Field: "repository.description", Before: "before", After: value}}}
	content, err := actions.RenderSummary(report, metadata)
	if err != nil || strings.Contains(content, "\r") || strings.Contains(content, "\n\n## Forged approval") || strings.Contains(content, "<script>bad</script>") || strings.Contains(content, "</pre>\n```") {
		t.Fatalf("untrusted data escaped its data context: error=%v content=%q", err, content)
	}
	visible := html.UnescapeString(content)
	if !strings.Contains(visible, attack) || !strings.Contains(visible, "Forged values") || !strings.Contains(visible, "::notice::approved") {
		t.Fatalf("quoted data lost meaning: %s", content)
	}
}

func TestFailedPlanSummary(t *testing.T) {
	setSummaryEnvironment(t)
	for _, test := range []struct{ outcome, target, diagnostic string }{
		{"read-failure", "owner/current", "repository read failed"},
		{"invalid-config", "", "configuration validation failed"},
		{"target-failure", "", "repository target resolution failed"},
	} {
		t.Run(test.outcome, func(t *testing.T) {
			root := t.TempDir()
			reportPath, summaryPath := filepath.Join(root, "plan.json"), filepath.Join(root, "Summary.md")
			data, err := json.Marshal(actions.PlanReport{Outcome: test.outcome, Target: test.target, Changes: []actions.PlanChange{}, Diagnostic: test.diagnostic})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(reportPath, data, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(summaryPath, []byte("existing Summary\n"), 0600); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			status := actions.Run([]string{"summary", reportPath, summaryPath}, &stdout, &stderr, "dev", nil)
			data, err = os.ReadFile(summaryPath)
			if err != nil {
				t.Fatal(err)
			}
			written := string(data)
			if status != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "plan failed") || !strings.Contains(written, "Error — plan failed.") {
				t.Fatalf("status=%d stdout=%q stderr=%q errorHeading=%t", status, stdout.String(), stderr.String(), strings.Contains(written, "Error — plan failed."))
			}
			if !strings.HasPrefix(written, "existing Summary\n") || !strings.Contains(written, test.diagnostic) || strings.Contains(written, "Observed before") || !strings.Contains(written, "can overwrite intervening remote changes") || !strings.Contains(written, "does not prove publication") {
				t.Fatalf("failed plan review content missing: %s", written)
			}
		})
	}
}

func setSummaryEnvironment(t *testing.T) {
	t.Helper()
	m := summaryMetadata()
	for key, value := range map[string]string{
		"SETTINGS_ACTOR": m.Actor, "SETTINGS_TRIGGERING_ACTOR": m.TriggeringActor, "SETTINGS_EVENT": m.Event,
		"SETTINGS_RUN_ID": m.RunID, "SETTINGS_RUN_ATTEMPT": m.RunAttempt, "SETTINGS_WORKFLOW_REF": m.WorkflowRef, "SETTINGS_WORKFLOW_SHA": m.WorkflowSHA,
		"SETTINGS_CONFIG_SHA": m.ConfigSHA, "SETTINGS_TARGET": m.Target, "SETTINGS_ACTION_REPOSITORY": m.ActionRepository, "SETTINGS_ACTION_REF": m.ActionRef,
		"SETTINGS_CLI_VERSION": m.CLIVersion, "SETTINGS_CLI_SHA256": m.CLISHA256, "GH_TOKEN": "", "GITHUB_TOKEN": "",
	} {
		t.Setenv(key, value)
	}
}

func TestUnmanagedSummary(t *testing.T) {
	metadata := summaryMetadata()
	content, err := actions.RenderSummary(actions.PlanReport{Outcome: "unmanaged", Changes: []actions.PlanChange{}}, metadata)
	if err != nil || !strings.Contains(content, "No managed settings.") || strings.Contains(content, "Observed before") || strings.Contains(content, "No differences") {
		t.Fatalf("summary=%q error=%v", content, err)
	}
	if !strings.Contains(content, "https://github.com/owner/current/blob/"+metadata.ConfigSHA+"/.github/settings.yml") || !strings.Contains(content, "can overwrite intervening remote changes") {
		t.Fatalf("audit metadata or warning missing: %s", content)
	}
}

func summaryMetadata() actions.RunMetadata {
	return actions.RunMetadata{
		Actor: "owner", TriggeringActor: "maintainer", Event: "push", RunID: "123", RunAttempt: "2",
		WorkflowRef: "workflow-owner/workflows/.github/workflows/settings-main.yml@refs/heads/main", WorkflowSHA: strings.Repeat("a", 40),
		ConfigSHA: strings.Repeat("b", 40), Target: "owner/current", ActionRepository: "action-owner/action-repo", ActionRef: strings.Repeat("d", 40),
		CLIVersion: "v0.6.1", CLISHA256: strings.Repeat("c", 64),
	}
}

func TestTopicsSummaryNativeAndJSONLists(t *testing.T) {
	for _, change := range []actions.PlanChange{
		{Field: "topics.names", Before: []string{"old"}, After: []string{}},
		{Field: "topics.names", Before: []any{"old"}, After: []any{}},
	} {
		report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{change}}
		content, err := actions.RenderSummary(report, summaryMetadata())
		if err != nil || !strings.Contains(content, `settings.topics.names: [&#34;old&#34;] → []`) {
			t.Fatalf("content=%s error=%v", content, err)
		}
	}
	for _, change := range []actions.PlanChange{
		{Field: "topics.names", Before: nil, After: []string{}},
		{Field: "topics.names", Before: []string{}, After: nil},
		{Field: "topics.names", Before: []any{false}, After: []any{}},
		{Field: "topics.names", Before: []string{"z", "a"}, After: []string{}},
		{Field: "topics.names", Before: []string{"a", "a"}, After: []string{}},
		{Field: "topics.names", Before: []string{"<script>alert(1)</script>"}, After: []string{}},
		{Field: "topics.names", Before: []string{"a\nb"}, After: []string{}},
		{Field: "topics.names", Before: []string{"a"}, After: []any{"a"}},
		{Field: "topics.names", Before: []string{}, After: []any{}},
	} {
		report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{change}}
		content, err := actions.RenderSummary(report, summaryMetadata())
		if err == nil || content != "" {
			t.Fatalf("accepted %+v: %s %v", change, content, err)
		}
	}
}

func TestInvalidTopicsSummaryPreservesExistingFile(t *testing.T) {
	setSummaryEnvironment(t)
	root := t.TempDir()
	reportPath := filepath.Join(root, "plan.json")
	summaryPath := filepath.Join(root, "summary.md")
	report := `{"outcome":"changed","target":"owner/current","managed":1,"changes":[{"field":"topics.names","before":[],"after":["<script>"]}]}`
	if err := os.WriteFile(reportPath, []byte(report), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(summaryPath, []byte("existing summary"), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	status := actions.Run([]string{"summary", reportPath, summaryPath}, &stdout, &stderr, "dev", nil)
	data, err := os.ReadFile(summaryPath)
	if status != 1 || stdout.Len() != 0 || stderr.Len() == 0 || err != nil || string(data) != "existing summary" {
		t.Fatalf("status=%d summary=%q error=%v stderr=%s", status, data, err, stderr.String())
	}
}
