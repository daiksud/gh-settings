package github

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func (client *Client) ReadTopics(repo repository.Repository) ([]string, error) {
	var response struct {
		Names json.RawMessage `json:"names"`
	}
	if err := client.rest.Get("repos/"+repo.Owner+"/"+repo.Name+"/topics", &response); err != nil {
		return nil, fmt.Errorf("read repository topics: %w", err)
	}
	if len(response.Names) == 0 || string(response.Names) == "null" {
		return nil, fmt.Errorf("GitHub response is missing topics names array")
	}
	var names []string
	if err := json.Unmarshal(response.Names, &names); err != nil {
		return nil, fmt.Errorf("decode repository topics: %w", err)
	}
	if err := settings.ValidateTopics(names); err != nil {
		return nil, fmt.Errorf("decode repository topics: %w", err)
	}
	return settings.SortedTopics(names), nil
}

func (client *Client) ApplyTopics(repo repository.Repository, desired settings.TopicsDeclaration) error {
	if desired.Names == nil {
		return nil
	}
	payload := struct {
		Names []string `json:"names"`
	}{Names: settings.SortedTopics(*desired.Names)}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode repository topics: %w", err)
	}
	if err := client.rest.Put("repos/"+repo.Owner+"/"+repo.Name+"/topics", bytes.NewReader(body), nil); err != nil {
		return fmt.Errorf("apply repository topics (any earlier repository settings update remains applied): %w", err)
	}
	return nil
}
