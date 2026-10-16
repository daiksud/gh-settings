package actions

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"os"
	"strconv"
	"strings"
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
	if err := validatePlanReport(report, metadata.Target); err != nil {
		return "", err
	}
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
	result, failed := summaryOutcome(report.Outcome)
	fmt.Fprintf(&content, "</table>\n\n<p>%s</p>\n\n<pre>", result)
	if failed {
		content.WriteString(quoteHTML(report.Diagnostic))
	}
	for _, change := range report.Changes {
		fmt.Fprintf(&content, "settings.repository.%s: %s → %s\n", quoteHTML(change.Field), quoteHTML(summaryValue(change.Before)), quoteHTML(summaryValue(change.After)))
	}
	content.WriteString("</pre>\n\nDirect apply uses the declaration and can overwrite intervening remote changes to managed fields. Review this Summary before authorizing a separate direct apply; successful rendering alone does not prove publication.\n")
	return content.String(), nil
}

func validatePlanReport(report PlanReport, target string) error {
	if report.Changes == nil || report.Managed < len(report.Changes) {
		return fmt.Errorf("invalid plan report comparison")
	}
	valid := false
	switch report.Outcome {
	case "changed":
		valid = report.Managed > 0 && len(report.Changes) > 0 && report.Diagnostic == "" && report.Target == target
	case "no-difference":
		valid = report.Managed > 0 && len(report.Changes) == 0 && report.Diagnostic == "" && report.Target == target
	case "unmanaged":
		valid = report.Managed == 0 && report.Target == "" && report.Diagnostic == ""
	case "invalid-config":
		valid = report.Managed == 0 && report.Target == "" && report.Diagnostic == invalidConfigDiagnostic
	case "target-failure":
		valid = report.Managed == 0 && report.Target == "" && report.Diagnostic == targetFailureDiagnostic
	case "read-failure":
		valid = report.Managed == 0 && report.Target == target && report.Diagnostic == readFailureDiagnostic
	}
	if !valid {
		return fmt.Errorf("invalid plan report state")
	}
	for _, change := range report.Changes {
		nativePair := false
		switch change.Field {
		case "description", "homepage":
			_, before := change.Before.(string)
			_, after := change.After.(string)
			nativePair = before && after
		case "has_issues", "has_projects", "has_wiki":
			_, before := change.Before.(bool)
			_, after := change.After.(bool)
			nativePair = before && after
		}
		if !nativePair {
			return fmt.Errorf("invalid plan report change")
		}
	}
	return nil
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

func summaryValue(value any) string {
	if text, ok := value.(string); ok {
		return strconv.Quote(text)
	}
	return fmt.Sprint(value)
}

func writeSummary(reportPath, summaryPath string) error {
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return fmt.Errorf("read plan report: %w", err)
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
	content, err := RenderSummary(report, runMetadataFromEnv())
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
	return nil
}
