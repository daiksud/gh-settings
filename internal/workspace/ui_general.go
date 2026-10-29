package workspace

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"go.yaml.in/yaml/v4"
)

func decodeUISettings(node *yaml.Node) (uimodel.RepositorySettings, error) {
	fields, err := loadMapping(node, "repository.settings")
	if err != nil {
		return uimodel.RepositorySettings{}, err
	}
	var result uimodel.RepositorySettings
	for key, value := range fields {
		if key == "security_and_quality" {
			if err := decodeUIControl(&value, reflect.ValueOf(&result.SecurityAndQuality).Elem(), "repository.settings.security_and_quality"); err != nil {
				return uimodel.RepositorySettings{}, err
			}
			continue
		}
		if key == "actions" {
			if err := decodeUIControl(&value, reflect.ValueOf(&result.Actions).Elem(), "repository.settings.actions"); err != nil {
				return uimodel.RepositorySettings{}, err
			}
			continue
		}
		if key == "rulesets" {
			if err := decodeUIControl(&value, reflect.ValueOf(&result.Rulesets).Elem(), "repository.settings.rulesets"); err != nil {
				return uimodel.RepositorySettings{}, err
			}
			continue
		}
		if key != "general" {
			return uimodel.RepositorySettings{}, fmt.Errorf("unknown repository.settings key %q", key)
		}
		var general uimodel.General
		if err := decodeUIControl(&value, reflect.ValueOf(&general).Elem(), "repository.settings.general"); err != nil {
			return uimodel.RepositorySettings{}, err
		}
		result.General = &general
	}
	return result, nil
}

func encodeUISettings(settings uimodel.RepositorySettings) (*yaml.Node, error) {
	result := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if settings.General != nil {
		node, err := encodeUIControl(reflect.ValueOf(*settings.General))
		if err != nil {
			return nil, err
		}
		result.Content = append(result.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "general"}, node)
	}
	if settings.Rulesets != nil {
		node, err := encodeUIControl(reflect.ValueOf(settings.Rulesets))
		if err != nil {
			return nil, err
		}
		result.Content = append(result.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "rulesets"}, node)
	}
	if settings.Actions != nil {
		node, err := encodeUIControl(reflect.ValueOf(settings.Actions))
		if err != nil {
			return nil, err
		}
		result.Content = append(result.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "actions"}, node)
	}
	if settings.SecurityAndQuality != nil {
		node, err := encodeUIControl(reflect.ValueOf(settings.SecurityAndQuality))
		if err != nil {
			return nil, err
		}
		result.Content = append(result.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "security_and_quality"}, node)
	}
	return result, nil
}

var uiInitialismBoundary = regexp.MustCompile(`([A-Z]+)([A-Z][a-z])`)
var uiWordBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

// UI models contain reviewed control names, never API transport fields.
// Literal YAML fixtures, rather than this spelling mechanism, fix their contract.
func uiControlName(name string) string {
	name = strings.ReplaceAll(name, "GitHub", "Github")
	name = uiInitialismBoundary.ReplaceAllString(name, "${1}_${2}")
	return strings.ToLower(uiWordBoundary.ReplaceAllString(name, "${1}_${2}"))
}

func decodeUIControl(node *yaml.Node, target reflect.Value, path string) error {
	node, err := resolveAlias(node)
	if err != nil {
		return err
	}
	if node.Tag == "!!null" {
		return fmt.Errorf("%s must not be null", path)
	}
	if target.Kind() == reflect.Pointer {
		value := reflect.New(target.Type().Elem())
		if err := decodeUIControl(node, value.Elem(), path); err != nil {
			return err
		}
		target.Set(value)
		return nil
	}
	switch target.Kind() {
	case reflect.Slice:
		if node.Kind != yaml.SequenceNode || node.Tag != "!!seq" {
			return fmt.Errorf("%s must be an array", path)
		}
		values := reflect.MakeSlice(target.Type(), len(node.Content), len(node.Content))
		for i, item := range node.Content {
			if err := decodeUIControl(item, values.Index(i), fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
		target.Set(values)
	case reflect.Int, reflect.Int64:
		if node.Kind != yaml.ScalarNode || node.Tag != "!!int" {
			return fmt.Errorf("%s must be an integer", path)
		}
		value := reflect.New(target.Type())
		if err := node.Decode(value.Interface()); err != nil {
			return fmt.Errorf("%s must be a supported integer: %w", path, err)
		}
		target.Set(value.Elem())
	case reflect.Struct:
		fields, err := loadMapping(node, path)
		if err != nil {
			return err
		}
		for key, value := range fields {
			index := -1
			for i := 0; i < target.NumField(); i++ {
				if uiControlName(target.Type().Field(i).Name) == key {
					index = i
					break
				}
			}
			if index < 0 {
				return fmt.Errorf("unknown %s key %q", path, key)
			}
			if err := decodeUIControl(&value, target.Field(index), path+"."+key); err != nil {
				return err
			}
		}
	case reflect.String:
		if node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
			return fmt.Errorf("%s must be a string", path)
		}
		target.SetString(node.Value)
	case reflect.Bool:
		if node.Kind != yaml.ScalarNode || node.Tag != "!!bool" || (!strings.EqualFold(node.Value, "true") && !strings.EqualFold(node.Value, "false")) {
			return fmt.Errorf("%s must be a boolean spelled true or false", path)
		}
		target.SetBool(strings.EqualFold(node.Value, "true"))
	default:
		return fmt.Errorf("unsupported UI control type at %s", path)
	}
	return nil
}

func encodeUIControl(value reflect.Value) (*yaml.Node, error) {
	if value.Kind() == reflect.Pointer {
		return encodeUIControl(value.Elem())
	}
	if value.Kind() == reflect.Slice {
		result := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for i := 0; i < value.Len(); i++ {
			child, err := encodeUIControl(value.Index(i))
			if err != nil {
				return nil, err
			}
			result.Content = append(result.Content, child)
		}
		return result, nil
	}
	if value.Kind() == reflect.Struct {
		result := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i)
			if field.IsZero() {
				continue
			}
			child, err := encodeUIControl(field)
			if err != nil {
				return nil, err
			}
			result.Content = append(result.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: uiControlName(value.Type().Field(i).Name)}, child)
		}
		return result, nil
	}
	if value.Kind() != reflect.Bool && value.Kind() != reflect.String && value.Kind() != reflect.Int64 && value.Kind() != reflect.Int {
		return nil, fmt.Errorf("unsupported UI control type %s", value.Type())
	}
	var result yaml.Node
	if err := result.Encode(value.Interface()); err != nil {
		return nil, err
	}
	return &result, nil
}
