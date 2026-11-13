package app

import "github.com/daiksud/gh-settings/internal/settings"

// DeclarationLoader supplies a locally validated declaration or its loading error.
type DeclarationLoader func() (settings.Declaration, error)

// Validate reads the current checkout's declaration without remote access or writes.
func Validate(load DeclarationLoader) (settings.Declaration, error) {
	d, err := load()
	if err != nil {
		return settings.Declaration{}, err
	}
	if d.Autolinks != nil {
		if err := settings.ValidateAutolinks(*d.Autolinks); err != nil {
			return settings.Declaration{}, err
		}
	}
	if d.BranchProtectionRules != nil {
		if err := settings.ValidateClassicRules(*d.BranchProtectionRules); err != nil {
			return settings.Declaration{}, err
		}
	}
	if err := d.Repository.ValidateCommitDependencies(); err != nil {
		return settings.Declaration{}, err
	}
	if err := d.Security.Validate(); err != nil {
		return settings.Declaration{}, err
	}
	if d.OIDCSubject != nil {
		if err := settings.ValidateOIDCSubject(*d.OIDCSubject); err != nil {
			return settings.Declaration{}, err
		}
	}
	if err := d.Pages.Validate(); err != nil {
		return settings.Declaration{}, err
	}
	return d, nil
}
