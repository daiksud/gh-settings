package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// ReadLabels completes every page before exposing a collection to export/plan/apply.
func (client *Client) ReadLabels(repo repository.Repository) ([]settings.LabelState, error) {
	result := []settings.LabelState{}
	for page := 1; ; page++ {
		var raw json.RawMessage
		path := fmt.Sprintf("repos/%s/%s/labels?per_page=100&page=%d", repo.Owner, repo.Name, page)
		if err := client.rest.Get(path, &raw); err != nil {
			return nil, fmt.Errorf("read repository labels page %d: %w", page, err)
		}
		if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
			return nil, fmt.Errorf("GitHub returned no labels array")
		}
		var items []map[string]json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, fmt.Errorf("decode repository labels: %w", err)
		}
		for _, item := range items {
			var label settings.LabelState
			for _, field := range []struct {
				name   string
				target *string
			}{{"name", &label.Name}, {"color", &label.Color}, {"description", &label.Description}} {
				value, present := item[field.name]
				if !present {
					return nil, fmt.Errorf("GitHub label response is missing %s", field.name)
				}
				if string(value) == "null" {
					if field.name == "description" {
						continue
					}
					return nil, fmt.Errorf("GitHub returned null label %s", field.name)
				}
				if err := json.Unmarshal(value, field.target); err != nil {
					return nil, fmt.Errorf("decode label %s: %w", field.name, err)
				}
			}
			label.Color = strings.ToLower(label.Color)
			result = append(result, label)
		}
		if len(items) < 100 {
			break
		}
	}
	// Reuse domain completeness/identity checks without interpreting membership as a write.
	declarations := make([]settings.LabelDeclaration, 0, len(result))
	for _, label := range result {
		color, description := label.Color, label.Description
		declarations = append(declarations, settings.LabelDeclaration{Name: label.Name, Color: &color, Description: &description})
	}
	if err := settings.ValidateLabels(declarations); err != nil {
		return nil, fmt.Errorf("decode repository labels: %w", err)
	}
	return result, nil
}

func labelPath(repo repository.Repository, name string) string {
	escaped := url.PathEscape(name)
	// A literal dot segment is an identity, not URL traversal.
	if name == "." || name == ".." {
		escaped = strings.ReplaceAll(name, ".", "%2E")
	}
	return "repos/" + repo.Owner + "/" + repo.Name + "/labels/" + escaped
}

// ApplyLabels executes only a fresh, typed comparison in its existing stable order.
func (client *Client) ApplyLabels(repo repository.Repository, changes []settings.LabelChange) error {
	if err := settings.ValidateLabelChanges(changes); err != nil {
		return err
	}
	for _, change := range changes {
		var err error
		switch change.Operation {
		case "create":
			payload := struct {
				Name        string  `json:"name"`
				Color       *string `json:"color,omitempty"`
				Description *string `json:"description,omitempty"`
			}{change.Name, change.After.Color, change.After.Description}
			body, encodeErr := json.Marshal(payload)
			if encodeErr != nil {
				return encodeErr
			}
			err = client.rest.Post("repos/"+repo.Owner+"/"+repo.Name+"/labels", bytes.NewReader(body), nil)
		case "update":
			payload := struct {
				Color       *string `json:"color,omitempty"`
				Description *string `json:"description,omitempty"`
			}{change.After.Color, change.After.Description}
			body, encodeErr := json.Marshal(payload)
			if encodeErr != nil {
				return encodeErr
			}
			err = client.rest.Patch(labelPath(repo, change.Name), bytes.NewReader(body), nil)
		case "delete":
			err = client.rest.Delete(labelPath(repo, change.Name), nil)
		}
		if err != nil {
			return fmt.Errorf("%s label %q (earlier successful writes remain applied): %w", change.Operation, change.Name, err)
		}
	}
	return nil
}
