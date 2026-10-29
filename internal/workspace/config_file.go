package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// SettingsPath is the fixed configuration path relative to a checkout root.
const SettingsPath = ".github/settings.yml"

func ReadConfig(root string) (uimodel.Repository, error) {
	data, err := os.ReadFile(filepath.Join(root, SettingsPath))
	if err != nil {
		return uimodel.Repository{}, fmt.Errorf("read %s: %w", SettingsPath, err)
	}
	declaration, err := DecodeConfig(data)
	if err != nil {
		return uimodel.Repository{}, fmt.Errorf("validate %s: %w", SettingsPath, err)
	}
	return declaration, nil
}

// WriteConfig saves already-encoded configuration bytes at the checkout root.
func WriteConfig(root string, data []byte) error {
	path := filepath.Join(root, SettingsPath)
	dir := filepath.Dir(path)
	if err := os.Mkdir(dir, 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("create .github: %w", err)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("inspect .github: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf(".github must be a directory, not a symlink or other file")
	}
	info, err = os.Lstat(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("inspect %s: %w", SettingsPath, err)
	}
	if info != nil && !info.Mode().IsRegular() {
		return fmt.Errorf("%s must be a regular file, not a symlink or other file", SettingsPath)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", SettingsPath, err)
	}
	return nil
}

func saveDeclaration(root string, declaration uimodel.Repository) error {
	data, err := EncodeConfig(declaration)
	if err != nil {
		return err
	}
	return WriteConfig(root, data)
}
