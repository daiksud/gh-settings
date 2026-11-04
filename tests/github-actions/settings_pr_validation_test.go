package githubactions

import (
	"reflect"
	"testing"
)

// The normal pull_request event checks out GitHub's merge ref by default.
// Do not add an explicit ref or unsafe checkout inputs.
func TestSettingsPRUsesDefaultMergeCheckout(t *testing.T) {
	w, _ := readWorkflow(t, "../../.github/workflows/settings-pr.yml")
	j := w.Jobs["settings-pr"]
	if w.Name != "settings-pr" || len(j.Steps) != 2 ||
		j.Steps[0].Uses != "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1" ||
		len(j.Steps[0].With) != 0 || j.Steps[0].If != "" ||
		j.Steps[1].Uses != operationalAction ||
		!reflect.DeepEqual(j.Steps[1].With, map[string]string{"command": "validate"}) {
		t.Fatal("settings PR must use a default checkout and pinned validate-only Action")
	}
}
