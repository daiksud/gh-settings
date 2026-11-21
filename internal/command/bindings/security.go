package bindings

import (
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func readSecretScanning(repo repository.Repository) (settings.SecretScanningObservation, error) {
	client, err := github.NewSecretScanningClient(repo.Host)
	if err != nil {
		return settings.SecretScanningObservation{}, err
	}
	return client.Read(repo)
}

func readSecretScanningForApply(repo repository.Repository, desired settings.SecurityDeclaration) (settings.SecretScanningObservation, error) {
	observed, err := readSecretScanning(repo)
	if err != nil {
		return settings.SecretScanningObservation{}, err
	}
	if _, _, err := observed.Values(); err != nil {
		return settings.SecretScanningObservation{}, err
	}
	for _, record := range observed.Normalizations() {
		if record.Path == settings.SecretProtectionControl.Path() && desired.SecretScanning != nil || record.Path == settings.PushProtectionControl.Path() && desired.PushProtection != nil {
			warnSecurityStatusNormalization(record)
		}
	}
	return observed, nil
}

func writeSecretScanning(repo repository.Repository, change settings.SecurityChange) (bool, error) {
	client, err := github.NewSecretScanningClient(repo.Host)
	if err != nil {
		return false, err
	}
	client.NormalizationWarning = warnSecurityStatusNormalization
	return client.Update(repo, change)
}

func readCodeQLForExport(repo repository.Repository) (settings.CodeQLDefaultSetupObservation, settings.CodeQLConfigurationObservation, error) {
	client, err := github.NewCodeQLDefaultSetupClient(repo.Host)
	if err != nil {
		return settings.CodeQLDefaultSetupObservation{}, settings.CodeQLConfigurationObservation{}, err
	}
	return client.ReadForExport(repo)
}

func dependabotClient(repo repository.Repository) (*github.DependabotClient, error) {
	c, err := github.NewDependabotClient(repo.Host)
	if err != nil {
		return nil, err
	}
	c.Warning = func(message string) { fmt.Fprintln(os.Stderr, "Warning: "+message) }
	return c, nil
}

func readDependabot(repo repository.Repository) (settings.DependabotObservation, error) {
	c, err := dependabotClient(repo)
	if err != nil {
		return settings.DependabotObservation{}, err
	}
	return c.Read(repo)
}

func writeDependabot(repo repository.Repository, change settings.SecurityChange) (settings.DependabotWriteResult, error) {
	c, err := dependabotClient(repo)
	if err != nil {
		return settings.DependabotWriteResult{}, err
	}
	return c.Update(repo, change)
}
