package main

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyCurrent() ([]byte, error) {
	result, err := app.Apply(func(repo repository.Repository, desired settings.RepositoryDeclaration) error {
		client, err := github.NewWriteClient(repo.Host)
		if err != nil {
			return err
		}
		return client.ApplyRepository(repo, desired)
	})
	if err != nil {
		return nil, err
	}
	if result.Applied {
		return []byte("Applied repository settings.\n"), nil
	}
	return []byte("No managed repository settings.\n"), nil
}
