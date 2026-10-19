package app

import "github.com/daiksud/gh-settings/internal/settings"

// ExportCheckout defers target and destination resolution until export runs.
// PrepareSave resolves the destination before reads and returns its bound saver.
type ExportCheckout struct {
	Repository  TargetResolver
	PrepareSave func() (func(settings.Declaration) error, error)
}

// Export saves all observed fields only after a complete read.
func Export(checkout ExportCheckout, read Reader) error {
	repo, err := checkout.Repository()
	if err != nil {
		return err
	}
	save, err := checkout.PrepareSave()
	if err != nil {
		return err
	}
	current, err := read.Repository(repo)
	if err != nil {
		return err
	}
	declaration := settings.Declaration{Repository: current.Declaration()}
	methods, err := read.MergeMethods(repo)
	if err != nil {
		return err
	}
	declaration.Repository = declaration.Repository.WithMergeMethods(methods)
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
	return save(declaration)
}
