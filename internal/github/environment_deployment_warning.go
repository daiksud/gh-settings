package github

import "github.com/daiksud/gh-settings/internal/settings"

func (client *EnvironmentsClient) warnDeploymentNormalization(state settings.EnvironmentState, requested settings.EnvironmentDeclaration) {
	if client.NormalizationWarning == nil || (!requested.DeploymentBranchPolicyDeclared && requested.DeploymentBranchPolicy == nil) || settings.EnvironmentIdentity(state.Policy.Name) != settings.EnvironmentIdentity(requested.Name) {
		return
	}
	record := state.DeploymentModeNormalization
	if record == (settings.EnvironmentDeploymentNormalization{}) {
		return
	}
	if _, err := state.RawDeploymentBranchPolicy(); err != nil {
		return
	}
	client.NormalizationWarning(state.Policy.Name, record)
}
