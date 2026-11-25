package workspace

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"go.yaml.in/yaml/v4"
)

// PrepareApply reads one declaration snapshot. Its recorder adds only confirmed
// creation IDs to that same file, never a Git commit or a replacement declaration.
func (checkout Checkout) PrepareApply() (uimodel.Repository, func(string, string, int64) error, error) {
	root, err := gitRoot(checkout.directory)
	if err != nil {
		return uimodel.Repository{}, nil, fmt.Errorf("locate %s: %w", SettingsPath, err)
	}
	data, err := os.ReadFile(filepath.Join(root, SettingsPath))
	if err != nil {
		return uimodel.Repository{}, nil, fmt.Errorf("read %s: %w", SettingsPath, err)
	}
	desired, err := DecodeConfig(data)
	if err != nil {
		return uimodel.Repository{}, nil, fmt.Errorf("validate %s: %w", SettingsPath, err)
	}
	return desired, createdIDRecorder(root, data), nil
}

// ID writeback targets a literal object in the same YAML document that was applied.
type idCollection struct {
	path    string
	nameKey string
}

var idCollections = map[string]idCollection{
	"labels":                  {"repository.labels", "name"},
	"rulesets":                {"repository.settings.rulesets", "ruleset_name"},
	"actions_policies":        {"repository.settings.actions.policies", "policy_name"},
	"collaborators":           {"repository.settings.collaborators", "username"},
	"webhooks":                {settings.WebhooksPath, "payload_url"},
	"deploy_keys":             {settings.DeployKeysPath, "key"},
	"autolinks":               {settings.AutolinksPath, "reference_prefix"},
	"branch_protection_rules": {settings.ClassicProtectionPath, "branch_name_pattern"},
}

func createdIDRecorder(root string, snapshot []byte) func(string, string, int64) error {
	expected := bytes.Clone(snapshot)
	return func(collection, name string, id int64) error {
		target, ok := idCollections[collection]
		if !ok || id <= 0 {
			return fmt.Errorf("invalid object identity")
		}
		var document yaml.Node
		if err := yaml.Unmarshal(expected, &document); err != nil {
			return fmt.Errorf("decode original settings for ID writeback: %w", err)
		}
		if len(document.Content) != 1 {
			return fmt.Errorf("invalid original settings document")
		}
		selected, err := target.entry(document.Content[0], name)
		if err != nil {
			return err
		}
		if mappingValue(selected, "id") != nil {
			return fmt.Errorf("refusing to replace an existing ID for %q", name)
		}
		position := len(selected.Content)
		if collection == "labels" {
			position = 0
		}
		selected.Content = slices.Insert(selected.Content, position,
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
		expected = encoded.Bytes()
		return nil
	}
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
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

func (collection idCollection) entry(node *yaml.Node, name string) (*yaml.Node, error) {
	for key := range strings.SplitSeq(collection.path, ".") {
		node = mappingValue(node, key)
	}
	if node == nil || node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("cannot write ID into aliased or missing %s", collection.path)
	}
	var selected *yaml.Node
	for _, entry := range node.Content {
		if entry.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("cannot write IDs into aliased collection entries")
		}
		if mappingValue(entry, "<<") != nil {
			return nil, fmt.Errorf("cannot write IDs into merged collection entries")
		}
		key := mappingValue(entry, collection.nameKey)
		if key == nil || key.Kind != yaml.ScalarNode {
			continue
		}
		identity := key.Value
		if collection.path == settings.DeployKeysPath {
			publicKey, err := settings.ParseDeployPublicKey(identity)
			if err != nil {
				continue
			}
			identity = publicKey.Fingerprint()
		}
		if identity == name {
			if selected != nil {
				return nil, fmt.Errorf("ambiguous selected object %q", name)
			}
			selected = entry
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("selected object %q is missing from original settings", name)
	}
	return selected, nil
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
		return fmt.Errorf("replace settings with recorded IDs: %w", err)
	}
	return nil
}
