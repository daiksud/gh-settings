package acceptance

import (
	"fmt"
	"os"
	"path/filepath"
)

func writeSettingsFile(work, input string) error {
	if input == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Join(work, ".github"), 0700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(work, ".github", "settings.yml"), []byte(input), 0600)
}

func checkSettingsUnchanged(work, input string) error {
	if input == "" {
		return nil
	}
	unchanged, err := os.ReadFile(filepath.Join(work, ".github", "settings.yml"))
	if err != nil || string(unchanged) != input {
		return fmt.Errorf("configuration changed: %q, error=%v", unchanged, err)
	}
	return nil
}
