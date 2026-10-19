package main

import (
	"os"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/cli"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

// version is set with -ldflags at build time.
var version = "dev"

func main() {
	checkout := workspace.NewCheckout(".")
	reader := app.Reader{ObserveRepository: github.ObserveRepository, Topics: readTopics, Labels: readLabels}
	if len(os.Args) > 1 && os.Args[1] == "action" {
		os.Exit(actions.Run(os.Args[2:], os.Stdout, os.Stderr, version, func() (app.PlanResult, error) {
			return app.Plan(checkout.Declaration, checkout.Repository, reader)
		}))
	}
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, version, cli.Operations{
		Export: func() error {
			return app.Export(app.ExportCheckout{Repository: checkout.Repository, PrepareSave: checkout.PrepareSave}, reader)
		},
		Validate: func() (settings.Declaration, error) { return app.Validate(checkout.Declaration) },
		Plan: func() (app.PlanResult, error) {
			return app.Plan(checkout.Declaration, checkout.Repository, reader)
		},
		Apply: func() (app.ApplyResult, error) {
			return app.Apply(checkout.Declaration, checkout.Repository, app.Writer{Repository: writeRepository, Topics: writeTopics, ReadLabels: readLabelsForApply, Labels: writeLabels})
		},
	}))
}

func writeRepository(repo repository.Repository, desired settings.RepositoryDeclaration) error {
	client, err := github.NewWriteClient(repo.Host)
	if err != nil {
		return err
	}
	return client.ApplyRepository(repo, desired)
}

func readTopics(repo repository.Repository) ([]string, error) {
	client, err := github.NewReadClient(repo.Host)
	if err != nil {
		return nil, err
	}
	return client.ReadTopics(repo)
}

func writeTopics(repo repository.Repository, desired settings.TopicsDeclaration) error {
	client, err := github.NewWriteClient(repo.Host)
	if err != nil {
		return err
	}
	return client.ApplyTopics(repo, desired)
}

func readLabels(repo repository.Repository) (settings.LabelObservation, error) {
	client, err := github.NewReadClient(repo.Host)
	if err != nil {
		return settings.LabelObservation{}, err
	}
	return client.ReadLabels(repo)
}
func readLabelsForApply(repo repository.Repository) (settings.LabelObservation, error) {
	client, err := github.NewWriteClient(repo.Host)
	if err != nil {
		return settings.LabelObservation{}, err
	}
	return client.ReadLabels(repo)
}
func writeLabels(repo repository.Repository, changes []settings.LabelChange) error {
	client, err := github.NewWriteClient(repo.Host)
	if err != nil {
		return err
	}
	return client.ApplyLabels(repo, changes)
}
