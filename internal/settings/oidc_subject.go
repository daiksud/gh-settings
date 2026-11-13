package settings

import (
	"fmt"
	"slices"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

const OIDCSubjectPath = "repository.settings.actions.oidc.subject_claim"
const OIDCImmutablePath = OIDCSubjectPath + ".use_immutable_subject_claim"
const OIDCTemplatePath = OIDCSubjectPath + ".use_default_template"

type OIDCSubjectTemplate = uimodel.OIDCSubjectTemplate

func ValidateOIDCSubject(d OIDCSubjectTemplate) error {
	if d.UseDefaultTemplate == nil {
		if d.SubjectClaimTemplate == nil && d.UseImmutableSubjectClaim != nil {
			return nil
		}
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
	if v.UseImmutableSubjectClaim != nil {
		x := *v.UseImmutableSubjectClaim
		v.UseImmutableSubjectClaim = &x
	}
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
	if ValidateOIDCSubject(a) != nil || ValidateOIDCSubject(b) != nil {
		return false
	}
	if b.UseImmutableSubjectClaim != nil && (a.UseImmutableSubjectClaim == nil || *a.UseImmutableSubjectClaim != *b.UseImmutableSubjectClaim) {
		return false
	}
	if b.UseDefaultTemplate == nil {
		return true
	}
	return a.UseDefaultTemplate != nil && *a.UseDefaultTemplate == *b.UseDefaultTemplate && (*a.UseDefaultTemplate || slices.Equal(*a.SubjectClaimTemplate, *b.SubjectClaimTemplate))
}

// Selected evidence never turns an unknown optional member into a desired value.
func OIDCSubjectBefore(observed, desired OIDCSubjectTemplate) (OIDCSubjectTemplate, error) {
	if err := ValidateOIDCSubject(desired); err != nil {
		return OIDCSubjectTemplate{}, err
	}
	before := CloneOIDCSubject(observed)
	if desired.UseDefaultTemplate == nil {
		before.UseDefaultTemplate, before.SubjectClaimTemplate = nil, nil
	} else if before.UseDefaultTemplate == nil {
		return OIDCSubjectTemplate{}, fmt.Errorf("OIDC active template unavailable")
	}
	if desired.UseImmutableSubjectClaim == nil {
		before.UseImmutableSubjectClaim = nil
	} else if before.UseImmutableSubjectClaim == nil {
		return OIDCSubjectTemplate{}, fmt.Errorf("OIDC immutable configuration unavailable")
	}
	return before, ValidateOIDCSubject(before)
}

type OIDCSubjectObservation struct {
	acquired bool
	value    OIDCSubjectTemplate
}

func NewOIDCSubjectObservation(v OIDCSubjectTemplate) (OIDCSubjectObservation, error) {
	if v.UseDefaultTemplate == nil {
		return OIDCSubjectObservation{}, fmt.Errorf("OIDC active template unavailable")
	}
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
	if (before.UseDefaultTemplate == nil) != (after.UseDefaultTemplate == nil) || (before.UseImmutableSubjectClaim == nil) != (after.UseImmutableSubjectClaim == nil) {
		return OIDCSubjectChange{}, fmt.Errorf("OIDC report ownership units differ")
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
	before, err = OIDCSubjectBefore(before, *d)
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
