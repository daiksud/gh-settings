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

func (client *RepositoryAccessClient) readAccessPrerequisites(repo repository.Repository, desired settings.RepositoryDeclaration) (string, repositoryCreationObservations, error) {
	if desired.RepositoryAccessEmpty() {
		return "", nil, nil
	}
	var controls []settings.CreationPolicyControl
	if desired.PullRequestCreationPolicy != nil {
		controls = append(controls, settings.PullRequestPermissionsControl)
	}
	if desired.IssueCreationPolicy != nil {
		controls = append(controls, settings.IssuePermissionsControl)
	}
	query := repositoryAccessIdentityQuery
	if len(controls) > 0 {
		fields := []string{"id", "nameWithOwner"}
		for _, control := range controls {
			fields = append(fields, creationPolicyGraphQLField(control))
		}
		query = `query($owner: String!, $name: String!) {
 repository(owner: $owner, name: $name, followRenames: false) { ` + strings.Join(fields, " ") + ` }
}`
	}
	var response map[string]json.RawMessage
	if err := client.graphql.Do(query, map[string]any{"owner": repo.Owner, "name": repo.Name}, &response); err != nil {
		return "", nil, fmt.Errorf("read repository access prerequisites: %w", err)
	}
	fields, err := accessObject(response["repository"], "repository access prerequisites")
	if err != nil {
		return "", nil, err
	}
	id, err := validateAccessTarget(fields, repo, "access prerequisites")
	if err != nil {
		return "", nil, err
	}
	var observed repositoryCreationObservations
	if len(controls) > 0 {
		observed = make(repositoryCreationObservations, len(controls))
	}
	for _, control := range controls {
		value, err := acquiredCreationPolicy(fields, creationPolicyGraphQLField(control))
		if err != nil {
			return "", nil, err
		}
		prepared, err := prepareCreationPolicyObservation(value, control)
		if err != nil {
			return "", nil, err
		}
		observed[control] = prepared
	}
	return id, observed, nil
}

func creationPolicyGraphQLField(control settings.CreationPolicyControl) string {
	if control == settings.PullRequestPermissionsControl {
		return "pullRequestCreationPolicy"
	}
	return "issueCreationPolicy"
}
