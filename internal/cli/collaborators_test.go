package cli

import (
	"bytes"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
	"strings"
	"testing"
)

func TestCollaboratorOutputFailurePreservesBindingAndPending(t *testing.T) {
	var stderr bytes.Buffer
	calls := 0
	code := Run([]string{"apply"}, brokenWriter{}, &stderr, "dev", Operations{Apply: func(ApplyOptions) (app.ApplyResult, error) {
		calls++
		return app.ApplyResult{Outcome: app.ApplyCompleted, Collaborators: []app.CollaboratorResult{{Username: "person", CollaboratorWriteResult: settings.CollaboratorWriteResult{Person: settings.CollaboratorIdentity{ID: 3, Username: "person"}, Status: "pending", InvitationID: 7, Attempted: true, Invited: true}, BindingVerified: true, BindingRecorded: true}}}, nil
	}})
	if code != 1 || calls != 1 {
		t.Fatal(code, calls)
	}
	for _, want := range []string{"write output: output unavailable", "user id 3", "acceptance pending", "reported invitation id 7", "user ID recorded locally"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatal(stderr.String())
		}
	}
	if strings.Contains(stderr.String(), "Created") {
		t.Fatal("binding described as user creation")
	}
}
