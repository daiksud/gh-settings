package workspace

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

// Checkout scopes local operations to an explicit working directory.
type Checkout struct{ directory string }

// NewCheckout binds a directory without resolving Git or reading configuration.
func NewCheckout(directory string) Checkout { return Checkout{directory: directory} }

// Declaration loads the fixed settings file from this directory's Git root.
func (checkout Checkout) Declaration() (settings.Declaration, error) {
	root, err := gitRoot(checkout.directory)
	if err != nil {
		return settings.Declaration{}, fmt.Errorf("locate %s: current directory is not in a Git repository: %w", SettingsPath, err)
	}
	return ReadConfig(root)
}
