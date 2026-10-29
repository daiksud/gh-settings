package workspace

import (
	"fmt"
	"reflect"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"go.yaml.in/yaml/v4"
)

const configVersion = "daiksud/gh-settings@v0"

func DecodeUIConfig(data []byte) (uimodel.Repository, error) {
	node, err := loadYAMLRoot(data)
	if err != nil {
		return uimodel.Repository{}, err
	}
	if err := checkMappingKeys(node, make(map[*yaml.Node]bool)); err != nil {
		return uimodel.Repository{}, err
	}
	root, err := loadMapping(node, "root")
	if err != nil {
		return uimodel.Repository{}, err
	}
	version, ok := root["version"]
	if !ok {
		return uimodel.Repository{}, fmt.Errorf("missing version: expected %s", configVersion)
	}
	resolved, err := resolveAlias(&version)
	if err != nil {
		return uimodel.Repository{}, err
	}
	if resolved.Kind != yaml.ScalarNode || resolved.Tag != "!!str" || resolved.Value != configVersion {
		return uimodel.Repository{}, fmt.Errorf("version must be the string %q", configVersion)
	}
	var repository uimodel.Repository
	for key, value := range root {
		switch key {
		case "version":
		case "repository":
			fields, err := loadMapping(&value, "repository")
			if err != nil {
				return uimodel.Repository{}, err
			}
			for name, value := range fields {
				if name == "labels" {
					if err := decodeUIControl(&value, reflect.ValueOf(&repository.Labels).Elem(), "repository.labels"); err != nil {
						return uimodel.Repository{}, err
					}
					continue
				}
				if name == "settings" {
					controls, err := decodeUISettings(&value)
					if err != nil {
						return uimodel.Repository{}, err
					}
					repository.Settings = &controls
					continue
				}
				if name != "about" {
					return uimodel.Repository{}, fmt.Errorf("unknown repository key %q", name)
				}
				about, err := decodeUIAbout(&value)
				if err != nil {
					return uimodel.Repository{}, err
				}
				repository.About = &about
			}
		default:
			return uimodel.Repository{}, fmt.Errorf("unknown root key %q", key)
		}
	}
	return repository, nil
}

func EncodeUIConfig(repository uimodel.Repository) ([]byte, error) {
	var document struct {
		Version    string `yaml:"version"`
		Repository struct {
			About    *uiAboutDocument `yaml:"about,omitempty"`
			Labels   *yaml.Node       `yaml:"labels,omitempty"`
			Settings *yaml.Node       `yaml:"settings,omitempty"`
		} `yaml:"repository"`
	}
	document.Version = configVersion
	if repository.About != nil {
		document.Repository.About = &uiAboutDocument{
			Description: repository.About.Description,
			Website:     repository.About.Website,
			Topics:      repository.About.Topics,
		}
	}
	if repository.Labels != nil {
		labels, err := encodeUIControl(reflect.ValueOf(repository.Labels))
		if err != nil {
			return nil, err
		}
		document.Repository.Labels = labels
	}
	if repository.Settings != nil {
		settings, err := encodeUISettings(*repository.Settings)
		if err != nil {
			return nil, err
		}
		document.Repository.Settings = settings
	}
	return yaml.Marshal(document)
}

type uiAboutDocument struct {
	Description *string   `yaml:"description,omitempty"`
	Website     *string   `yaml:"website,omitempty"`
	Topics      *[]string `yaml:"topics,omitempty"`
}

func decodeUIAbout(node *yaml.Node) (uimodel.About, error) {
	fields, err := loadMapping(node, "repository.about")
	if err != nil {
		return uimodel.About{}, err
	}
	var about uimodel.About
	for key, value := range fields {
		node, err := resolveAlias(&value)
		if err != nil {
			return uimodel.About{}, err
		}
		switch key {
		case "description", "website":
			if node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
				return uimodel.About{}, fmt.Errorf("repository.about.%s must be a string", key)
			}
			text := node.Value
			if key == "description" {
				about.Description = &text
			} else {
				about.Website = &text
			}
		case "topics":
			if node.Kind != yaml.SequenceNode || node.Tag != "!!seq" {
				return uimodel.About{}, fmt.Errorf("repository.about.topics must be a string array")
			}
			values := make([]string, 0, len(node.Content))
			for _, item := range node.Content {
				item, err := resolveAlias(item)
				if err != nil || item.Kind != yaml.ScalarNode || item.Tag != "!!str" {
					return uimodel.About{}, fmt.Errorf("repository.about.topics must contain strings")
				}
				values = append(values, item.Value)
			}
			about.Topics = &values
		default:
			return uimodel.About{}, fmt.Errorf("unknown repository.about key %q", key)
		}
	}
	return about, nil
}
