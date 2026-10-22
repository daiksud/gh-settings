package github

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func decodeArchive(raw map[string]json.RawMessage, repo repository.Repository) (settings.ArchiveState, error) {
	var state settings.ArchiveState
	if len(raw) == 0 {
		return state, fmt.Errorf("GitHub returned no repository archive identity")
	}
	if raw["id"] == nil || json.Unmarshal(raw["id"], &state.ID) != nil || state.ID <= 0 {
		return settings.ArchiveState{}, fmt.Errorf("GitHub returned missing or invalid archive repository ID")
	}
	if raw["full_name"] == nil || json.Unmarshal(raw["full_name"], &state.FullName) != nil {
		return settings.ArchiveState{}, fmt.Errorf("GitHub returned missing or invalid archive repository full name")
	}
	var archived *bool
	if raw["archived"] == nil || json.Unmarshal(raw["archived"], &archived) != nil || archived == nil {
		return settings.ArchiveState{}, fmt.Errorf("GitHub returned missing or invalid archived state")
	}
	state.Archived = *archived
	if err := state.Validate(repo); err != nil {
		return settings.ArchiveState{}, err
	}
	return state, nil
}

// ReadArchive checks all lifecycle-relevant fields and rejects redirected targets.
// Use NewWriteClient for this path to preserve its no-redirect policy.
func (client *Client) ReadArchive(repo repository.Repository) (settings.ArchiveState, error) {
	var response map[string]json.RawMessage
	if err := client.rest.Get("repos/"+repo.Owner+"/"+repo.Name, &response); err != nil {
		return settings.ArchiveState{}, fmt.Errorf("read repository archive state: %w", err)
	}
	return decodeArchive(response, repo)
}

// UpdateArchive sends exactly one lifecycle field and confirms the expected identity
// in both the response and an independent readback. Callers perform fresh preflight.
func (client *Client) UpdateArchive(repo repository.Repository, initial settings.ArchiveState, archived bool) error {
	if err := initial.Validate(repo); err != nil {
		return err
	}
	body, _ := json.Marshal(struct {
		Archived bool `json:"archived"`
	}{Archived: archived})
	var response map[string]json.RawMessage
	if err := client.rest.Patch("repos/"+repo.Owner+"/"+repo.Name, bytes.NewReader(body), &response); err != nil {
		return fmt.Errorf("update repository archived: %w", err)
	}
	updated, err := decodeArchive(response, repo)
	if err != nil || updated.ID != initial.ID || updated.FullName != initial.FullName || updated.Archived != archived {
		return fmt.Errorf("archive PATCH response did not confirm requested state and repository identity; write may have applied")
	}
	observed, err := client.ReadArchive(repo)
	if err != nil || observed.ID != initial.ID || observed.FullName != initial.FullName || observed.Archived != archived {
		return fmt.Errorf("archive readback did not confirm requested state and repository identity; write may have applied: %v", err)
	}
	return nil
}
