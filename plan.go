package main

import (
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/cli"
)

func planCurrent() ([]byte, error) {
	result, err := app.Plan(readRepository)
	if err != nil {
		return nil, err
	}
	return cli.RenderRepositoryComparison(result.Comparison), nil
}
