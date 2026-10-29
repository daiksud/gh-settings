package presentation

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func ActionsPolicyLine(change settings.ActionsPolicyChange) string {
	before, after := change.Values()
	if normalization := change.Normalization(); normalization.Path != "" {
		choice := uimodel.ReadRepositoryContentsAndPackagesPermissions
		if after == "write" {
			choice = uimodel.ReadAndWritePermissions
		}
		return NormalizationLine(normalization) + "\n" + fmt.Sprintf("%s: %s (normalized observation) → %s (write required)", normalization.Path, strconv.Quote(normalization.Fallback), strconv.Quote(string(choice)))
	}
	b, _ := json.Marshal(before)
	a, _ := json.Marshal(after)
	value := string(b)
	if before == nil {
		value = "(unavailable while policy is not selected)"
	}
	return fmt.Sprintf("settings.%s: %s → %s", change.Field(), value, a)
}
