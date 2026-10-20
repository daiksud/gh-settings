package workspace

import (
	"reflect"
	"testing"
)

func TestWorkflowPermissionsStrictYAMLRoundtrip(t *testing.T) {
	for _, body := range []string{`{default_workflow_permissions: read}`, `{default_workflow_permissions: write}`, `{can_approve_pull_request_reviews: false}`, `{can_approve_pull_request_reviews: true}`, `{default_workflow_permissions: read, can_approve_pull_request_reviews: false}`} {
		d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {actions: " + body + "}"))
		if err != nil {
			t.Fatal(err)
		}
		if d.Empty() || !d.Actions.ExecutionEmpty() {
			t.Fatal("independent management lost")
		}
		data, err := EncodeConfig(d)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeConfig(data)
		if err != nil || !reflect.DeepEqual(d, again) {
			t.Fatal(string(data), err)
		}
	}
	for _, body := range []string{`{default_workflow_permissions: null}`, `{default_workflow_permissions: Read}`, `{default_workflow_permissions: unknown}`, `{default_workflow_permissions: false}`, `{can_approve_pull_request_reviews: null}`, `{can_approve_pull_request_reviews: 'false'}`, `{can_approve_pull_request_reviews: 0}`} {
		if _, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {actions: " + body + "}")); err == nil {
			t.Fatal(body)
		}
	}
}
