package main

import (
	"fmt"

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

func prepareCheckoutSave(checkout workspace.Checkout) (func(settings.Declaration) error, error) {
	save, err := checkout.PrepareSave()
	if err != nil {
		return nil, err
	}
	return func(declaration settings.Declaration) error {
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
