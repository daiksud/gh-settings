package workspace

import (
	"fmt"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
	"go.yaml.in/yaml/v4"
)

func parseSecurity(node *yaml.Node) (settings.SecurityDeclaration, error) {
	var d settings.SecurityDeclaration
	fields, err := loadMapping(node, "settings.security")
	if err != nil {
		return d, err
	}
	for key, node := range fields {
		if key != "secret_scanning" && key != "secret_scanning_push_protection" {
			return d, fmt.Errorf("unknown settings.security key %q", key)
		}
		n, err := resolveAlias(&node)
		if err != nil {
			return d, err
		}
		if n.Kind != yaml.ScalarNode || n.Tag != "!!bool" || (!strings.EqualFold(n.Value, "true") && !strings.EqualFold(n.Value, "false")) {
			return d, fmt.Errorf("security.%s must be true or false", key)
		}
		value := strings.EqualFold(n.Value, "true")
		if key == "secret_scanning" {
			d.SecretScanning = &value
		} else {
			d.PushProtection = &value
		}
	}
	return d, d.Validate()
}
