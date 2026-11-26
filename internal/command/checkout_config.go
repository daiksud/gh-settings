package command

import (
	"fmt"
	"io"
	"strings"

	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func loadCheckoutDeclaration(checkout workspace.Checkout, diagnostics io.Writer) (settings.Declaration, error) {
	ui, err := checkout.Declaration()
	if err != nil {
		return settings.Declaration{}, err
	}
	return checkoutIntent(ui, diagnostics)
}

func checkoutIntent(ui uimodel.Repository, diagnostics io.Writer) (settings.Declaration, error) {
	declaration, err := github.DeclarationFromUI(ui)
	if err != nil {
		return settings.Declaration{}, fmt.Errorf("validate %s: %w", workspace.SettingsPath, err)
	}
	if err := reportIgnoredPullRequestLimitKeys(diagnostics, ui); err != nil {
		return settings.Declaration{}, err
	}
	return declaration, nil
}

func reportIgnoredPullRequestLimitKeys(output io.Writer, ui uimodel.Repository) error {
	if ui.Settings == nil || len(ui.Settings.IgnoredPullRequestLimitKeys) == 0 {
		return nil
	}
	_, err := fmt.Fprintf(output, "info: ignored %s because %s is false\n", strings.Join(ui.Settings.IgnoredPullRequestLimitKeys, ", "), settings.PullRequestLimitEnabledPath)
	if err != nil {
		return fmt.Errorf("write ignored pull request limits info: %w", err)
	}
	return nil
}

func prepareCheckoutSave(checkout workspace.Checkout, diagnostics io.Writer) (settings.Declaration, func(settings.Declaration) error, error) {
	previous, save, err := checkout.PrepareSave()
	if err != nil {
		return settings.Declaration{}, nil, err
	}
	existing, err := app.Validate(func() (settings.Declaration, error) { return github.DeclarationFromUI(previous) })
	if err != nil {
		return settings.Declaration{}, nil, fmt.Errorf("existing %s cannot be preserved safely", workspace.SettingsPath)
	}
	if err := reportIgnoredPullRequestLimitKeys(diagnostics, previous); err != nil {
		return settings.Declaration{}, nil, err
	}
	return existing, func(declaration settings.Declaration) error {
		ui, err := github.UIFromExportDeclaration(declaration)
		if err != nil {
			return err
		}
		return save(ui)
	}, nil
}

func prepareCheckoutApply(checkout workspace.Checkout, diagnostics io.Writer) (settings.Declaration, func(string, string, int64) error, error) {
	ui, record, err := checkout.PrepareApply()
	if err != nil {
		return settings.Declaration{}, nil, err
	}
	declaration, err := checkoutIntent(ui, diagnostics)
	if err != nil {
		return settings.Declaration{}, nil, err
	}
	return declaration, record, nil
}
