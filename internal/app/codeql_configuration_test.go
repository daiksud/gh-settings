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
	o, _ := settings.NewCodeQLConfigurationObservation(settings.CodeQLConfigurationState{Languages: []string{"go"}, QuerySuite: "default", Runner: settings.CodeQLRunner{Type: "standard_github_runner"}, ThreatModel: "remote_sources"})
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

func TestCodeQLThreatModelApplyFailureStopsLaterSetup(t *testing.T) {
	threat, enabled := "remote_and_local_sources", true
	d := settings.Declaration{Security: settings.SecurityDeclaration{CodeQLDefaultSetup: &enabled, CodeQLConfiguration: settings.CodeQLConfigurationDeclaration{ThreatModel: &threat}}}
	o, err := settings.NewCodeQLConfigurationObservation(settings.CodeQLConfigurationState{Languages: []string{"csharp"}, QuerySuite: "default", Runner: settings.CodeQLRunner{Type: "standard_github_runner"}, ThreatModel: "remote_sources"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := Plan(func() (settings.Declaration, error) {
		return settings.Declaration{Security: settings.SecurityDeclaration{CodeQLConfiguration: d.Security.CodeQLConfiguration}}, nil
	}, checkoutRepository, Reader{CodeQLConfiguration: func(repository.Repository) (settings.CodeQLConfigurationObservation, error) { return o, nil }})
	if err != nil || p.Comparison.Managed != 1 || len(p.Comparison.CodeQLConfiguration) != 1 || p.Comparison.CodeQLConfiguration[0].Field() != "threat_model" {
		t.Fatal(p, err)
	}
	called := false
	r, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{
		ReadCodeQLConfiguration: func(repository.Repository) (settings.CodeQLConfigurationObservation, error) { return o, nil },
		CodeQLConfiguration: func(_ repository.Repository, got settings.CodeQLConfigurationDeclaration) (bool, error) {
			called = true
			if got.ThreatModel == nil || *got.ThreatModel != threat || got.Languages != nil || got.QuerySuite != nil || got.Runner != nil {
				t.Fatal(got)
			}
			return false, errors.New("verification failed after PATCH")
		},
		ReadCodeQLDefaultSetup: func(repository.Repository) (settings.CodeQLDefaultSetupObservation, error) {
			t.Fatal("continued after uncertain write")
			return settings.CodeQLDefaultSetupObservation{}, nil
		},
	})
	if err == nil || !called || len(r.Operations) != 2 || r.Operations[1].Kind != CodeQLConfigurationUpdate || r.Operations[1].Status != OperationFailedUnknown {
		t.Fatal(r, err)
	}
}
