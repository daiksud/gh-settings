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
