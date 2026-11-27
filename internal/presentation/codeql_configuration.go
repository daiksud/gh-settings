package presentation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
)

func RenderCodeQLConfiguration(changes []settings.CodeQLConfigurationChange, writeLine func(string)) {
	for _, change := range changes {
		before, after := change.Values()
		switch change.Field() {
		case "languages":
			writeLine(codeQLLine("languages", before, after))
		case "runner":
			oldRunner, newRunner := before.(settings.CodeQLRunner), after.(settings.CodeQLRunner)
			if oldRunner.Type != newRunner.Type {
				writeLine(codeQLLine("scan_settings.runner_type", oldRunner.Type, newRunner.Type))
			}
			if oldRunner.Label != newRunner.Label {
				writeLine(codeQLLine("scan_settings.runner_label", runnerLabel(oldRunner), runnerLabel(newRunner)))
			}
		default:
			writeLine(codeQLLine("scan_settings."+change.Field(), before, after))
		}
	}
}

// Standard runners have no configured label.
func runnerLabel(r settings.CodeQLRunner) *string {
	if r.Type == "labeled_runner" {
		return &r.Label
	}
	return nil
}

func codeQLLine(field string, before, after any) string {
	encode := func(v any) string {
		var out bytes.Buffer
		enc := json.NewEncoder(&out)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(v)
		return strings.TrimSuffix(out.String(), "\n")
	}
	return fmt.Sprintf("%s.%s: %s → %s", settings.CodeQLConfigurationPath, field, encode(before), encode(after))
}
