package settings

import (
	"fmt"
	"slices"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

const OIDCSubjectPath = "repository.settings.actions.oidc.subject_claim"

type OIDCSubjectTemplate = uimodel.OIDCSubjectTemplate

func ValidateOIDCSubject(d OIDCSubjectTemplate) error {
	if d.UseDefaultTemplate == nil {
		return fmt.Errorf("%s requires use_default_template", OIDCSubjectPath)
	}
	if *d.UseDefaultTemplate {
		if d.SubjectClaimTemplate != nil {
			return fmt.Errorf("default OIDC template forbids custom keys")
		}
		return nil
	}
	if d.SubjectClaimTemplate == nil {
		return fmt.Errorf("custom OIDC template requires subject_claim_template")
	}
	seen := map[string]bool{}
	for _, key := range *d.SubjectClaimTemplate {
		if key == "" || seen[key] {
			return fmt.Errorf("OIDC claim keys must be nonempty and unique")
		}
		for _, c := range key {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_') {
				return fmt.Errorf("OIDC claim keys require ASCII alphanumeric characters or underscores")
			}
		}
		seen[key] = true
	}
	return nil
}
func CloneOIDCSubject(v OIDCSubjectTemplate) OIDCSubjectTemplate {
	if v.UseDefaultTemplate != nil {
		x := *v.UseDefaultTemplate
		v.UseDefaultTemplate = &x
	}
	if v.SubjectClaimTemplate != nil {
		x := append([]string{}, (*v.SubjectClaimTemplate)...)
		v.SubjectClaimTemplate = &x
	}
	return v
}
func OIDCSubjectMatches(a, b OIDCSubjectTemplate) bool {
	if ValidateOIDCSubject(a) != nil || ValidateOIDCSubject(b) != nil || *a.UseDefaultTemplate != *b.UseDefaultTemplate {
		return false
	}
	return *a.UseDefaultTemplate || slices.Equal(*a.SubjectClaimTemplate, *b.SubjectClaimTemplate)
}

type OIDCSubjectObservation struct {
	acquired bool
	value    OIDCSubjectTemplate
}

func NewOIDCSubjectObservation(v OIDCSubjectTemplate) (OIDCSubjectObservation, error) {
	if err := ValidateOIDCSubject(v); err != nil {
		return OIDCSubjectObservation{}, err
	}
	return OIDCSubjectObservation{true, CloneOIDCSubject(v)}, nil
}
func (o OIDCSubjectObservation) Value() (OIDCSubjectTemplate, error) {
	if !o.acquired {
		return OIDCSubjectTemplate{}, fmt.Errorf("OIDC subject observation unavailable")
	}
	return CloneOIDCSubject(o.value), nil
}

type OIDCSubjectChange struct{ before, after OIDCSubjectTemplate }

func NewOIDCSubjectChange(before, after OIDCSubjectTemplate) (OIDCSubjectChange, error) {
	if err := ValidateOIDCSubject(before); err != nil {
		return OIDCSubjectChange{}, err
	}
	if err := ValidateOIDCSubject(after); err != nil {
		return OIDCSubjectChange{}, err
	}
	if OIDCSubjectMatches(before, after) {
		return OIDCSubjectChange{}, fmt.Errorf("unchanged OIDC template")
	}
	return OIDCSubjectChange{CloneOIDCSubject(before), CloneOIDCSubject(after)}, nil
}
func (c OIDCSubjectChange) Field() string { return OIDCSubjectPath }
func (c OIDCSubjectChange) Values() (OIDCSubjectTemplate, OIDCSubjectTemplate) {
	return CloneOIDCSubject(c.before), CloneOIDCSubject(c.after)
}
func CompareOIDCSubject(o OIDCSubjectObservation, d *OIDCSubjectTemplate) (int, []OIDCSubjectChange, error) {
	if d == nil {
		return 0, nil, nil
	}
	if err := ValidateOIDCSubject(*d); err != nil {
		return 0, nil, err
	}
	before, err := o.Value()
	if err != nil {
		return 0, nil, err
	}
	if OIDCSubjectMatches(before, *d) {
		return 1, nil, nil
	}
	change, err := NewOIDCSubjectChange(before, *d)
	if err != nil {
		return 0, nil, err
	}
	return 1, []OIDCSubjectChange{change}, nil
}
