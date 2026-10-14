package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const settingsPath = ".github/settings.yml"

func validateCurrent() error {
	_, err := loadCurrentConfig()
	return err
}

func loadCurrentConfig() (Config, error) {
	root, err := gitRoot()
	if err != nil {
		return Config{}, fmt.Errorf("locate %s: current directory is not in a Git repository: %w", settingsPath, err)
	}
	data, err := os.ReadFile(filepath.Join(root, settingsPath))
	if err != nil {
		return Config{}, fmt.Errorf("read %s: %w", settingsPath, err)
	}
	config, err := ParseConfig(data)
	if err != nil {
		return Config{}, fmt.Errorf("validate %s: %w", settingsPath, err)
	}
	return config, nil
}
