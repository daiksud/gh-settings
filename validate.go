package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const settingsPath = ".github/settings.yml"

func validateCurrent() error {
	root, err := gitRoot()
	if err != nil {
		return fmt.Errorf("locate %s: current directory is not in a Git repository: %w", settingsPath, err)
	}
	data, err := os.ReadFile(filepath.Join(root, settingsPath))
	if err != nil {
		return fmt.Errorf("read %s: %w", settingsPath, err)
	}
	if err := validateSettings(data); err != nil {
		return fmt.Errorf("validate %s: %w", settingsPath, err)
	}
	return nil
}
