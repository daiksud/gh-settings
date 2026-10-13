package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"go.yaml.in/yaml/v4"
)

func validateSettings(data []byte) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return fmt.Errorf("decode YAML: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return fmt.Errorf("decode YAML: %w", err)
		}
		return fmt.Errorf("expected one YAML document")
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode || document.Content[0].Tag != "!!map" {
		return fmt.Errorf("expected a root mapping")
	}
	if err := checkMappingKeys(&document, make(map[*yaml.Node]bool)); err != nil {
		return err
	}
	root, err := loadMapping(document.Content[0], "root")
	if err != nil {
		return err
	}
	for key, value := range root {
		if key == "schemaVersion" {
			resolved, err := resolveAlias(&value)
			if err != nil {
				return err
			}
			value = *resolved
			if value.Kind != yaml.ScalarNode || value.Tag != "!!int" {
				return fmt.Errorf("schemaVersion must be integer 1")
			}
			var version int
			if err := value.Decode(&version); err != nil || version != 1 {
				return fmt.Errorf("unsupported schemaVersion %q: expected 1", value.Value)
			}
			continue
		}
		if key != "settings" {
			return fmt.Errorf("unknown root key %q", key)
		}
		groups, err := loadMapping(&value, "settings")
		if err != nil {
			return err
		}
		for group, value := range groups {
			if group != "repository" {
				return fmt.Errorf("unknown settings group %q", group)
			}
			fields, err := loadMapping(&value, "settings.repository")
			if err != nil {
				return err
			}
			for field, value := range fields {
				value, err := resolveAlias(&value)
				if err != nil {
					return err
				}
				switch field {
				case "description", "homepage":
					if value.Kind != yaml.ScalarNode || value.Tag != "!!str" {
						return fmt.Errorf("%s must be a string", field)
					}
				case "has_issues", "has_projects", "has_wiki":
					if value.Kind != yaml.ScalarNode || value.Tag != "!!bool" {
						return fmt.Errorf("%s must be a boolean", field)
					}
					if !strings.EqualFold(value.Value, "true") && !strings.EqualFold(value.Value, "false") {
						return fmt.Errorf("%s must be a boolean spelled true or false", field)
					}
					var parsed bool
					if err := value.Decode(&parsed); err != nil {
						return fmt.Errorf("%s must be a boolean: %w", field, err)
					}
				default:
					return fmt.Errorf("unknown setting %q", field)
				}
			}
		}
	}
	if _, ok := root["schemaVersion"]; !ok {
		return fmt.Errorf("missing schemaVersion: expected 1")
	}
	return nil
}

func loadMapping(node *yaml.Node, name string) (map[string]yaml.Node, error) {
	node, err := resolveAlias(node)
	if err != nil {
		return nil, err
	}
	if node.Kind != yaml.MappingNode || node.Tag != "!!map" {
		return nil, fmt.Errorf("%s must be a mapping", name)
	}
	var values map[string]yaml.Node
	if err := node.Load(&values, yaml.WithUniqueKeys()); err != nil {
		return nil, fmt.Errorf("decode %s: %w", name, err)
	}
	return values, nil
}

func resolveAlias(node *yaml.Node) (*yaml.Node, error) {
	seen := make(map[*yaml.Node]bool)
	for node.Kind == yaml.AliasNode {
		if node.Alias == nil || seen[node] {
			return nil, fmt.Errorf("invalid YAML alias")
		}
		seen[node] = true
		node = node.Alias
	}
	return node, nil
}

func checkMappingKeys(node *yaml.Node, visited map[*yaml.Node]bool) error {
	node, err := resolveAlias(node)
	if err != nil {
		return err
	}
	if visited[node] {
		return nil
	}
	visited[node] = true
	if node.Kind == yaml.MappingNode {
		if node.Tag != "!!map" {
			return fmt.Errorf("mapping must have YAML map tag at line %d", node.Line)
		}
		seen := make(map[string]bool)
		for i := 0; i < len(node.Content); i += 2 {
			rawKey := node.Content[i]
			key, err := resolveAlias(rawKey)
			if err != nil {
				return err
			}
			if rawKey.Kind == yaml.AliasNode && key.Tag == "!!merge" {
				return fmt.Errorf("merge key cannot be an alias at line %d", rawKey.Line)
			}
			if key.Kind != yaml.ScalarNode || (key.Tag != "!!str" && key.Tag != "!!merge") {
				return fmt.Errorf("mapping keys must be strings at line %d", key.Line)
			}
			if key.Tag == "!!str" {
				if seen[key.Value] {
					return fmt.Errorf("duplicate key %q at line %d", key.Value, key.Line)
				}
				seen[key.Value] = true
			}
			if err := checkMappingKeys(node.Content[i+1], visited); err != nil {
				return err
			}
		}
	} else if node.Kind == yaml.SequenceNode {
		if node.Tag != "!!seq" {
			return fmt.Errorf("sequence must have YAML sequence tag at line %d", node.Line)
		}
		for _, child := range node.Content {
			if err := checkMappingKeys(child, visited); err != nil {
				return err
			}
		}
	} else if node.Kind == yaml.DocumentNode {
		for _, child := range node.Content {
			if err := checkMappingKeys(child, visited); err != nil {
				return err
			}
		}
	}
	return nil
}
