package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

const prAccessQuery = `query($owner: String!, $name: String!) {
 repository(owner: $owner, name: $name) { hasPullRequestsEnabled pullRequestCreationPolicy }
}`
const prAccessIdentityQuery = `query($owner: String!, $name: String!) {
 repository(owner: $owner, name: $name) { id nameWithOwner }
}`

type PRAccessClient struct{ graphql *api.GraphQLClient }

// NewPRAccessClient shares gh authentication without broadening grants. Queries
// and mutations are bounded and never follow redirects to a different target.
func NewPRAccessClient(host string) (*PRAccessClient, error) {
	client, err := api.NewGraphQLClient(api.ClientOptions{Host: host, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, fmt.Errorf("configure PR access API: %w", err)
	}
	return &PRAccessClient{graphql: client}, nil
}
func prAccessObject(raw json.RawMessage, name string) (map[string]json.RawMessage, error) {
	var value map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &value) != nil || value == nil {
		return nil, fmt.Errorf("GitHub returned missing/null or invalid %s", name)
	}
	return value, nil
}
func prAccessString(fields map[string]json.RawMessage, key string) (string, error) {
	var v *string
	if json.Unmarshal(fields[key], &v) != nil || v == nil || strings.TrimSpace(*v) == "" {
		return "", fmt.Errorf("GitHub returned missing/null or invalid %s", key)
	}
	return *v, nil
}
func (client *PRAccessClient) ReadPRAccess(repo repository.Repository) (settings.PRAccessState, error) {
	var response map[string]json.RawMessage
	if err := client.graphql.Do(prAccessQuery, map[string]any{"owner": repo.Owner, "name": repo.Name}, &response); err != nil {
		return settings.PRAccessState{}, fmt.Errorf("read PR access: %w", err)
	}
	fields, err := prAccessObject(response["repository"], "repository PR access")
	if err != nil {
		return settings.PRAccessState{}, err
	}
	var enabled *bool
	if json.Unmarshal(fields["hasPullRequestsEnabled"], &enabled) != nil || enabled == nil {
		return settings.PRAccessState{}, fmt.Errorf("GitHub returned missing/null or invalid hasPullRequestsEnabled")
	}
	policy, err := prAccessString(fields, "pullRequestCreationPolicy")
	if err != nil {
		return settings.PRAccessState{}, err
	}
	if !settings.ValidPullRequestCreationPolicy(policy) {
		return settings.PRAccessState{}, fmt.Errorf("GitHub returned unsupported pullRequestCreationPolicy %q", policy)
	}
	return settings.PRAccessState{HasPullRequests: *enabled, PullRequestCreationPolicy: policy}, nil
}
func validatePRAccessTarget(fields map[string]json.RawMessage, repo repository.Repository) (string, error) {
	id, err := prAccessString(fields, "id")
	if err != nil {
		return "", err
	}
	name, err := prAccessString(fields, "nameWithOwner")
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(name, repo.Owner+"/"+repo.Name) {
		return "", fmt.Errorf("GitHub PR access target does not match checkout repository")
	}
	return id, nil
}
func (client *PRAccessClient) ApplyPRAccess(repo repository.Repository, desired settings.RepositoryDeclaration) error {
	if desired.PRAccessEmpty() {
		return nil
	}
	if desired.PullRequestCreationPolicy != nil && !settings.ValidPullRequestCreationPolicy(*desired.PullRequestCreationPolicy) {
		return fmt.Errorf("unsupported pull_request_creation_policy")
	}
	var lookup map[string]json.RawMessage
	if err := client.graphql.Do(prAccessIdentityQuery, map[string]any{"owner": repo.Owner, "name": repo.Name}, &lookup); err != nil {
		return fmt.Errorf("resolve PR access repository ID: %w", err)
	}
	fields, err := prAccessObject(lookup["repository"], "PR access repository identity")
	if err != nil {
		return err
	}
	id, err := validatePRAccessTarget(fields, repo)
	if err != nil {
		return err
	}
	input := map[string]any{"repositoryId": id}
	selected := "id nameWithOwner"
	if desired.HasPullRequests != nil {
		input["hasPullRequestsEnabled"] = *desired.HasPullRequests
		selected += " hasPullRequestsEnabled"
	}
	if desired.PullRequestCreationPolicy != nil {
		input["pullRequestCreationPolicy"] = *desired.PullRequestCreationPolicy
		selected += " pullRequestCreationPolicy"
	}
	query := `mutation($input: UpdateRepositoryInput!) { updateRepository(input:$input) { repository { ` + selected + ` } } }`
	var response map[string]json.RawMessage
	if err := client.graphql.Do(query, map[string]any{"input": input}, &response); err != nil {
		return fmt.Errorf("apply PR access (repository writes may already have applied): %w", err)
	}
	if err := verifyPRAccessMutation(response, repo, id, desired); err != nil {
		return fmt.Errorf("verify PR access response (repository writes may already have applied): %w", err)
	}
	return nil
}
func verifyPRAccessMutation(response map[string]json.RawMessage, repo repository.Repository, id string, desired settings.RepositoryDeclaration) error {
	result, err := prAccessObject(response["updateRepository"], "updateRepository result")
	if err != nil {
		return err
	}
	fields, err := prAccessObject(result["repository"], "updated repository")
	if err != nil {
		return err
	}
	actualID, err := validatePRAccessTarget(fields, repo)
	if err != nil {
		return err
	}
	if actualID != id {
		return fmt.Errorf("GitHub returned another PR access repository ID")
	}
	if desired.HasPullRequests != nil {
		var enabled *bool
		if json.Unmarshal(fields["hasPullRequestsEnabled"], &enabled) != nil || enabled == nil || *enabled != *desired.HasPullRequests {
			return fmt.Errorf("GitHub did not retain requested hasPullRequestsEnabled")
		}
	}
	if desired.PullRequestCreationPolicy != nil {
		actual, err := prAccessString(fields, "pullRequestCreationPolicy")
		if err != nil {
			return err
		}
		if actual != *desired.PullRequestCreationPolicy {
			return fmt.Errorf("GitHub did not retain requested pullRequestCreationPolicy")
		}
	}
	return nil
}
