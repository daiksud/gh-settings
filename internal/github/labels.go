package github

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// ReadLabels completes every page before exposing a collection to export/plan/apply.
func (client *Client) ReadLabels(repo repository.Repository) (settings.LabelObservation, error) {
	result := []settings.LabelState{}
	for page := 1; ; page++ {
		var raw json.RawMessage
		path := fmt.Sprintf("repos/%s/%s/labels?per_page=100&page=%d", repo.Owner, repo.Name, page)
		if err := client.rest.Get(path, &raw); err != nil {
			return settings.LabelObservation{}, fmt.Errorf("read repository labels page %d: %w", page, err)
		}
		if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
			return settings.LabelObservation{}, fmt.Errorf("GitHub returned no labels array")
		}
		var items []map[string]json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return settings.LabelObservation{}, fmt.Errorf("decode repository labels: %w", err)
		}
		for _, item := range items {
			var label settings.LabelState
			for _, field := range []struct {
				name   string
				target *string
			}{{"name", &label.Name}, {"color", &label.Color}, {"description", &label.Description}} {
				value, present := item[field.name]
				if !present {
					return settings.LabelObservation{}, fmt.Errorf("GitHub label response is missing %s", field.name)
				}
				if string(value) == "null" {
					if field.name == "description" {
						continue
					}
					return settings.LabelObservation{}, fmt.Errorf("GitHub returned null label %s", field.name)
				}
				if err := json.Unmarshal(value, field.target); err != nil {
					return settings.LabelObservation{}, fmt.Errorf("decode label %s: %w", field.name, err)
				}
			}
			result = append(result, label)
		}
		if len(items) < 100 {
			break
		}
	}
	observed, err := settings.NewLabelObservation(result)
	if err != nil {
		return settings.LabelObservation{}, fmt.Errorf("decode repository labels: %w", err)
	}
	return observed, nil
}

func labelPath(repo repository.Repository, name string) string {
	escaped := url.PathEscape(name)
	// A literal dot segment is an identity, not URL traversal.
	if name == "." || name == ".." {
		escaped = strings.ReplaceAll(name, ".", "%2E")
	}
	return "repos/" + repo.Owner + "/" + repo.Name + "/labels/" + escaped
}
