package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/daiksud/gh-settings/internal/presentation"
	"github.com/daiksud/gh-settings/internal/settings"
)

func RenderComparison(comparison settings.Comparison) []byte {
	var output strings.Builder
	if comparison.Managed == 0 {
		output.WriteString("No managed repository settings.\n")
	} else if !comparison.HasChanges() {
		output.WriteString("No differences for managed repository settings.\n")
	} else {
		for _, change := range comparison.Changes {
			if change.Normalization() != (settings.ObservationNormalization{}) {
				fmt.Fprintln(&output, presentation.CreationPolicyLine(change))
				continue
			}
			line := func(before, after string) {
				fmt.Fprintf(&output, "%s: %s → %s\n", presentation.SettingPath(change.Field().Path()), before, after)
			}
			change.Visit(
				func(before, after bool) { line(strconv.FormatBool(before), strconv.FormatBool(after)) },
				func(before, after string) { line(strconv.Quote(before), strconv.Quote(after)) },
				func(before, after []string) { line(topicsValue(before), topicsValue(after)) },
			)
			if note := presentation.ArchiveOrder(change); note != "" {
				fmt.Fprintln(&output, note)
			}
		}
		for _, change := range comparison.CommitMessages {
			fmt.Fprintln(&output, presentation.CommitMessageLine(change))
		}
		for _, change := range comparison.Labels.Values() {
			fmt.Fprintln(&output, presentation.LabelLine(change))
		}
	}
	for _, change := range comparison.Rulesets {
		fmt.Fprintln(&output, presentation.RulesetLine(change.Display()))
	}
	for _, change := range comparison.Environments {
		fmt.Fprintln(&output, presentation.EnvironmentLine(change))
	}
	for _, change := range comparison.ActionsExecution {
		fmt.Fprintln(&output, presentation.ActionsExecutionLine(change))
	}
	for _, change := range comparison.Actions {
		fmt.Fprintln(&output, presentation.ActionsPolicyLine(change))
	}
	for _, change := range comparison.Security {
		fmt.Fprintln(&output, presentation.SecurityLine(change))
	}
	for _, c := range comparison.Pages {
		fmt.Fprintln(&output, presentation.PagesLine(c))
	}
	return []byte(output.String())
}

func topicsValue(names []string) string {
	data, _ := json.Marshal(names)
	return string(data)
}
