package cli

import (
	"bytes"
	"errors"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"strings"
	"testing"
)

func TestEnvironmentVariableExportOptions(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want []settings.EnvironmentVariableSelection
		fail bool
	}{
		{args: []string{"export"}},
		{args: []string{"export", "--environment-variable", "prod/blue", "shared", "--environment-variable", "staging", "shared", "--environments"}, want: []settings.EnvironmentVariableSelection{{Environment: "prod/blue", Name: "SHARED"}, {Environment: "staging", Name: "SHARED"}}},
		{args: []string{"export", "--environment-variable"}, fail: true},
		{args: []string{"export", "--environment-variable", "prod"}, fail: true},
		{args: []string{"export", "--environment-variable", "prod", "a", "--environment-variable", "PROD", "A"}, fail: true},
		{args: []string{"export", "--environment-variable", " prod", "A"}, fail: true},
		{args: []string{"export", "--environment-variable", "prod", "GITHUB_A"}, fail: true},
		{args: []string{"export", "--environment-variables-all"}, fail: true},
	} {
		var out, diag bytes.Buffer
		calls := 0
		code := Run(tc.args, &out, &diag, "dev", Operations{Export: func(o ExportOptions) error {
			calls++
			if !reflect.DeepEqual(o.EnvironmentVariables, tc.want) {
				t.Fatal(o)
			}
			return nil
		}})
		if (code != 0) != tc.fail || calls != map[bool]int{false: 1, true: 0}[tc.fail] {
			t.Fatal(tc.args, code, calls, diag.String())
		}
	}
	var out, diag bytes.Buffer
	if Run([]string{"--help"}, &out, &diag, "dev", Operations{}) != 0 || !strings.Contains(out.String(), "--environment-variable ENVIRONMENT NAME") {
		t.Fatal(out.String(), diag.String())
	}
}

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
		if err.Error() != "repository.settings.environments[\"prod/blue\"].variables[\"SHARED\"]: uncertain" {
			t.Fatal(err)
		}
	}
}
