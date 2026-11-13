package actions

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

func oidcReportValue(v any) (settings.OIDCSubjectTemplate, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return settings.OIDCSubjectTemplate{}, err
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(b, &raw) != nil || raw == nil {
		return settings.OIDCSubjectTemplate{}, fmt.Errorf("invalid OIDC template report")
	}
	for k, v := range raw {
		if (k != "use_default_template" && k != "subject_claim_template") || bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return settings.OIDCSubjectTemplate{}, fmt.Errorf("invalid OIDC template report member")
		}
	}
	var d settings.OIDCSubjectTemplate
	if json.Unmarshal(b, &d) != nil {
		return d, fmt.Errorf("invalid OIDC template report value")
	}
	return d, settings.ValidateOIDCSubject(d)
}
func validateOIDCSubjectReport(raw []PlanChange) ([]settings.OIDCSubjectChange, error) {
	if len(raw) > 1 {
		return nil, fmt.Errorf("too many OIDC template changes")
	}
	var result []settings.OIDCSubjectChange
	for _, entry := range raw {
		if entry.Field != settings.OIDCSubjectPath || entry.Normalization != nil || entry.SecurityStatusNormalization != nil {
			return nil, fmt.Errorf("invalid OIDC template report")
		}
		before, err := oidcReportValue(entry.Before)
		if err != nil {
			return nil, err
		}
		after, err := oidcReportValue(entry.After)
		if err != nil {
			return nil, err
		}
		change, err := settings.NewOIDCSubjectChange(before, after)
		if err != nil {
			return nil, err
		}
		result = append(result, change)
	}
	return result, nil
}
