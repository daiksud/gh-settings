package app

import "github.com/daiksud/gh-settings/internal/settings"

// DeclarationLoader supplies a locally validated declaration or its loading error.
type DeclarationLoader func() (settings.Declaration, error)

// Validate reads the current checkout's declaration without remote access or writes.
func Validate(load DeclarationLoader) (settings.Declaration, error) {
	return load()
}
