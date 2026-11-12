package main

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func loadCheckoutDeclaration(checkout workspace.Checkout) (settings.Declaration, error) {
	ui, err := checkout.Declaration()
	if err != nil {
		return settings.Declaration{}, err
	}
	declaration, err := github.DeclarationFromUI(ui)
	if err != nil {
		return settings.Declaration{}, fmt.Errorf("validate %s: %w", workspace.SettingsPath, err)
	}
	return declaration, nil
}

func prepareCheckoutSave(checkout workspace.Checkout) (settings.Declaration, func(settings.Declaration) error, error) {
	previous, save, err := checkout.PrepareSave()
	if err != nil {
		return settings.Declaration{}, nil, err
	}
	existing, err := app.Validate(func() (settings.Declaration, error) { return github.DeclarationFromUI(previous) })
	if err != nil {
		return settings.Declaration{}, nil, fmt.Errorf("existing %s cannot be preserved safely", workspace.SettingsPath)
	}
	return existing, func(declaration settings.Declaration) error {
		ui, err := github.UIFromExportDeclaration(declaration)
		if err != nil {
			return err
		}
		return save(ui)
	}, nil
}

func prepareCheckoutApply(checkout workspace.Checkout) (settings.Declaration, func(string, string, int64) error, error) {
	ui, record, err := checkout.PrepareApply()
	if err != nil {
		return settings.Declaration{}, nil, err
	}
	declaration, err := github.DeclarationFromUI(ui)
	if err != nil {
		return settings.Declaration{}, nil, fmt.Errorf("validate %s: %w", workspace.SettingsPath, err)
	}
	return declaration, record, nil
}
