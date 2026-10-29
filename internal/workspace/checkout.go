package workspace

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// Checkout scopes local operations to an explicit working directory.
type Checkout struct{ directory string }

// NewCheckout binds a directory without resolving Git or reading configuration.
func NewCheckout(directory string) Checkout { return Checkout{directory: directory} }

// Declaration loads the fixed settings file from this directory's Git root.
func (checkout Checkout) Declaration() (uimodel.Repository, error) {
	root, err := gitRoot(checkout.directory)
	if err != nil {
		return uimodel.Repository{}, fmt.Errorf("locate %s: current directory is not in a Git repository: %w", SettingsPath, err)
	}
	return ReadConfig(root)
}

// PrepareSave resolves the Git root without modifying the destination. The saver
// encodes completely before applying the existing guarded file-writing policy.
func (checkout Checkout) PrepareSave() (func(uimodel.Repository) error, error) {
	root, err := gitRoot(checkout.directory)
	if err != nil {
		return nil, fmt.Errorf("locate %s: %w", SettingsPath, err)
	}
	return func(declaration uimodel.Repository) error {
		return saveDeclaration(root, declaration)
	}, nil
}
