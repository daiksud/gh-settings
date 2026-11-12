package workspace

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestExportPreparationProtectsInvalidAndConcurrentFiles(t *testing.T) {
	for _, mode := range []string{"invalid", "edited", "created-after-prepare"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			if out, err := exec.Command("git", "init", "--quiet", root).CombinedOutput(); err != nil {
				t.Fatal(err, string(out))
			}
			checkout := NewCheckout(root)
			path := filepath.Join(root, SettingsPath)
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			before := []byte("version: daiksud/gh-settings@v0\nrepository: {}\n")
			if mode == "invalid" {
				before = []byte("invalid: secret-canary\n")
			}
			if mode != "created-after-prepare" {
				if err := os.WriteFile(path, before, 0600); err != nil {
					t.Fatal(err)
				}
			}
			_, save, err := checkout.PrepareSave()
			if mode == "invalid" {
				if err == nil || strings.Contains(err.Error(), "secret-canary") {
					t.Fatal(err)
				}
				after, _ := os.ReadFile(path)
				if !bytes.Equal(before, after) {
					t.Fatal(string(after))
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			changed := []byte("# user edit\n" + string(before))
			if err := os.WriteFile(path, changed, 0600); err != nil {
				t.Fatal(err)
			}
			if err := save(uimodel.Repository{}); err == nil {
				t.Fatal("concurrent file overwritten")
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(after, changed) {
				t.Fatal(string(after))
			}
		})
	}
}
