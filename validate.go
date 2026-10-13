package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const settingsPath = ".github/settings.yml"

func validateCurrent() error {
	root, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return fmt.Errorf("locate %s: current directory is not in a Git repository: %w", settingsPath, err)
	}
	data, err := os.ReadFile(filepath.Join(strings.TrimSuffix(string(root), "\n"), settingsPath))
	if err != nil {
		return fmt.Errorf("read %s: %w", settingsPath, err)
	}
	if err := validateSettings(data); err != nil {
		return fmt.Errorf("validate %s: %w", settingsPath, err)
	}
	return nil
}
