package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"testing"
)

func TestCodeQLConfigurationSelectivePlanApplyAndDisabledPrerequisite(t *testing.T) {
	suite := "extended"
	d := settings.Declaration{Security: settings.SecurityDeclaration{CodeQLConfiguration: settings.CodeQLConfigurationDeclaration{QuerySuite: &suite}}}
	o, _ := settings.NewCodeQLConfigurationObservation(settings.CodeQLConfigurationState{Languages: []string{"go"}, QuerySuite: "default", Runner: settings.CodeQLRunner{Type: "standard_github_runner"}, RawThreatModel: "remote"})
	p, err := Plan(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Reader{CodeQLConfiguration: func(repository.Repository) (settings.CodeQLConfigurationObservation, error) { return o, nil }})
	if err != nil || p.Comparison.Managed != 1 || len(p.Comparison.CodeQLConfiguration) != 1 {
		t.Fatal(p, err)
	}
	r, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{ReadCodeQLConfiguration: func(repository.Repository) (settings.CodeQLConfigurationObservation, error) { return o, nil }, CodeQLConfiguration: func(repository.Repository, settings.CodeQLConfigurationDeclaration) (bool, error) { return true, nil }})
	if err != nil || !r.Completed() {
		t.Fatal(r, err)
	}
	enabled := true
	d.Security.CodeQLDefaultSetup = &enabled
	r, err = Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{ReadCodeQLConfiguration: func(repository.Repository) (settings.CodeQLConfigurationObservation, error) {
		return settings.CodeQLConfigurationObservation{}, errors.New("already configured default setup required")
	}})
	if err == nil || r.Outcome != ApplyFailed || len(r.Operations) != 1 || r.Operations[0].Kind != CodeQLConfigurationRead {
		t.Fatal("implicitly enabled prerequisite", r, err)
	}
}
