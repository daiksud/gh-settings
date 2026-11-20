package presentation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"strings"
)

func CodeQLConfigurationLine(c settings.CodeQLConfigurationChange) string {
	b, a := c.Values()
	encode := func(v any) string {
		var out bytes.Buffer
		enc := json.NewEncoder(&out)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(v)
		return strings.TrimSuffix(out.String(), "\n")
	}
	before, after := encode(b), encode(a)
	field := c.Field()
	if field == "query_suite" {
		field = "query_suites"
	}
	return fmt.Sprintf("%s.%s: %s → %s", settings.CodeQLConfigurationPath, field, before, after)
}
