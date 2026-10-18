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

// mergeMethodsQuery reads only the three settings unavailable to read-only REST callers.
const mergeMethodsQuery = `query($owner: String!, $name: String!) {
 repository(owner: $owner, name: $name) {
  mergeCommitAllowed
  squashMergeAllowed
  rebaseMergeAllowed
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
	return result, nil
}
