package app

import (
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// One raw observation supplies independent export evidence for the active template
// and optional immutable leaf. A retained leaf never enters the acquired check.
func exportOIDCSubject(checkout ExportCheckout, read Reader, repo repository.Repository, prior *settings.OIDCSubjectTemplate) (*settings.OIDCSubjectTemplate, error) {
	var diagnostic error
	info := func(path string, err error) {
		if checkout.Info != nil {
			diagnostic = errors.Join(diagnostic, checkout.Info(path, err))
		}
	}
	if read.OIDCSubject == nil {
		info(settings.OIDCSubjectPath, fmt.Errorf("reader unavailable"))
		return prior, diagnostic
	}
	observed, err := read.OIDCSubject(repo)
	if err != nil {
		info(settings.OIDCSubjectPath, err)
		return prior, diagnostic
	}
	value, err := observed.Value()
	if err != nil {
		info(settings.OIDCSubjectPath, err)
		return prior, diagnostic
	}
	var result settings.OIDCSubjectTemplate
	if prior != nil {
		result = settings.CloneOIDCSubject(*prior)
	}
	check := func(path string, acquired settings.OIDCSubjectTemplate) bool {
		err := settings.ValidateOIDCSubject(acquired)
		if err == nil && checkout.Check != nil {
			err = checkout.Check(settings.Declaration{OIDCSubject: &acquired})
		}
		if err != nil {
			info(path, err)
			return false
		}
		return true
	}
	template := settings.CloneOIDCSubject(value)
	template.UseImmutableSubjectClaim = nil
	if check(settings.OIDCTemplatePath, template) {
		result.UseDefaultTemplate, result.SubjectClaimTemplate = template.UseDefaultTemplate, template.SubjectClaimTemplate
	}
	if value.UseImmutableSubjectClaim == nil {
		info(settings.OIDCImmutablePath, fmt.Errorf("optional OIDC immutable configuration unavailable"))
	} else {
		immutable := settings.OIDCSubjectTemplate{UseImmutableSubjectClaim: value.UseImmutableSubjectClaim}
		if check(settings.OIDCImmutablePath, immutable) {
			result.UseImmutableSubjectClaim = immutable.UseImmutableSubjectClaim
		}
	}
	if result.UseDefaultTemplate == nil && result.UseImmutableSubjectClaim == nil {
		return nil, diagnostic
	}
	return &result, diagnostic
}
