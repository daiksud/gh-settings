package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteExportFileRejectsSymlinkReplacement(t *testing.T) {
	for _, component := range []string{".github", "settings.yml"} {
		t.Run(component, func(t *testing.T) {
			root := t.TempDir()
			outside := t.TempDir()
			github := filepath.Join(root, ".github")
			if err := os.Mkdir(github, 0755); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(github, "settings.yml")
			if err := os.WriteFile(path, []byte("original\n"), 0644); err != nil {
				t.Fatal(err)
			}
			sentinel := filepath.Join(outside, "settings.yml")
			if err := os.WriteFile(sentinel, []byte("external configuration\n"), 0644); err != nil {
				t.Fatal(err)
			}
			link, target := path, sentinel
			if component == ".github" {
				link, target = github, outside
			}
			// Replace a previously regular destination before entering the writer.
			if err := os.RemoveAll(link); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, link); err != nil {
				t.Fatal(err)
			}
			err := writeExportFile(root, []byte("exported configuration\n"))
			data, readErr := os.ReadFile(sentinel)
			if err == nil || readErr != nil || string(data) != "external configuration\n" {
				t.Fatalf("write error=%v external target=%q read error=%v", err, data, readErr)
			}
		})
	}
}
