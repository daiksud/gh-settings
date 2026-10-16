package main

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func validateCurrent() error {
	_, err := loadCurrentConfig()
	return err
}

func loadCurrentConfig() (settings.RepositoryDeclaration, error) {
	root, err := workspace.GitRoot()
	if err != nil {
		return settings.RepositoryDeclaration{}, fmt.Errorf("locate %s: current directory is not in a Git repository: %w", workspace.SettingsPath, err)
	}
	return workspace.ReadConfig(root)
}
