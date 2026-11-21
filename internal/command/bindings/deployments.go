package bindings

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func deleteEnvironment(repo repository.Repository, candidate settings.EnvironmentPruneCandidate) error {
	client, err := github.NewEnvironmentPruneClient(repo.Host)
	if err != nil {
		return &clientPreWriteFailure{err}
	}
	return client.DeleteEnvironment(repo, candidate)
}

func writeEnvironment(repo repository.Repository, desired settings.EnvironmentDeclaration) (bool, error) {
	client, err := github.NewEnvironmentsClient(repo.Host)
	if err != nil {
		return false, err
	}
	client.NormalizationWarning = warnEnvironmentDeploymentNormalization
	client.RefTypeNormalizationWarning = warnEnvironmentRefTypeNormalization
	return client.Update(repo, desired)
}

func readPages(repo repository.Repository) (settings.PagesObservation, error) {
	c, err := github.NewPagesClient(repo.Host)
	if err != nil {
		return settings.PagesObservation{}, err
	}
	return c.Read(repo)
}

func writePages(repo repository.Repository, d settings.PagesDeclaration) (bool, error) {
	c, err := github.NewPagesClient(repo.Host)
	if err != nil {
		return false, err
	}
	c.NormalizationWarning = warnNormalization
	return c.Update(repo, d)
}

func readPagesForApply(repo repository.Repository, d settings.PagesDeclaration) (settings.PagesObservation, error) {
	o, err := readPages(repo)
	if err != nil {
		return settings.PagesObservation{}, err
	}
	if absent, err := o.Absent(); err != nil {
		return settings.PagesObservation{}, err
	} else if absent || d.Delete() {
		return o, nil
	}
	if record := o.ModeNormalization(); d.BuildType != nil && record != (settings.ObservationNormalization{}) {
		warnNormalization(record)
	}
	if record := o.FolderNormalization(); d.Source != nil && record != (settings.ObservationNormalization{}) {
		warnNormalization(record)
	}
	return o, nil
}
