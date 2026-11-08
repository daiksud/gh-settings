package actions

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

func codeQLReportValue(field string, v any) (any, error) {
	if field == "languages" {
		if s, ok := v.([]string); ok {
			return s, nil
		}
		a, ok := v.([]any)
		if !ok {
			return nil, fmt.Errorf("invalid CodeQL languages report")
		}
		s := []string{}
		for _, x := range a {
			v, ok := x.(string)
			if !ok {
				return nil, fmt.Errorf("invalid CodeQL language report")
			}
			s = append(s, v)
		}
		return s, nil
	}
	if field == "runner" {
		if r, ok := v.(settings.CodeQLRunner); ok {
			return r, nil
		}
		m, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid CodeQL runner report")
		}
		for k := range m {
			if k != "type" && k != "label" {
				return nil, fmt.Errorf("unknown CodeQL runner member")
			}
		}
		t, ok := m["type"].(string)
		if !ok {
			return nil, fmt.Errorf("missing CodeQL runner type")
		}
		r := settings.CodeQLRunner{Type: t}
		if l, present := m["label"]; present {
			var ok bool
			r.Label, ok = l.(string)
			if !ok {
				return nil, fmt.Errorf("invalid CodeQL runner label")
			}
		}
		return r, nil
	}
	return v, nil
}
func validateCodeQLConfigurationReport(raw []PlanChange) ([]settings.CodeQLConfigurationChange, error) {
	result := []settings.CodeQLConfigurationChange{}
	seen := map[string]bool{}
	for _, r := range raw {
		if r.Normalization != nil || r.SecurityStatusNormalization != nil || seen[r.Field] {
			return nil, fmt.Errorf("invalid or duplicate CodeQL configuration report")
		}
		seen[r.Field] = true
		b, err := codeQLReportValue(r.Field, r.Before)
		if err != nil {
			return nil, err
		}
		a, err := codeQLReportValue(r.Field, r.After)
		if err != nil {
			return nil, err
		}
		c, err := settings.NewCodeQLConfigurationChange(r.Field, b, a)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}
