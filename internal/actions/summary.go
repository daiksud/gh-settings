package actions

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/daiksud/gh-settings/internal/presentation"
)

// RunMetadata is explicit nonsecret review context, separate from plan data.
type RunMetadata struct {
	Actor, TriggeringActor, Event string
	RunID, RunAttempt             string
	WorkflowRef, WorkflowSHA      string
	ConfigSHA, Target             string
	ActionRepository, ActionRef   string
	CLIVersion, CLISHA256         string
}

func runMetadataFromEnv() RunMetadata {
	return RunMetadata{
		Actor: os.Getenv("SETTINGS_ACTOR"), TriggeringActor: os.Getenv("SETTINGS_TRIGGERING_ACTOR"), Event: os.Getenv("SETTINGS_EVENT"),
		RunID: os.Getenv("SETTINGS_RUN_ID"), RunAttempt: os.Getenv("SETTINGS_RUN_ATTEMPT"),
		WorkflowRef: os.Getenv("SETTINGS_WORKFLOW_REF"), WorkflowSHA: os.Getenv("SETTINGS_WORKFLOW_SHA"),
		ConfigSHA: os.Getenv("SETTINGS_CONFIG_SHA"), Target: os.Getenv("SETTINGS_TARGET"),
		ActionRepository: os.Getenv("SETTINGS_ACTION_REPOSITORY"), ActionRef: os.Getenv("SETTINGS_ACTION_REF"),
		CLIVersion: os.Getenv("SETTINGS_CLI_VERSION"), CLISHA256: os.Getenv("SETTINGS_CLI_SHA256"),
	}
}

// RenderSummary produces review content; rendering does not establish publication.
func RenderSummary(report PlanReport, metadata RunMetadata) (string, error) {
	for name, value := range map[string]string{
		"SETTINGS_ACTOR": metadata.Actor, "SETTINGS_TRIGGERING_ACTOR": metadata.TriggeringActor, "SETTINGS_EVENT": metadata.Event,
		"SETTINGS_RUN_ID": metadata.RunID, "SETTINGS_RUN_ATTEMPT": metadata.RunAttempt,
		"SETTINGS_WORKFLOW_REF": metadata.WorkflowRef, "SETTINGS_WORKFLOW_SHA": metadata.WorkflowSHA,
		"SETTINGS_CONFIG_SHA": metadata.ConfigSHA, "SETTINGS_TARGET": metadata.Target,
		"SETTINGS_CLI_VERSION": metadata.CLIVersion, "SETTINGS_CLI_SHA256": metadata.CLISHA256,
	} {
		if value == "" {
			return "", fmt.Errorf("missing %s", name)
		}
	}
	if (metadata.ActionRepository == "") != (metadata.ActionRef == "") {
		return "", fmt.Errorf("incomplete Action repository/ref metadata")
	}
	for _, identity := range []struct {
		name, value string
		size        int
	}{
		{"SETTINGS_WORKFLOW_SHA", metadata.WorkflowSHA, 40},
		{"SETTINGS_CONFIG_SHA", metadata.ConfigSHA, 40},
		{"SETTINGS_CLI_SHA256", metadata.CLISHA256, 64},
	} {
		if _, err := hex.DecodeString(identity.value); err != nil || len(identity.value) != identity.size {
			return "", fmt.Errorf("invalid %s", identity.name)
		}
	}
	for name, value := range map[string]string{"SETTINGS_RUN_ID": metadata.RunID, "SETTINGS_RUN_ATTEMPT": metadata.RunAttempt} {
		if number, err := strconv.ParseUint(value, 10, 64); err != nil || number == 0 {
			return "", fmt.Errorf("invalid %s", name)
		}
	}
	for name, value := range map[string]string{"SETTINGS_TARGET": metadata.Target, "SETTINGS_ACTION_REPOSITORY": metadata.ActionRepository} {
		if value == "" {
			continue
		}
		parts := strings.Split(value, "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", fmt.Errorf("invalid %s", name)
		}
	}
	source, ref, ok := strings.Cut(metadata.WorkflowRef, "@refs/")
	workflow := strings.SplitN(source, "/", 3)
	if !ok || ref == "" || len(workflow) != 3 || workflow[0] == "" || workflow[1] == "" || workflow[2] == "" {
		return "", fmt.Errorf("invalid workflow reference")
	}
	validated, err := validatePlanReport(report, metadata.Target)
	if err != nil {
		return "", err
	}
	return renderSummary(validated, metadata, workflow), nil
}

func renderSummary(report validatedPlanReport, metadata RunMetadata, workflow []string) string {
	workflowRepository := workflow[0] + "/" + workflow[1]
	actionRepository, actionRef := metadata.ActionRepository, metadata.ActionRef
	if actionRepository == "" && actionRef == "" {
		actionRepository, actionRef = "Local checkout", "Unavailable (local action)"
	}
	rows := []struct{ label, value string }{
		{"Original actor", quoteHTML(metadata.Actor)},
		{"Initiating / rerun actor", quoteHTML(metadata.TriggeringActor)},
		{"Event", quoteHTML(metadata.Event)},
		{"Run / attempt", summaryLink(workflowRepository+"/actions/runs/"+metadata.RunID, metadata.RunID) + " / " + quoteHTML(metadata.RunAttempt)},
		{"Workflow path / ref", quoteHTML(metadata.WorkflowRef)},
		{"Workflow revision", summaryLink(workflowRepository+"/blob/"+metadata.WorkflowSHA+"/"+workflow[2], metadata.WorkflowSHA)},
		{"Target repository", summaryLink(metadata.Target, metadata.Target)},
		{"Configuration revision", summaryLink(metadata.Target+"/blob/"+metadata.ConfigSHA+"/.github/settings.yml", metadata.ConfigSHA)},
		{"Action repository", quoteHTML(actionRepository)},
		{"Action ref", quoteHTML(actionRef)},
		{"CLI version", quoteHTML(metadata.CLIVersion)},
		{"CLI SHA-256", quoteHTML(metadata.CLISHA256)},
	}
	var content strings.Builder
	content.WriteString("\n\n## Settings review\n\n<table>\n")
	for _, row := range rows {
		fmt.Fprintf(&content, "<tr><th>%s</th><td>%s</td></tr>\n", quoteHTML(row.label), row.value)
	}
	result, failed := summaryOutcome(report.outcome)
	fmt.Fprintf(&content, "</table>\n\n<p>%s</p>\n\n<pre>", result)
	if failed {
		if report.resource != "" {
			content.WriteString(quoteHTML(report.resource + ": "))
		}
		content.WriteString(quoteHTML(report.diagnostic))
	}
	writeLine := func(line string) { fmt.Fprintln(&content, quoteHTML(line)) }
	presentation.RenderRepositoryChanges(&report.comparison, writeLine)
	presentation.RenderChanges(report.comparison.CopilotInstructions, presentation.CopilotInstructionsLine, writeLine)
	presentation.RenderChanges(report.comparison.Runners, presentation.RunnerLabelsLine, writeLine)
	presentation.RenderChanges(report.comparison.RunnerDeletions, presentation.RunnerDeletionLine, writeLine)
	presentation.RenderChanges(report.comparison.OIDCSubject, presentation.OIDCSubjectLine, writeLine)
	presentation.RenderChanges(report.comparison.CopilotMCP, presentation.CopilotMCPLine, writeLine)
	presentation.RenderChanges(report.comparison.PullRequestExemptions, presentation.PullRequestExemptionsLine, writeLine)
	presentation.RenderChanges(report.comparison.PullRequestLimits, presentation.PullRequestLimitsLine, writeLine)
	presentation.RenderChanges(report.comparison.CopilotReviewEffort, presentation.CopilotReviewEffortLine, writeLine)
	presentation.RenderChanges(report.comparison.Variables, presentation.VariableLine, writeLine)
	presentation.RenderChanges(report.comparison.EnvironmentVariables, presentation.EnvironmentVariableLine, writeLine)
	presentation.RenderChanges(report.comparison.AgentsVariables, presentation.AgentVariableLine, writeLine)
	presentation.RenderChanges(report.comparison.Webhooks, presentation.WebhookLine, writeLine)
	presentation.RenderChanges(report.comparison.DeployKeys, presentation.DeployKeyLine, writeLine)
	presentation.RenderChanges(report.comparison.CollaboratorDeletions, presentation.CollaboratorDeletionLine, writeLine)
	presentation.RenderChanges(report.comparison.Collaborators, presentation.CollaboratorLine, writeLine)
	presentation.RenderChanges(report.comparison.Labels.Values(), presentation.LabelLine, writeLine)
	presentation.RenderChanges(report.comparison.ActionsPolicies, presentation.ActionsEventPolicyLine, writeLine)
	presentation.RenderChanges(report.rulesets, presentation.RulesetLine, writeLine)
	presentation.RenderChanges(report.comparison.Autolinks, presentation.AutolinkLine, writeLine)
	presentation.RenderChanges(report.comparison.BranchProtectionRules, presentation.ClassicProtectionLine, writeLine)
	presentation.RenderChanges(report.comparison.Environments, presentation.EnvironmentLine, writeLine)
	presentation.RenderActionsPolicyChanges(&report.comparison, writeLine)
	presentation.RenderChanges(report.comparison.Security, presentation.SecurityLine, writeLine)
	presentation.RenderChanges(report.comparison.CodeQLConfiguration, presentation.CodeQLConfigurationLine, writeLine)
	presentation.RenderChanges(report.comparison.Pages, presentation.PagesLine, writeLine)
	content.WriteString("</pre>\n\nDirect apply uses the declaration and can overwrite intervening remote changes to managed fields. Review this Summary before authorizing a separate direct apply; successful rendering alone does not prove publication.\n")
	return content.String()
}

func summaryOutcome(outcome string) (label string, failed bool) {
	switch outcome {
	case "no-difference":
		return "No differences for managed settings.", false
	case "unmanaged":
		return "No managed settings.", false
	case "invalid-config", "target-failure", "read-failure":
		return "Error — plan failed.", true
	default:
		return "Observed before → desired after.", false
	}
}

func quoteHTML(value string) string {
	quoted := html.EscapeString(value)
	// Raw blank lines can end a GFM HTML data block.
	return strings.NewReplacer("\r", "&#13;", "\n", "&#10;").Replace(quoted)
}

func summaryLink(path, label string) string {
	u := url.URL{Scheme: "https", Host: "github.com", Path: "/" + path}
	return `<a href="` + quoteHTML(u.String()) + `">` + quoteHTML(label) + `</a>`
}

func writeSummary(reportPath, summaryPath string) error {
	return writeSummaryMode(reportPath, summaryPath, "detailed", nil)
}

func writeSummaryMode(reportPath, summaryPath, mode string, output io.Writer) error {
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return fmt.Errorf("read plan report: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("decode plan report: %w", err)
	}
	if _, present := fields["custom_properties"]; present {
		return fmt.Errorf("custom_properties reports are no longer supported")
	}
	var decoded struct {
		PlanReport
		Managed *int `json:"managed"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return fmt.Errorf("decode plan report: %w", err)
	}
	if decoded.Managed == nil {
		return fmt.Errorf("missing plan report managed count")
	}
	report := decoded.PlanReport
	report.Managed = *decoded.Managed
	var content string
	if mode == "status-only" {
		// Validate the complete report before reducing it. Never include identities,
		// diagnostics or policy values supplied by the report in private output.
		_, err = validatePlanReport(report, os.Getenv("SETTINGS_TARGET"))
		content = fmt.Sprintf("\n\n## Settings plan\n\nOutcome: %s\n\nManaged values/resources: %d\n", report.Outcome, report.Managed)
	} else {
		content, err = RenderSummary(report, runMetadataFromEnv())
	}
	if err != nil {
		return err
	}
	file, err := os.OpenFile(summaryPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("open Summary: %w", err)
	}
	_, writeErr := file.WriteString(content)
	closeErr := file.Close()
	if writeErr != nil {
		return fmt.Errorf("write Summary: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close Summary: %w", closeErr)
	}
	if _, failed := summaryOutcome(report.Outcome); failed {
		return fmt.Errorf("plan failed")
	}
	if output != nil {
		if _, err := fmt.Fprintf(output, "plan-outcome=%s\n", report.Outcome); err != nil {
			return fmt.Errorf("write plan outcome: %w", err)
		}
	}
	return nil
}
