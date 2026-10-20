package app

import "github.com/daiksud/gh-settings/internal/settings"

// ExportCheckout defers target and destination resolution until export runs.
// PrepareSave resolves the destination before reads and returns its bound saver.
type ExportCheckout struct {
	Repository       TargetResolver
	IncludeActions   bool
	IncludeRulesets  bool
	ExcludedRulesets func(int)
	PrepareSave      func() (func(settings.Declaration) error, error)
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
	observed, err := read.ObserveRepository(repo, settings.AllRepositoryFields())
	if err != nil {
		return err
	}
	repositoryDeclaration, err := observed.Declaration()
	if err != nil {
		return err
	}
	declaration := settings.Declaration{Repository: repositoryDeclaration}
	names, err := read.Topics(repo)
	if err != nil {
		return err
	}
	names = settings.SortedTopics(names)
	declaration.Topics.Names = &names
	labelObservation, err := read.Labels(repo)
	if err != nil {
		return err
	}
	labels, err := labelObservation.Values()
	if err != nil {
		return err
	}
	labelDeclarations := make([]settings.LabelDeclaration, 0, len(labels))
	for _, label := range labels {
		color, description := label.Color, label.Description
		labelDeclarations = append(labelDeclarations, settings.LabelDeclaration{Name: label.Name, Color: &color, Description: &description})
	}
	validated, err := settings.NewLabelDeclarations(labelDeclarations)
	if err != nil {
		return err
	}
	declaration.Labels = &validated
	if checkout.IncludeRulesets {
		observed, err := read.Rulesets(repo)
		if err != nil {
			return err
		}
		values, err := observed.Values()
		if err != nil {
			return err
		}
		entries := make([]settings.RulesetDeclaration, 0, len(values))
		for _, value := range values {
			entries = append(entries, value.Policy)
		}
		desired, err := settings.NewRulesetDeclarations(entries)
		if err != nil {
			return err
		}
		declaration.Rulesets = &desired
		if observed.ExcludedPush() > 0 && checkout.ExcludedRulesets != nil {
			checkout.ExcludedRulesets(observed.ExcludedPush())
		}
	}
	if checkout.IncludeActions {
		observed, err := read.Actions(repo, true)
		if err != nil {
			return err
		}
		declaration.Actions, err = observed.Declaration()
		if err != nil {
			return err
		}
	}
	return save(declaration)
}
