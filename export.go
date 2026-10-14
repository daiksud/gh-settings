package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"go.yaml.in/yaml/v4"
)

func exportCurrent() ([]byte, error) {
	repo, err := currentRepository()
	if err != nil {
		return nil, err
	}
	root, err := gitRoot()
	if err != nil {
		return nil, fmt.Errorf("locate %s: %w", settingsPath, err)
	}
	client, err := api.NewRESTClient(api.ClientOptions{Host: repo.Host, Timeout: 30 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("configure GitHub API client: %w", err)
	}
	data, err := exportSettings(repo, client)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(root, settingsPath)
	dir := filepath.Dir(path)
	if err := os.Mkdir(dir, 0755); err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("create .github: %w", err)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, fmt.Errorf("inspect .github: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf(".github must be a directory, not a symlink or other file")
	}
	if err := writeExportFile(path, data); err != nil {
		return nil, fmt.Errorf("write %s: %w", settingsPath, err)
	}
	return nil, nil
}

func writeExportFile(path string, data []byte) error {
	info, err := os.Lstat(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if info != nil && !info.Mode().IsRegular() {
		return fmt.Errorf("destination must be a regular file, not a symlink or other file")
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".settings-*.yml")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	defer temp.Close()
	if info != nil && info.Mode().IsRegular() {
		if err := temp.Chmod(info.Mode().Perm()); err != nil {
			return err
		}
	}
	if _, err := temp.Write(data); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	// Publish only after every fallible file operation has completed.
	return os.Rename(temp.Name(), path)
}

func exportSettings(repo repository.Repository, client *api.RESTClient) ([]byte, error) {
	result, err := readRepositorySettings(repo, client)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(Config{SchemaVersion: 1, Settings: configSettings{Repository: result}})
}
