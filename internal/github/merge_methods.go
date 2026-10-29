package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// mergeMethodsQuery reads only the merge-related settings through the same read-only GraphQL path.
const mergeMethodsQuery = `query($owner: String!, $name: String!) {
 repository(owner: $owner, name: $name) {
  mergeCommitAllowed
  squashMergeAllowed
  rebaseMergeAllowed
  deleteBranchOnMerge
  autoMergeAllowed
  allowUpdateBranch
  squashMergeCommitTitle
  squashMergeCommitMessage
  mergeCommitTitle
  mergeCommitMessage
 }
}`

// MergeMethodsClient keeps the narrow GraphQL read separate from REST writes.
type MergeMethodsClient struct{ graphql *api.GraphQLClient }

func NewMergeMethodsReadClient(host string) (*MergeMethodsClient, error) {
	client, err := api.NewGraphQLClient(api.ClientOptions{Host: host, Timeout: 30 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("configure GitHub GraphQL read client: %w", err)
	}
	return &MergeMethodsClient{graphql: client}, nil
}

func (client *MergeMethodsClient) ReadMergeMethods(repo repository.Repository) (settings.MergeMethodsState, error) {
	// Raw values distinguish missing/null from a valid false. GraphQL can return
	// partial data together with errors; discard it whenever Do returns an error.
	var response struct {
		Repository *map[string]json.RawMessage `json:"repository"`
	}
	if err := client.graphql.Do(mergeMethodsQuery, map[string]interface{}{"owner": repo.Owner, "name": repo.Name}, &response); err != nil {
		return settings.MergeMethodsState{}, fmt.Errorf("read repository merge methods: %w", err)
	}
	if response.Repository == nil {
		return settings.MergeMethodsState{}, fmt.Errorf("GitHub returned no repository merge methods")
	}
	var result settings.MergeMethodsState
	for _, field := range []struct {
		name   string
		target *bool
	}{
		{"mergeCommitAllowed", &result.AllowMergeCommit},
		{"squashMergeAllowed", &result.AllowSquashMerge},
		{"rebaseMergeAllowed", &result.AllowRebaseMerge},
		{"deleteBranchOnMerge", &result.DeleteBranchOnMerge},
		{"autoMergeAllowed", &result.AllowAutoMerge},
		{"allowUpdateBranch", &result.AllowUpdateBranch},
	} {
		value, present := (*response.Repository)[field.name]
		if !present {
			return settings.MergeMethodsState{}, fmt.Errorf("GitHub merge methods response is missing %s", field.name)
		}
		if !bytes.Equal(bytes.TrimSpace(value), []byte("true")) && !bytes.Equal(bytes.TrimSpace(value), []byte("false")) {
			return settings.MergeMethodsState{}, fmt.Errorf("GitHub merge method %s must be a boolean, not null or another type", field.name)
		}
		if err := json.Unmarshal(value, field.target); err != nil {
			return settings.MergeMethodsState{}, fmt.Errorf("decode merge method %s: %w", field.name, err)
		}
	}
	for _, field := range []struct {
		name   string
		target *string
	}{
		{"squashMergeCommitTitle", &result.SquashMergeCommitTitle},
		{"squashMergeCommitMessage", &result.SquashMergeCommitMessage},
		{"mergeCommitTitle", &result.MergeCommitTitle},
		{"mergeCommitMessage", &result.MergeCommitMessage},
	} {
		value, present := (*response.Repository)[field.name]
		if !present {
			return settings.MergeMethodsState{}, fmt.Errorf("GitHub merge methods response is missing %s", field.name)
		}
		var acquired *string
		if err := json.Unmarshal(value, &acquired); err != nil || acquired == nil {
			return settings.MergeMethodsState{}, fmt.Errorf("GitHub merge method %s must be an acquired string, not null or another type", field.name)
		}
		*field.target = *acquired
	}
	squash, err := prepareCommitMessageObservation(&result.SquashMergeCommitTitle, &result.SquashMergeCommitMessage, settings.SquashCommitMenu)
	if err != nil {
		return settings.MergeMethodsState{}, err
	}
	merge, err := prepareCommitMessageObservation(&result.MergeCommitTitle, &result.MergeCommitMessage, settings.MergeCommitMenu)
	if err != nil {
		return settings.MergeMethodsState{}, err
	}
	result.SquashMergeCommitTitle, result.SquashMergeCommitMessage = squash.Pair.Title, squash.Pair.Message
	result.MergeCommitTitle, result.MergeCommitMessage = merge.Pair.Title, merge.Pair.Message
	result.SquashCommitNormalization, result.MergeCommitNormalization = squash.Normalization, merge.Normalization
	return result, nil
}
