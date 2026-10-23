package workspace

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
	"go.yaml.in/yaml/v4"
)

type topicsDocument struct {
	Names *[]string `yaml:"names,omitempty"`
}

// EncodeConfig writes the supported schemaVersion: 1 settings resource envelope.
func EncodeConfig(declaration settings.Declaration) ([]byte, error) {
	var document struct {
		SchemaVersion int `yaml:"schemaVersion"`
		Settings      struct {
			Repository yaml.Node                          `yaml:"repository"`
			Topics     *topicsDocument                    `yaml:"topics,omitempty"`
			Labels     *[]labelDocument                   `yaml:"labels,omitempty"`
			Rulesets   *[]settings.RulesetDeclaration     `yaml:"rulesets,omitempty"`
			Actions    *settings.ActionsPolicyDeclaration `yaml:"actions,omitempty"`
			Security   *settings.SecurityDeclaration      `yaml:"security,omitempty"`
			Pages      map[string]any                     `yaml:"pages,omitempty"`
			Properties settings.PropertyDeclarations      `yaml:"custom_properties,omitempty"`
		} `yaml:"settings"`
	}
	document.SchemaVersion = 1
	document.Settings.Repository = yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if err := settings.VisitDeclaredRepositoryFields(declaration.Repository,
		func(name, value string) error {
			return appendRepositoryYAMLValue(&document.Settings.Repository, name, value)
		},
		func(name string, value bool) error {
			return appendRepositoryYAMLValue(&document.Settings.Repository, name, value)
		},
	); err != nil {
		return nil, err
	}
	if declaration.Topics.Names != nil {
		names := settings.SortedTopics(*declaration.Topics.Names)
		document.Settings.Topics = &topicsDocument{Names: &names}
	}
	if declaration.Labels != nil {
		labels := make([]labelDocument, 0, declaration.Labels.Len())
		for _, label := range settings.SortedLabels(declaration.Labels.Values()) {
			labels = append(labels, labelDocument{Name: label.Name, Color: label.Color, Description: label.Description})
		}
		document.Settings.Labels = &labels
	}
	if declaration.Rulesets != nil {
		values := declaration.Rulesets.Values()
		document.Settings.Rulesets = &values
	}
	if !declaration.Actions.Empty() {
		if err := declaration.Actions.Validate(); err != nil {
			return nil, err
		}
		document.Settings.Actions = &declaration.Actions
	}
	if !declaration.Security.Empty() {
		if err := declaration.Security.Validate(); err != nil {
			return nil, err
		}
		document.Settings.Security = &declaration.Security
	}
	if !declaration.Pages.Empty() {
		if err := declaration.Pages.Validate(); err != nil {
			return nil, err
		}
		document.Settings.Pages = pagesDocument(declaration.Pages)
	}
	if err := declaration.Properties.Validate(); err != nil {
		return nil, err
	}
	document.Settings.Properties = declaration.Properties
	return yaml.Marshal(document)
}

func appendRepositoryYAMLValue[V bool | string](repository *yaml.Node, name string, value V) error {
	var node yaml.Node
	if err := node.Encode(value); err != nil {
		return err
	}
	key := yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: strings.TrimPrefix(name, "repository.")}
	repository.Content = append(repository.Content, &key, &node)
	return nil
}

// DecodeConfig validates the YAML envelope and returns its supported resource declarations.
func DecodeConfig(data []byte) (settings.Declaration, error) {
	rootNode, err := loadYAMLRoot(data)
	if err != nil {
		return settings.Declaration{}, err
	}
	if err := checkMappingKeys(rootNode, make(map[*yaml.Node]bool)); err != nil {
		return settings.Declaration{}, err
	}
	root, err := loadMapping(rootNode, "root")
	if err != nil {
		return settings.Declaration{}, err
	}
	var declaration settings.Declaration
	for key, value := range root {
		if key == "schemaVersion" {
			resolved, err := resolveAlias(&value)
			if err != nil {
				return settings.Declaration{}, err
			}
			value = *resolved
			if value.Kind != yaml.ScalarNode || value.Tag != "!!int" {
				return settings.Declaration{}, fmt.Errorf("schemaVersion must be integer 1")
			}
			var version int
			if err := value.Decode(&version); err != nil || version != 1 {
				return settings.Declaration{}, fmt.Errorf("unsupported schemaVersion %q: expected 1", value.Value)
			}
			continue
		}
		if key != "settings" {
			return settings.Declaration{}, fmt.Errorf("unknown root key %q", key)
		}
		groups, err := loadMapping(&value, "settings")
		if err != nil {
			return settings.Declaration{}, err
		}
		for group, value := range groups {
			if group == "custom_properties" {
				d, err := parseProperties(&value)
				if err != nil {
					return settings.Declaration{}, err
				}
				declaration.Properties = d
				continue
			}
			if group == "pages" {
				d, err := parsePages(&value)
				if err != nil {
					return settings.Declaration{}, err
				}
				declaration.Pages = d
				continue
			}
			if group == "security" {
				security, err := parseSecurity(&value)
				if err != nil {
					return settings.Declaration{}, err
				}
				declaration.Security = security
				continue
			}
			if group == "actions" {
				policy, err := parseActionsPolicy(&value)
				if err != nil {
					return settings.Declaration{}, err
				}
				declaration.Actions = policy
				continue
			}
			if group == "rulesets" {
				rulesets, err := parseRulesets(&value)
				if err != nil {
					return settings.Declaration{}, err
				}
				declaration.Rulesets = rulesets
				continue
			}
			if group == "labels" {
				labels, err := parseLabels(&value)
				if err != nil {
					return settings.Declaration{}, err
				}
				declaration.Labels = labels
				continue
			}
			if group == "topics" {
				topics, err := parseTopicsFields(&value)
				if err != nil {
					return settings.Declaration{}, err
				}
				declaration.Topics = topics
				continue
			}
			if group != "repository" {
				return settings.Declaration{}, fmt.Errorf("unknown settings group %q", group)
			}
			repository, err := parseRepositoryFields(&value)
			if err != nil {
				return settings.Declaration{}, err
			}
			declaration.Repository = repository
		}
	}
	if _, ok := root["schemaVersion"]; !ok {
		return settings.Declaration{}, fmt.Errorf("missing schemaVersion: expected 1")
	}
	return declaration, nil
}

func parseRepositoryFields(node *yaml.Node) (settings.RepositoryDeclaration, error) {
	fields, err := loadMapping(node, "settings.repository")
	if err != nil {
		return settings.RepositoryDeclaration{}, err
	}
	var repository settings.RepositoryDeclaration
	for field, value := range fields {
		value, err := resolveAlias(&value)
		if err != nil {
			return settings.RepositoryDeclaration{}, err
		}
		if set, ok := settings.RepositoryBoolSetter("repository." + field); ok {
			if value.Kind != yaml.ScalarNode || value.Tag != "!!bool" {
				return settings.RepositoryDeclaration{}, fmt.Errorf("%s must be a boolean", field)
			}
			if !strings.EqualFold(value.Value, "true") && !strings.EqualFold(value.Value, "false") {
				return settings.RepositoryDeclaration{}, fmt.Errorf("%s must be a boolean spelled true or false", field)
			}
			var parsed bool
			if err := value.Decode(&parsed); err != nil {
				return settings.RepositoryDeclaration{}, fmt.Errorf("%s must be a boolean: %w", field, err)
			}
			set(&repository, parsed)
			continue
		}
		if set, enum, ok := settings.RepositoryStringSetter("repository." + field); ok {
			if value.Kind != yaml.ScalarNode || value.Tag != "!!str" {
				if enum {
					return settings.RepositoryDeclaration{}, fmt.Errorf("%s must be a supported string enum", field)
				}
				return settings.RepositoryDeclaration{}, fmt.Errorf("%s must be a string", field)
			}
			if err := set(&repository, value.Value); err != nil {
				return settings.RepositoryDeclaration{}, err
			}
			continue
		}
		return settings.RepositoryDeclaration{}, fmt.Errorf("unknown setting %q", field)
	}
	if err := repository.ValidateCommitDependencies(); err != nil {
		return settings.RepositoryDeclaration{}, err
	}
	return repository, nil
}

func loadYAMLRoot(data []byte) (*yaml.Node, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode YAML: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return nil, fmt.Errorf("decode YAML: %w", err)
		}
		return nil, fmt.Errorf("expected one YAML document")
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode || document.Content[0].Tag != "!!map" {
		return nil, fmt.Errorf("expected a root mapping")
	}
	return document.Content[0], nil
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

func parseTopicsFields(node *yaml.Node) (settings.TopicsDeclaration, error) {
	fields, err := loadMapping(node, "settings.topics")
	if err != nil {
		return settings.TopicsDeclaration{}, err
	}
	var result settings.TopicsDeclaration
	for key, value := range fields {
		if key != "names" {
			return settings.TopicsDeclaration{}, fmt.Errorf("unknown topics setting %q", key)
		}
		node, err := resolveAlias(&value)
		if err != nil {
			return settings.TopicsDeclaration{}, err
		}
		if node.Kind != yaml.SequenceNode || node.Tag != "!!seq" {
			return settings.TopicsDeclaration{}, fmt.Errorf("topics.names must be an array of strings")
		}
		names := make([]string, 0, len(node.Content))
		for _, item := range node.Content {
			item, err := resolveAlias(item)
			if err != nil {
				return settings.TopicsDeclaration{}, err
			}
			if item.Kind != yaml.ScalarNode || item.Tag != "!!str" {
				return settings.TopicsDeclaration{}, fmt.Errorf("topics.names must be an array of strings")
			}
			names = append(names, item.Value)
		}
		if err := settings.ValidateTopics(names); err != nil {
			return settings.TopicsDeclaration{}, err
		}
		result.Names = &names
	}
	return result, nil
}
