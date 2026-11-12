package workspace

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// Checkout scopes local operations to an explicit working directory.
type Checkout struct{ directory string }

// NewCheckout binds a directory without resolving Git or reading configuration.
func NewCheckout(directory string) Checkout { return Checkout{directory: directory} }

// Declaration loads the fixed settings file from this directory's Git root.
func (checkout Checkout) Declaration() (uimodel.Repository, error) {
	root, err := gitRoot(checkout.directory)
	if err != nil {
		return uimodel.Repository{}, fmt.Errorf("locate %s: current directory is not in a Git repository: %w", SettingsPath, err)
	}
	return ReadConfig(root)
}

// PrepareSave reads and validates a stable existing declaration before remote
// observation. The saver refuses to overwrite concurrent local edits.
func (checkout Checkout) PrepareSave() (uimodel.Repository, func(uimodel.Repository) error, error) {
	root, err := gitRoot(checkout.directory)
	if err != nil {
		return uimodel.Repository{}, nil, fmt.Errorf("locate %s: %w", SettingsPath, err)
	}
	path := filepath.Join(root, SettingsPath)
	if info, err := os.Lstat(filepath.Dir(path)); err == nil && !info.IsDir() {
		return uimodel.Repository{}, nil, fmt.Errorf(".github must be a directory, not a symlink or other file")
	} else if err != nil && !os.IsNotExist(err) {
		return uimodel.Repository{}, nil, fmt.Errorf("inspect .github: %w", err)
	}
	var existing uimodel.Repository
	var snapshot []byte
	present := false
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return existing, nil, fmt.Errorf("%s must be a regular file", SettingsPath)
		}
		present = true
		snapshot, err = os.ReadFile(path)
		if err != nil {
			return existing, nil, fmt.Errorf("read %s: %w", SettingsPath, err)
		}
		existing, err = DecodeConfig(snapshot)
		if err != nil {
			return existing, nil, fmt.Errorf("existing %s is invalid; export cannot preserve unread settings safely; run validate first", SettingsPath)
		}
	} else if !os.IsNotExist(err) {
		return existing, nil, fmt.Errorf("inspect %s: %w", SettingsPath, err)
	}
	expected := bytes.Clone(snapshot)
	return existing, func(declaration uimodel.Repository) error {
		data, err := EncodeConfig(declaration)
		if err != nil {
			return err
		}
		if present {
			if err := replaceSettingsSnapshot(root, expected, data); err != nil {
				return fmt.Errorf("write %s: %w", SettingsPath, err)
			}
			return nil
		}
		dir := filepath.Dir(path)
		if err := os.Mkdir(dir, 0755); err != nil && !os.IsExist(err) {
			return err
		}
		info, err := os.Lstat(dir)
		if err != nil || !info.IsDir() {
			return fmt.Errorf(".github must be a directory, not a symlink or other file")
		}
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return fmt.Errorf("create %s without replacing a concurrent file: %w", SettingsPath, err)
		}
		_, writeErr := file.Write(data)
		closeErr := file.Close()
		if writeErr != nil {
			return fmt.Errorf("write %s: %w", SettingsPath, writeErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close %s: %w", SettingsPath, closeErr)
		}
		return nil
	}, nil
}
