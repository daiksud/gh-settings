package app

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

// Validate reads the current checkout's declaration without remote access or writes.
func Validate() (settings.Declaration, error) {
	root, err := workspace.GitRoot()
	if err != nil {
		return settings.Declaration{}, fmt.Errorf("locate %s: current directory is not in a Git repository: %w", workspace.SettingsPath, err)
	}
	return workspace.ReadConfig(root)
}
