package workspace

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"go.yaml.in/yaml/v4"
	"strconv"
)

type labelDocument struct {
	ID          *int64  `yaml:"id,omitempty"`
	Name        string  `yaml:"name"`
	Color       *string `yaml:"color,omitempty"`
	Description *string `yaml:"description,omitempty"`
}

func parseLabels(node *yaml.Node) (*settings.LabelDeclarations, error) {
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
			if key != "id" && key != "name" && key != "color" && key != "description" {
				return nil, fmt.Errorf("unknown label setting %q", key)
			}
			value, err := resolveAlias(&value)
			if err != nil {
				return nil, err
			}
			if key == "id" {
				if value.Kind != yaml.ScalarNode || value.Tag != "!!int" {
					return nil, fmt.Errorf("label id must be a positive integer")
				}
				id, err := strconv.ParseInt(value.Value, 10, 64)
				if err != nil || id <= 0 {
					return nil, fmt.Errorf("label id must be a positive integer")
				}
				label.ID = &id
				continue
			}
			if value.Kind != yaml.ScalarNode || value.Tag != "!!str" {
				return nil, fmt.Errorf("label %s must be a string", key)
			}
			text := value.Value
			switch key {
			case "name":
				label.Name = text
			case "color":
				label.Color = &text
			case "description":
				label.Description = &text
			}
		}
		labels = append(labels, label)
	}
	declarations, err := settings.NewLabelDeclarations(labels)
	if err != nil {
		return nil, err
	}
	return &declarations, nil
}
