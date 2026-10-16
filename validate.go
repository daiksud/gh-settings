package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

const settingsPath = ".github/settings.yml"

func validateCurrent() error {
	_, err := loadCurrentConfig()
	return err
}

func loadCurrentConfig() (settings.RepositoryDeclaration, error) {
	root, err := gitRoot()
	if err != nil {
		return settings.RepositoryDeclaration{}, fmt.Errorf("locate %s: current directory is not in a Git repository: %w", settingsPath, err)
	}
	data, err := os.ReadFile(filepath.Join(root, settingsPath))
	if err != nil {
		return settings.RepositoryDeclaration{}, fmt.Errorf("read %s: %w", settingsPath, err)
	}
	declaration, err := workspace.DecodeConfig(data)
	if err != nil {
		return settings.RepositoryDeclaration{}, fmt.Errorf("validate %s: %w", settingsPath, err)
	}
	return declaration, nil
}
