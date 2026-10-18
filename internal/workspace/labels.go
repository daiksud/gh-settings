package workspace

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"go.yaml.in/yaml/v4"
	"strings"
)

type labelDocument struct {
	Name        string  `yaml:"name"`
	Color       *string `yaml:"color,omitempty"`
	Description *string `yaml:"description,omitempty"`
}

func parseLabels(node *yaml.Node) (*[]settings.LabelDeclaration, error) {
	node, err := resolveAlias(node)
	if err != nil {
		return nil, err
	}
	if node.Kind != yaml.SequenceNode || node.Tag != "!!seq" {
		return nil, fmt.Errorf("settings.labels must be an array of label mappings")
	}
	labels := make([]settings.LabelDeclaration, 0, len(node.Content))
	for _, item := range node.Content {
		fields, err := loadMapping(item, "label")
		if err != nil {
			return nil, err
		}
		var label settings.LabelDeclaration
		for key, value := range fields {
			if key != "name" && key != "color" && key != "description" {
				return nil, fmt.Errorf("unknown label setting %q", key)
			}
			value, err := resolveAlias(&value)
			if err != nil {
				return nil, err
			}
			if value.Kind != yaml.ScalarNode || value.Tag != "!!str" {
				return nil, fmt.Errorf("label %s must be a string", key)
			}
			text := value.Value
			switch key {
			case "name":
				label.Name = text
			case "color":
				text = strings.ToLower(text)
				label.Color = &text
			case "description":
				label.Description = &text
			}
		}
		labels = append(labels, label)
	}
	if err := settings.ValidateLabels(labels); err != nil {
		return nil, err
	}
	return &labels, nil
}
