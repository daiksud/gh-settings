package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
)

func RenderComparison(comparison settings.Comparison) []byte {
	var output strings.Builder
	if comparison.Managed == 0 {
		output.WriteString("No managed repository settings.\n")
	} else if len(comparison.Changes) == 0 && len(comparison.Labels) == 0 {
		output.WriteString("No differences for managed repository settings.\n")
	} else {
		for _, change := range comparison.Changes {
			fmt.Fprintf(&output, "settings.%s: %s → %s\n", change.Field().Path(), comparisonValue(change.Before), comparisonValue(change.After))
		}
		for _, change := range comparison.Labels {
			fmt.Fprintln(&output, labelLine(change))
		}
	}
	return []byte(output.String())
}

func comparisonValue(value any) string {
	if names, ok := value.([]string); ok {
		data, _ := json.Marshal(names)
		return string(data)
	}
	return fmt.Sprintf("%#v", value)
}
