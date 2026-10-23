package workspace

import (
	"fmt"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
	"go.yaml.in/yaml/v4"
)

func parsePages(node *yaml.Node) (settings.PagesDeclaration, error) {
	fields, err := loadMapping(node, "settings.pages")
	if err != nil {
		return settings.PagesDeclaration{}, err
	}
	d := settings.PagesDeclaration{}
	for key, value := range fields {
		n, err := resolveAlias(&value)
		if err != nil {
			return d, err
		}
		switch key {
		case "build_type":
			if n.Kind != yaml.ScalarNode || n.Tag != "!!str" {
				return d, fmt.Errorf("pages.build_type must be a string")
			}
			v := n.Value
			d.BuildType = &v
		case "https_enforced":
			if n.Kind != yaml.ScalarNode || n.Tag != "!!bool" || !strings.EqualFold(n.Value, "true") && !strings.EqualFold(n.Value, "false") {
				return d, fmt.Errorf("pages.https_enforced must be a boolean spelled true or false")
			}
			var v bool
			if err := n.Decode(&v); err != nil {
				return d, err
			}
			d.HTTPSEnforced = &v
		case "cname":
			if n.Kind != yaml.ScalarNode || n.Tag != "!!str" && n.Tag != "!!null" {
				return d, fmt.Errorf("pages.cname must be a hostname or null")
			}
			d.CNAME = &settings.PagesDomain{}
			if n.Tag == "!!str" {
				v := strings.ToLower(n.Value)
				d.CNAME.Value = &v
			}
		case "source":
			source, err := loadMapping(n, "pages.source")
			if err != nil {
				return d, err
			}
			if len(source) != 2 {
				return d, fmt.Errorf("pages.source requires exactly branch and path")
			}
			s := settings.PagesSource{}
			for k, v := range source {
				n, err := resolveAlias(&v)
				if err != nil {
					return d, err
				}
				if n.Kind != yaml.ScalarNode || n.Tag != "!!str" {
					return d, fmt.Errorf("pages.source values must be strings")
				}
				switch k {
				case "branch":
					s.Branch = n.Value
				case "path":
					s.Path = n.Value
				default:
					return d, fmt.Errorf("unknown pages.source key %q", k)
				}
			}
			d.Source = &s
		default:
			return d, fmt.Errorf("unknown Pages setting %q", key)
		}
	}
	return d, d.Validate()
}
func pagesDocument(d settings.PagesDeclaration) map[string]any {
	m := map[string]any{}
	if d.BuildType != nil {
		m["build_type"] = *d.BuildType
	}
	if d.Source != nil {
		m["source"] = *d.Source
	}
	if d.CNAME != nil {
		m["cname"] = d.CNAME.Value
	}
	if d.HTTPSEnforced != nil {
		m["https_enforced"] = *d.HTTPSEnforced
	}
	return m
}
