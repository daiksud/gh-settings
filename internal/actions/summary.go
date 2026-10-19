package actions

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
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
		content.WriteString(quoteHTML(report.diagnostic))
	}
	for _, change := range report.comparison.Changes {
		line := func(before, after string) {
			fmt.Fprintf(&content, "settings.%s: %s → %s\n", quoteHTML(change.Field().Path()), quoteHTML(before), quoteHTML(after))
		}
		change.Visit(
			func(before, after bool) { line(strconv.FormatBool(before), strconv.FormatBool(after)) },
			func(before, after string) { line(strconv.Quote(before), strconv.Quote(after)) },
			func(before, after []string) {
				beforeJSON, _ := json.Marshal(before)
				afterJSON, _ := json.Marshal(after)
				line(string(beforeJSON), string(afterJSON))
			},
		)
	}
	for _, change := range report.comparison.Labels.Values() {
		fmt.Fprintf(&content, "%s\n", quoteHTML(labelSummaryLine(change)))
	}
	content.WriteString("</pre>\n\nDirect apply uses the declaration and can overwrite intervening remote changes to managed fields. Review this Summary before authorizing a separate direct apply; successful rendering alone does not prove publication.\n")
	return content.String()
}

// validatedPlanReport retains the result of the JSON and report-state boundary.
type validatedPlanReport struct {
	outcome, diagnostic string
	comparison          settings.Comparison
}

func validatePlanReport(report PlanReport, target string) (validatedPlanReport, error) {
	maxManaged := settings.RepositoryFieldCount() + 2 // Topics and labels each count once.
	if report.Changes == nil || report.Managed < len(report.Changes) || report.Managed > maxManaged {
		return validatedPlanReport{}, fmt.Errorf("invalid plan report comparison")
	}
	if len(report.Labels) > 0 && report.Managed < len(report.Changes)+1 {
		return validatedPlanReport{}, fmt.Errorf("invalid label report managed count")
	}
	labels, err := validateLabelReport(report.Labels)
	if err != nil {
		return validatedPlanReport{}, err
	}
	hasChanges := len(report.Changes) > 0 || labels.Len() > 0
	valid := false
	switch report.Outcome {
	case "changed":
		valid = report.Managed > 0 && hasChanges && report.Diagnostic == "" && report.Target == target
	case "no-difference":
		valid = report.Managed > 0 && !hasChanges && report.Diagnostic == "" && report.Target == target
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
		return validatedPlanReport{}, fmt.Errorf("invalid plan report state")
	}
	comparison := settings.Comparison{Managed: report.Managed, Labels: labels}
	seen := make(map[settings.FieldID]bool, len(report.Changes))
	for _, raw := range report.Changes {
		change, err := parseReportChange(raw)
		if err != nil {
			return validatedPlanReport{}, err
		}
		if seen[change.Field()] {
			return validatedPlanReport{}, fmt.Errorf("invalid plan report change")
		}
		seen[change.Field()] = true
		comparison.Changes = append(comparison.Changes, change)
	}
	return validatedPlanReport{outcome: report.Outcome, diagnostic: report.Diagnostic, comparison: comparison}, nil
}

func parseReportChange(raw PlanChange) (settings.Change, error) {
	field, err := settings.ParseFieldID(raw.Field)
	if err != nil {
		return settings.Change{}, fmt.Errorf("invalid plan report change")
	}
	var change settings.Change
	if field == settings.TopicsField() {
		before, beforeOK := summaryTopics(raw.Before)
		after, afterOK := summaryTopics(raw.After)
		if !beforeOK || !afterOK || reflect.DeepEqual(before, after) {
			return settings.Change{}, fmt.Errorf("invalid plan report change")
		}
		change, err = settings.NewTopicsChange(before, after)
	} else {
		switch before := raw.Before.(type) {
		case bool:
			after, ok := raw.After.(bool)
			if !ok || before == after {
				return settings.Change{}, fmt.Errorf("invalid plan report change")
			}
			change, err = settings.NewBoolChange(field, before, after)
		case string:
			after, ok := raw.After.(string)
			if !ok || before == after {
				return settings.Change{}, fmt.Errorf("invalid plan report change")
			}
			change, err = settings.NewStringChange(field, before, after)
		default:
			return settings.Change{}, fmt.Errorf("invalid plan report change")
		}
	}
	if err != nil {
		return settings.Change{}, fmt.Errorf("invalid plan report change")
	}
	return change, nil
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

func summaryTopics(value any) ([]string, bool) {
	var names []string
	switch value := value.(type) {
	case []string:
		names = value
	case []any:
		names = make([]string, 0, len(value))
		for _, item := range value {
			name, ok := item.(string)
			if !ok {
				return nil, false
			}
			names = append(names, name)
		}
	default:
		return nil, false
	}
	if names == nil || settings.ValidateTopics(names) != nil {
		return nil, false
	}
	sorted := settings.SortedTopics(names)
	return sorted, reflect.DeepEqual(sorted, names)
}
