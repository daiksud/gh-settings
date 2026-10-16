package cli

import (
	"fmt"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
)

func RenderRepositoryComparison(comparison settings.RepositoryComparison) []byte {
	var output strings.Builder
	if comparison.Managed == 0 {
		output.WriteString("No managed repository settings.\n")
	} else if len(comparison.Changes) == 0 {
		output.WriteString("No differences for managed repository settings.\n")
	} else {
		for _, change := range comparison.Changes {
			fmt.Fprintf(&output, "settings.repository.%s: %#v → %#v\n", change.Field, change.Before, change.After)
		}
	}
	return []byte(output.String())
}
