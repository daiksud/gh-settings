package cli

import (
	"errors"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"

	"testing"
)

func TestEnvironmentVariablePresentationAndFailure(t *testing.T) {
	variable, _ := settings.NewVariableChange("shared", nil, "value")
	change, _ := settings.NewEnvironmentVariableChange("prod/blue", variable)
	comparison := settings.Comparison{Managed: 1, EnvironmentVariables: []settings.EnvironmentVariableChange{change}}
	want := "repository.settings.environments[\"prod/blue\"].variables: create \"SHARED\" value=\"value\"\n"
	if got := string(RenderComparison(comparison)); got != want {
		t.Fatal(got)
	}
	for _, status := range []app.ApplyOperationStatus{app.OperationFailedRead, app.OperationFailedUnknown} {
		err := applyDiagnostic(errors.New("uncertain"), app.ApplyResult{Operations: []app.ApplyOperation{{Kind: app.EnvironmentVariableUpsert, EnvironmentName: "prod/blue", VariableName: "SHARED", Status: status}}})
		want := "repository.settings.environments[\"prod/blue\"].variables[\"SHARED\"]: uncertain"
		if status == app.OperationFailedUnknown {
			want += " (write effects are unconfirmed)"
		}
		if err.Error() != want {
			t.Fatal(err)
		}
	}
}
