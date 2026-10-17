package main

import (
	"os"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/cli"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

// version is set with -ldflags at build time.
var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "action" {
		os.Exit(actions.Run(os.Args[2:], os.Stdout, os.Stderr, version, func() (app.PlanResult, error) {
			return app.Plan(app.Reader{Repository: readRepository, Topics: readTopics})
		}))
	}
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, version, cli.Operations{
		Export:   func() error { return app.Export(app.Reader{Repository: readRepository, Topics: readTopics}) },
		Validate: app.Validate,
		Plan: func() (app.PlanResult, error) {
			return app.Plan(app.Reader{Repository: readRepository, Topics: readTopics})
		},
		Apply: func() (app.ApplyResult, error) {
			return app.Apply(app.Writer{Repository: writeRepository, Topics: writeTopics})
		},
	}))
}

func readRepository(repo repository.Repository) (settings.RepositoryState, error) {
	client, err := github.NewReadClient(repo.Host)
	if err != nil {
		return settings.RepositoryState{}, err
	}
	return client.ReadRepository(repo)
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
