package github

import (
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"strings"
	"testing"
)

func TestUIDeployKeysPresenceAndValidation(t *testing.T) {
	title, key, grant := "", deployPublicOne, false
	good := uimodel.DeployKey{Title: &title, Key: &key, AllowWriteAccess: &grant}
	for _, test := range []struct {
		name string
		rows *[]uimodel.DeployKey
		bad  bool
	}{
		{name: "omitted"}, {name: "empty", rows: deployUIRows()}, {name: "explicit false and empty title", rows: deployUIRows(good)},
		{name: "missing title", rows: deployUIRows(uimodel.DeployKey{Key: &key, AllowWriteAccess: &grant}), bad: true},
		{name: "missing grant", rows: deployUIRows(uimodel.DeployKey{Title: &title, Key: &key}), bad: true},
		{name: "missing material", rows: deployUIRows(uimodel.DeployKey{Title: &title, AllowWriteAccess: &grant}), bad: true},
		{name: "duplicate material", rows: deployUIRows(good, good), bad: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			d, err := DeclarationFromUI(uimodel.Repository{Settings: &uimodel.RepositorySettings{DeployKeys: test.rows}})
			if (err != nil) != test.bad {
				t.Fatal(err)
			}
			if test.bad {
				if !strings.Contains(err.Error(), "repository.settings.deploy_keys") {
					t.Fatal("missing bounded resource", err)
				}
				return
			}
			if (d.DeployKeys != nil) != (test.rows != nil) {
				t.Fatal("presence lost")
			}
			ui, err := UIFromExportDeclaration(d)
			if err != nil {
				t.Fatal(err)
			}
			if test.rows != nil && (ui.Settings == nil || ui.Settings.DeployKeys == nil || len(*ui.Settings.DeployKeys) != len(*test.rows)) {
				t.Fatal("export presence lost")
			}
		})
	}
}
func deployUIRows(rows ...uimodel.DeployKey) *[]uimodel.DeployKey {
	if rows == nil {
		rows = []uimodel.DeployKey{}
	}
	return &rows
}
