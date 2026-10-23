package workspace

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
	"go.yaml.in/yaml/v4"
)

func parseProperties(node *yaml.Node) (settings.PropertyDeclarations, error) {
	fields, err := loadMapping(node, "settings.custom_properties")
	if err != nil {
		return nil, err
	}
	d := settings.PropertyDeclarations{}
	for name, value := range fields {
		n, err := resolveAlias(&value)
		if err != nil {
			return nil, err
		}
		var v any
		switch {
		case n.Kind == yaml.ScalarNode && n.Tag == "!!null":
			v = nil
		case n.Kind == yaml.ScalarNode && n.Tag == "!!str":
			v = n.Value
		case n.Kind == yaml.SequenceNode && n.Tag == "!!seq":
			a := []string{}
			for _, item := range n.Content {
				x, err := resolveAlias(item)
				if err != nil {
					return nil, err
				}
				if x.Kind != yaml.ScalarNode || x.Tag != "!!str" {
					return nil, fmt.Errorf("custom property arrays require strings")
				}
				a = append(a, x.Value)
			}
			v = a
		default:
			return nil, fmt.Errorf("custom property values require string, string array or null")
		}
		d[name], err = settings.PropertyValue(v)
		if err != nil {
			return nil, err
		}
	}
	return d, d.Validate()
}
