package github

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// Policy strings are acquired independently from nonempty repository identities.
func acquiredCreationPolicy(fields map[string]json.RawMessage, key string) (*string, error) {
	var value *string
	if json.Unmarshal(fields[key], &value) != nil || value == nil {
		return nil, fmt.Errorf("GitHub returned missing/null or invalid %s policy string", key)
	}
	return value, nil
}

func (client *RepositoryAccessClient) readAccessPrerequisites(repo repository.Repository, desired settings.RepositoryDeclaration) (string, settings.RepositoryDeclaration, repositoryCreationObservations, error) {
	if desired.RepositoryAccessEmpty() {
		return "", settings.RepositoryDeclaration{}, nil, nil
	}
	var controls []settings.CreationPolicyControl
	if desired.PullRequestCreationPolicy != nil {
		controls = append(controls, settings.PullRequestPermissionsControl)
	}
	if desired.IssueCreationPolicy != nil {
		controls = append(controls, settings.IssuePermissionsControl)
	}
	query := `query($owner: String!, $name: String!) {
 repository(owner: $owner, name: $name, followRenames: false) { ` + strings.Join(repositoryAccessFields(desired), " ") + ` }
}`
	var response map[string]json.RawMessage
	if err := client.graphql.Do(query, map[string]any{"owner": repo.Owner, "name": repo.Name}, &response); err != nil {
		return "", settings.RepositoryDeclaration{}, nil, fmt.Errorf("read repository access prerequisites: %w", err)
	}
	fields, err := accessObject(response["repository"], "repository access prerequisites")
	if err != nil {
		return "", settings.RepositoryDeclaration{}, nil, err
	}
	id, err := validateAccessTarget(fields, repo, "access prerequisites")
	if err != nil {
		return "", settings.RepositoryDeclaration{}, nil, err
	}
	var raw settings.RepositoryDeclaration
	for _, entry := range []struct {
		name    string
		desired *bool
		value   **bool
	}{
		{"hasPullRequestsEnabled", desired.HasPullRequests, &raw.HasPullRequests},
		{"hasSponsorshipsEnabled", desired.HasSponsorships, &raw.HasSponsorships},
	} {
		if entry.desired == nil {
			continue
		}
		var value *bool
		if json.Unmarshal(fields[entry.name], &value) != nil || value == nil {
			return "", settings.RepositoryDeclaration{}, nil, fmt.Errorf("GitHub returned missing/null or invalid %s boolean", entry.name)
		}
		*entry.value = value
	}
	var observed repositoryCreationObservations
	if len(controls) > 0 {
		observed = make(repositoryCreationObservations, len(controls))
	}
	for _, control := range controls {
		value, err := acquiredCreationPolicy(fields, creationPolicyGraphQLField(control))
		if err != nil {
			return "", settings.RepositoryDeclaration{}, nil, err
		}
		prepared, err := prepareCreationPolicyObservation(value, control)
		if err != nil {
			return "", settings.RepositoryDeclaration{}, nil, err
		}
		observed[control] = prepared
		if control == settings.PullRequestPermissionsControl {
			raw.PullRequestCreationPolicy = value
		} else {
			raw.IssueCreationPolicy = value
		}
	}
	return id, raw, observed, nil
}

func repositoryAccessFields(desired settings.RepositoryDeclaration) []string {
	fields := []string{"id", "nameWithOwner"}
	if desired.HasPullRequests != nil {
		fields = append(fields, "hasPullRequestsEnabled")
	}
	if desired.PullRequestCreationPolicy != nil {
		fields = append(fields, "pullRequestCreationPolicy")
	}
	if desired.IssueCreationPolicy != nil {
		fields = append(fields, "issueCreationPolicy")
	}
	if desired.HasSponsorships != nil {
		fields = append(fields, "hasSponsorshipsEnabled")
	}
	return fields
}

func creationPolicyGraphQLField(control settings.CreationPolicyControl) string {
	if control == settings.PullRequestPermissionsControl {
		return "pullRequestCreationPolicy"
	}
	return "issueCreationPolicy"
}
