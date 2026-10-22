package github

import (
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

const sponsorshipsQuery = `query($owner: String!, $name: String!) {
 repository(owner: $owner, name: $name, followRenames: false) { hasSponsorshipsEnabled }
}`

func (client *RepositoryAccessClient) ReadSponsorships(repo repository.Repository) (settings.SponsorshipsState, error) {
	var response map[string]json.RawMessage
	if err := client.graphql.Do(sponsorshipsQuery, map[string]any{"owner": repo.Owner, "name": repo.Name}, &response); err != nil {
		return settings.SponsorshipsState{}, fmt.Errorf("read sponsorships: %w", err)
	}
	fields, err := accessObject(response["repository"], "repository sponsorships")
	if err != nil {
		return settings.SponsorshipsState{}, err
	}
	var enabled *bool
	if json.Unmarshal(fields["hasSponsorshipsEnabled"], &enabled) != nil || enabled == nil {
		return settings.SponsorshipsState{}, fmt.Errorf("GitHub returned missing/null or invalid hasSponsorshipsEnabled")
	}
	return settings.SponsorshipsState{Enabled: *enabled}, nil
}
