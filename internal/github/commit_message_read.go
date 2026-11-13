package github

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// readRepositoryPatchMerge adds only selected booleans and complete owned menus
// to a base observation. The normal-merge menu retains plan's enabling dependency.
func (client *MergeMethodsClient) readRepositoryPatchMerge(repo repository.Repository, desired, observed settings.RepositoryDeclaration) (settings.RepositoryDeclaration, repositoryCommitObservations, error) {
	var fields []string
	booleans := []struct {
		name     string
		desired  *bool
		observed **bool
	}{
		{"mergeCommitAllowed", desired.AllowMergeCommit, &observed.AllowMergeCommit},
		{"squashMergeAllowed", desired.AllowSquashMerge, &observed.AllowSquashMerge},
		{"rebaseMergeAllowed", desired.AllowRebaseMerge, &observed.AllowRebaseMerge},
		{"deleteBranchOnMerge", desired.DeleteBranchOnMerge, &observed.DeleteBranchOnMerge},
		{"autoMergeAllowed", desired.AllowAutoMerge, &observed.AllowAutoMerge},
		{"allowUpdateBranch", desired.AllowUpdateBranch, &observed.AllowUpdateBranch},
	}
	for _, field := range booleans {
		if field.desired != nil || field.name == "mergeCommitAllowed" && (desired.MergeCommitTitle != nil || desired.MergeCommitMessage != nil) {
			fields = append(fields, field.name)
		}
	}
	var menus []settings.CommitMessageMenu
	if desired.SquashMergeCommitTitle != nil || desired.SquashMergeCommitMessage != nil {
		menus = append(menus, settings.SquashCommitMenu)
	}
	if desired.MergeCommitTitle != nil || desired.MergeCommitMessage != nil {
		menus = append(menus, settings.MergeCommitMenu)
	}
	for _, menu := range menus {
		title, message := commitMessageGraphQLFields(menu)
		fields = append(fields, title, message)
	}
	if len(fields) == 0 {
		return observed, nil, nil
	}
	query := `query($owner: String!, $name: String!) {
 repository(owner: $owner, name: $name, followRenames: false) {
 ` + strings.Join(fields, "\n ") + `
 }
}`
	var response struct {
		Repository *map[string]json.RawMessage `json:"repository"`
	}
	if err := client.graphql.Do(query, map[string]interface{}{"owner": repo.Owner, "name": repo.Name}, &response); err != nil {
		return settings.RepositoryDeclaration{}, nil, fmt.Errorf("read repository merge settings: %w", err)
	}
	if response.Repository == nil {
		return settings.RepositoryDeclaration{}, nil, fmt.Errorf("GitHub returned no repository merge settings")
	}
	for _, field := range booleans {
		if field.desired == nil && !(field.name == "mergeCommitAllowed" && (desired.MergeCommitTitle != nil || desired.MergeCommitMessage != nil)) {
			continue
		}
		raw, present := (*response.Repository)[field.name]
		if !present || json.Unmarshal(raw, field.observed) != nil || *field.observed == nil {
			return settings.RepositoryDeclaration{}, nil, fmt.Errorf("GitHub merge setting %s must be an acquired boolean", field.name)
		}
	}
	result := make(repositoryCommitObservations, len(menus))
	for _, menu := range menus {
		titleField, messageField := commitMessageGraphQLFields(menu)
		var title, message *string
		for _, field := range []struct {
			name  string
			value **string
		}{{titleField, &title}, {messageField, &message}} {
			raw, present := (*response.Repository)[field.name]
			if !present || json.Unmarshal(raw, field.value) != nil || *field.value == nil {
				return settings.RepositoryDeclaration{}, nil, fmt.Errorf("GitHub commit message %s must be an acquired string", field.name)
			}
		}
		prepared, err := prepareCommitMessageObservation(title, message, menu)
		if err != nil {
			return settings.RepositoryDeclaration{}, nil, err
		}
		result[menu] = prepared
		// Keep native strings for reconciliation and separate normalized UI
		// provenance for warnings. Never invent a complete MergeMethodsState.
		if menu == settings.SquashCommitMenu {
			observed.SquashMergeCommitTitle, observed.SquashMergeCommitMessage = title, message
		} else {
			observed.MergeCommitTitle, observed.MergeCommitMessage = title, message
		}
	}
	return observed, result, nil
}

func commitMessageGraphQLFields(menu settings.CommitMessageMenu) (string, string) {
	if menu == settings.SquashCommitMenu {
		return "squashMergeCommitTitle", "squashMergeCommitMessage"
	}
	return "mergeCommitTitle", "mergeCommitMessage"
}
