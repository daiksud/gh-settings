package github

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func (client *MergeMethodsClient) readCommitMessages(repo repository.Repository, desired settings.RepositoryDeclaration) (repositoryCommitObservations, error) {
	var menus []settings.CommitMessageMenu
	if desired.SquashMergeCommitTitle != nil || desired.SquashMergeCommitMessage != nil {
		menus = append(menus, settings.SquashCommitMenu)
	}
	if desired.MergeCommitTitle != nil || desired.MergeCommitMessage != nil {
		menus = append(menus, settings.MergeCommitMenu)
	}
	if len(menus) == 0 {
		return nil, nil
	}
	var fields []string
	for _, menu := range menus {
		title, message := commitMessageGraphQLFields(menu)
		fields = append(fields, title, message)
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
		return nil, fmt.Errorf("read repository commit messages: %w", err)
	}
	if response.Repository == nil {
		return nil, fmt.Errorf("GitHub returned no repository commit messages")
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
				return nil, fmt.Errorf("GitHub commit message %s must be an acquired string", field.name)
			}
		}
		prepared, err := prepareCommitMessageObservation(title, message, menu)
		if err != nil {
			return nil, err
		}
		result[menu] = prepared
	}
	return result, nil
}

func commitMessageGraphQLFields(menu settings.CommitMessageMenu) (string, string) {
	if menu == settings.SquashCommitMenu {
		return "squashMergeCommitTitle", "squashMergeCommitMessage"
	}
	return "mergeCommitTitle", "mergeCommitMessage"
}
