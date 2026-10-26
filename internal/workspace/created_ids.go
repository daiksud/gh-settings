package workspace

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/daiksud/gh-settings/internal/settings"
	"go.yaml.in/yaml/v4"
)

// PrepareApply reads one declaration snapshot. Its recorder adds only confirmed
// creation IDs to that same file, never a Git commit or a replacement declaration.
func (checkout Checkout) PrepareApply() (settings.Declaration, func(string, string, int64) error, error) {
	root, err := gitRoot(checkout.directory)
	if err != nil {
		return settings.Declaration{}, nil, fmt.Errorf("locate %s: %w", SettingsPath, err)
	}
	data, err := os.ReadFile(filepath.Join(root, SettingsPath))
	if err != nil {
		return settings.Declaration{}, nil, fmt.Errorf("read %s: %w", SettingsPath, err)
	}
	desired, err := DecodeConfig(data)
	if err != nil {
		return settings.Declaration{}, nil, fmt.Errorf("validate %s: %w", SettingsPath, err)
	}
	return desired, createdIDRecorder(root, data), nil
}

func createdIDRecorder(root string, snapshot []byte) func(string, string, int64) error {
	expected := bytes.Clone(snapshot)
	return func(collection, name string, id int64) error {
		if id <= 0 || (collection != "labels" && collection != "rulesets" && collection != "environments") {
			return fmt.Errorf("invalid created object identity")
		}
		var document yaml.Node
		if err := yaml.Unmarshal(expected, &document); err != nil {
			return fmt.Errorf("decode original settings for ID writeback: %w", err)
		}
		mappingValue := func(node *yaml.Node, key string) *yaml.Node {
			if node == nil || node.Kind != yaml.MappingNode {
				return nil
			}
			for i := 0; i < len(node.Content); i += 2 {
				if node.Content[i].Value == key {
					return node.Content[i+1]
				}
			}
			return nil
		}
		if len(document.Content) != 1 {
			return fmt.Errorf("invalid original settings document")
		}
		entries := mappingValue(mappingValue(document.Content[0], "settings"), collection)
		if entries == nil || entries.Kind != yaml.SequenceNode {
			return fmt.Errorf("cannot write ID into aliased or missing settings.%s", collection)
		}
		var selected *yaml.Node
		for _, entry := range entries.Content {
			if entry.Kind != yaml.MappingNode {
				return fmt.Errorf("cannot write IDs into aliased collection entries")
			}
			for i := 0; i < len(entry.Content); i += 2 {
				if entry.Content[i].Value == "<<" {
					return fmt.Errorf("cannot write IDs into merged collection entries")
				}
			}
			n := mappingValue(entry, "name")
			if n != nil && n.Kind == yaml.ScalarNode && n.Value == name {
				if selected != nil {
					return fmt.Errorf("ambiguous created object %q", name)
				}
				selected = entry
			}
		}
		if selected == nil {
			return fmt.Errorf("created object %q is missing from original settings", name)
		}
		if mappingValue(selected, "id") != nil {
			return fmt.Errorf("refusing to replace an existing ID for %q", name)
		}
		selected.Content = append(selected.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "id"},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.FormatInt(id, 10)})
		var encoded bytes.Buffer
		encoder := yaml.NewEncoder(&encoded)
		encoder.SetIndent(2)
		if err := encoder.Encode(&document); err != nil {
			return err
		}
		if err := encoder.Close(); err != nil {
			return err
		}
		if err := replaceSettingsSnapshot(root, expected, encoded.Bytes()); err != nil {
			return err
		}
		expected = bytes.Clone(encoded.Bytes())
		return nil
	}
}

// This is an atomic file replacement, not a concurrent-editor lock. Recheck the
// snapshot before rename; edits racing that last check are outside the guarantee.
func replaceSettingsSnapshot(root string, expected, replacement []byte) error {
	dir := filepath.Join(root, ".github")
	path := filepath.Join(root, SettingsPath)
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("ID writeback requires a regular .github directory")
	}
	info, err = os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("ID writeback requires a regular %s file", SettingsPath)
	}
	check := func() error {
		current, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.Equal(current, expected) {
			return fmt.Errorf("%s changed during apply; refusing ID writeback", SettingsPath)
		}
		return nil
	}
	if err := check(); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".settings-ids-*")
	if err != nil {
		return fmt.Errorf("prepare ID writeback: %w", err)
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err := file.Chmod(info.Mode().Perm()); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(replacement); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := check(); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		return fmt.Errorf("replace settings with created IDs: %w", err)
	}
	return nil
}
