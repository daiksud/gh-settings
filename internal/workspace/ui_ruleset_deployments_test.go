package workspace

import (
	"reflect"
	"testing"
)

func TestUIRulesetDeploymentsPresenceAndRoundtrip(t *testing.T) {
	const prefix = "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    rulesets:\n      - ruleset_name: staging\n        rules:\n          require_deployments_to_succeed: "
	for _, value := range []string{`{required_deployment_environments: [staging, Production]}`, `{required_deployment_environments: []}`} {
		got, err := DecodeConfig([]byte(prefix + value + "\n"))
		if err != nil {
			t.Fatal(err)
		}
		if (*got.Settings.Rulesets)[0].Rules.RequireDeploymentsToSucceed.RequiredDeploymentEnvironments == nil {
			t.Fatal("list lost")
		}
		encoded, err := EncodeConfig(got)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeConfig(encoded)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("roundtrip lost", err)
		}
	}
	for _, value := range []string{`null`, `false`, `{required_deployment_environments: null}`, `{required_deployment_environments: [null]}`, `{required_deployment_environments: [42]}`, `{required_deployment_environments: [staging], future: true}`} {
		if _, err := DecodeConfig([]byte(prefix + value + "\n")); err == nil {
			t.Fatal("accepted invalid control", value)
		}
	}
}
