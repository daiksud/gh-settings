package app

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

// Export saves all observed fields only after a complete read and serialization.
func Export(read Reader) error {
	repo, err := workspace.CurrentRepository()
	if err != nil {
		return err
	}
	root, err := workspace.GitRoot()
	if err != nil {
		return fmt.Errorf("locate %s: %w", workspace.SettingsPath, err)
	}
	current, err := read.Repository(repo)
	if err != nil {
		return err
	}
	declaration := settings.Declaration{Repository: settings.RepositoryDeclaration{
		Description: &current.Description, Homepage: &current.Homepage,
		HasIssues: &current.HasIssues, HasProjects: &current.HasProjects, HasWiki: &current.HasWiki,
	}}
	names, err := read.Topics(repo)
	if err != nil {
		return err
	}
	names = settings.SortedTopics(names)
	declaration.Topics.Names = &names
	labels, err := read.Labels(repo)
	if err != nil {
		return err
	}
	labelDeclarations := make([]settings.LabelDeclaration, 0, len(labels))
	for _, label := range labels {
		color, description := label.Color, label.Description
		labelDeclarations = append(labelDeclarations, settings.LabelDeclaration{Name: label.Name, Color: &color, Description: &description})
	}
	declaration.Labels = &labelDeclarations
	data, err := workspace.EncodeConfig(declaration)
	if err != nil {
		return err
	}
	return workspace.WriteConfig(root, data)
}
