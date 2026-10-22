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
 repository(owner: $owner, name: $name, followRenames: false) { hasPullRequestsEnabled pullRequestCreationPolicy }
}`
const repositoryAccessIdentityQuery = `query($owner: String!, $name: String!) {
 repository(owner: $owner, name: $name, followRenames: false) { id nameWithOwner }
}`

type RepositoryAccessClient struct{ graphql *api.GraphQLClient }

// NewRepositoryAccessClient shares gh authentication without broadening grants. Queries
// and mutations are bounded and never follow redirects to a different target.
func NewRepositoryAccessClient(host string) (*RepositoryAccessClient, error) {
	client, err := api.NewGraphQLClient(api.ClientOptions{Host: host, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, fmt.Errorf("configure repository access API: %w", err)
	}
	return &RepositoryAccessClient{graphql: client}, nil
}
func accessObject(raw json.RawMessage, name string) (map[string]json.RawMessage, error) {
	var value map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &value) != nil || value == nil {
		return nil, fmt.Errorf("GitHub returned missing/null or invalid %s", name)
	}
	return value, nil
}
func accessString(fields map[string]json.RawMessage, key string) (string, error) {
	var v *string
	if json.Unmarshal(fields[key], &v) != nil || v == nil || strings.TrimSpace(*v) == "" {
		return "", fmt.Errorf("GitHub returned missing/null or invalid %s", key)
	}
	return *v, nil
}
func (client *RepositoryAccessClient) ReadPRAccess(repo repository.Repository) (settings.PRAccessState, error) {
	var response map[string]json.RawMessage
	if err := client.graphql.Do(prAccessQuery, map[string]any{"owner": repo.Owner, "name": repo.Name}, &response); err != nil {
		return settings.PRAccessState{}, fmt.Errorf("read PR access: %w", err)
	}
	fields, err := accessObject(response["repository"], "repository PR access")
	if err != nil {
		return settings.PRAccessState{}, err
	}
	var enabled *bool
	if json.Unmarshal(fields["hasPullRequestsEnabled"], &enabled) != nil || enabled == nil {
		return settings.PRAccessState{}, fmt.Errorf("GitHub returned missing/null or invalid hasPullRequestsEnabled")
	}
	policy, err := accessString(fields, "pullRequestCreationPolicy")
	if err != nil {
		return settings.PRAccessState{}, err
	}
	if !settings.ValidPullRequestCreationPolicy(policy) {
		return settings.PRAccessState{}, fmt.Errorf("GitHub returned unsupported pullRequestCreationPolicy %q", policy)
	}
	return settings.PRAccessState{HasPullRequests: *enabled, PullRequestCreationPolicy: policy}, nil
}
func validateAccessTarget(fields map[string]json.RawMessage, repo repository.Repository, label string) (string, error) {
	id, err := accessString(fields, "id")
	if err != nil {
		return "", err
	}
	name, err := accessString(fields, "nameWithOwner")
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(name, repo.Owner+"/"+repo.Name) {
		return "", fmt.Errorf("GitHub %s target does not match checkout repository", label)
	}
	return id, nil
}
func (client *RepositoryAccessClient) ApplyRepositoryAccess(repo repository.Repository, desired settings.RepositoryDeclaration) error {
	if desired.RepositoryAccessEmpty() {
		return nil
	}
	if desired.PullRequestCreationPolicy != nil && !settings.ValidPullRequestCreationPolicy(*desired.PullRequestCreationPolicy) {
		return fmt.Errorf("unsupported pull_request_creation_policy")
	}
	if desired.IssueCreationPolicy != nil && !settings.ValidIssueCreationPolicy(*desired.IssueCreationPolicy) {
		return fmt.Errorf("unsupported issue_creation_policy")
	}
	label := repositoryAccessLabel(desired)
	var lookup map[string]json.RawMessage
	if err := client.graphql.Do(repositoryAccessIdentityQuery, map[string]any{"owner": repo.Owner, "name": repo.Name}, &lookup); err != nil {
		return fmt.Errorf("resolve %s repository ID: %w", label, err)
	}
	fields, err := accessObject(lookup["repository"], label+" repository identity")
	if err != nil {
		return err
	}
	id, err := validateAccessTarget(fields, repo, label)
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
	if desired.IssueCreationPolicy != nil {
		input["issueCreationPolicy"] = *desired.IssueCreationPolicy
		selected += " issueCreationPolicy"
	}
	if desired.HasSponsorships != nil {
		input["hasSponsorshipsEnabled"] = *desired.HasSponsorships
		selected += " hasSponsorshipsEnabled"
	}
	query := `mutation($input: UpdateRepositoryInput!) { updateRepository(input:$input) { repository { ` + selected + ` } } }`
	var response map[string]json.RawMessage
	if err := client.graphql.Do(query, map[string]any{"input": input}, &response); err != nil {
		return fmt.Errorf("apply %s (repository writes may already have applied): %w", label, err)
	}
	if err := verifyAccessMutation(response, repo, id, desired); err != nil {
		return fmt.Errorf("verify %s response (repository writes may already have applied): %w", label, err)
	}
	return nil
}
func verifyAccessMutation(response map[string]json.RawMessage, repo repository.Repository, id string, desired settings.RepositoryDeclaration) error {
	label := repositoryAccessLabel(desired)
	result, err := accessObject(response["updateRepository"], "updateRepository result")
	if err != nil {
		return err
	}
	fields, err := accessObject(result["repository"], "updated repository")
	if err != nil {
		return err
	}
	actualID, err := validateAccessTarget(fields, repo, label)
	if err != nil {
		return err
	}
	if actualID != id {
		return fmt.Errorf("GitHub returned another %s repository ID", label)
	}
	if desired.HasPullRequests != nil {
		var enabled *bool
		if json.Unmarshal(fields["hasPullRequestsEnabled"], &enabled) != nil || enabled == nil || *enabled != *desired.HasPullRequests {
			return fmt.Errorf("GitHub did not retain requested hasPullRequestsEnabled")
		}
	}
	if desired.PullRequestCreationPolicy != nil {
		actual, err := accessString(fields, "pullRequestCreationPolicy")
		if err != nil {
			return err
		}
		if actual != *desired.PullRequestCreationPolicy {
			return fmt.Errorf("GitHub did not retain requested pullRequestCreationPolicy")
		}
	}
	if desired.IssueCreationPolicy != nil {
		actual, err := accessString(fields, "issueCreationPolicy")
		if err != nil {
			return err
		}
		if actual != *desired.IssueCreationPolicy {
			return fmt.Errorf("GitHub did not retain requested issueCreationPolicy")
		}
	}
	if desired.HasSponsorships != nil {
		var enabled *bool
		if json.Unmarshal(fields["hasSponsorshipsEnabled"], &enabled) != nil || enabled == nil || *enabled != *desired.HasSponsorships {
			return fmt.Errorf("GitHub did not retain requested hasSponsorshipsEnabled")
		}
	}
	return nil
}

func repositoryAccessLabel(d settings.RepositoryDeclaration) string {
	if d.HasSponsorships != nil {
		if d.PRAccessEmpty() && d.IssueCreationPolicy == nil {
			return "sponsorships"
		}
		return "repository access"
	}
	if d.IssueCreationPolicy == nil {
		return "PR access"
	}
	if d.PRAccessEmpty() {
		return "Issue creation policy"
	}
	return "repository access"
}

const issueCreationPolicyQuery = `query($owner: String!, $name: String!) {
 repository(owner: $owner, name: $name, followRenames: false) { issueCreationPolicy }
}`

func (client *RepositoryAccessClient) ReadIssueCreationPolicy(repo repository.Repository) (settings.IssueCreationPolicyState, error) {
	var response map[string]json.RawMessage
	if err := client.graphql.Do(issueCreationPolicyQuery, map[string]any{"owner": repo.Owner, "name": repo.Name}, &response); err != nil {
		return settings.IssueCreationPolicyState{}, fmt.Errorf("read Issue creation policy: %w", err)
	}
	fields, err := accessObject(response["repository"], "repository Issue creation policy")
	if err != nil {
		return settings.IssueCreationPolicyState{}, err
	}
	policy, err := accessString(fields, "issueCreationPolicy")
	if err != nil {
		return settings.IssueCreationPolicyState{}, err
	}
	if !settings.ValidIssueCreationPolicy(policy) {
		return settings.IssueCreationPolicyState{}, fmt.Errorf("GitHub returned unsupported issueCreationPolicy %q", policy)
	}
	return settings.IssueCreationPolicyState{Policy: policy}, nil
}
