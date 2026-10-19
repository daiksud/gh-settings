package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
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
			line := func(before, after string) {
				fmt.Fprintf(&output, "settings.%s: %s → %s\n", change.Field().Path(), before, after)
			}
			change.Visit(
				func(before, after bool) { line(strconv.FormatBool(before), strconv.FormatBool(after)) },
				func(before, after string) { line(strconv.Quote(before), strconv.Quote(after)) },
				func(before, after []string) { line(topicsValue(before), topicsValue(after)) },
			)
		}
		for _, change := range comparison.Labels {
			fmt.Fprintln(&output, labelLine(change))
		}
	}
	return []byte(output.String())
}

func topicsValue(names []string) string {
	data, _ := json.Marshal(names)
	return string(data)
}
