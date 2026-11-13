package github

import (
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// ReadForExport observes state and details from one response. A known state is
// returned even when configured details cannot be decoded; errors never invent
// a disabled state. Strict plan/apply readers retain their separate contracts.
func (c *CodeQLDefaultSetupClient) ReadForExport(repo repository.Repository) (settings.CodeQLDefaultSetupObservation, settings.CodeQLConfigurationObservation, error) {
	var configuration settings.CodeQLConfigurationObservation
	if err := readPersonalPublicSecurityTarget(c.rest, repo); err != nil {
		return settings.CodeQLDefaultSetupObservation{}, configuration, err
	}
	var raw map[string]json.RawMessage
	if err := c.rest.Get(codeQLDefaultSetupPath(repo), &raw); err != nil {
		return settings.CodeQLDefaultSetupObservation{}, configuration, fmt.Errorf("read CodeQL export: %w", err)
	}
	setup, err := decodeCodeQLDefaultSetup(raw)
	if err != nil {
		return setup, configuration, err
	}
	enabled, _ := setup.Value()
	if !enabled {
		return setup, configuration, nil
	}
	configuration, err = decodeCodeQLConfiguration(raw)
	return setup, configuration, err
}
