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

func TestCollaboratorDeletionOutputFailurePreservesProgress(t *testing.T) {
	for _, verified := range []bool{false, true} {
		result := settings.CollaboratorDeleteResult{Deletion: settings.CollaboratorDeletion{Person: settings.CollaboratorIdentity{ID: 3, Username: "person"}, InvitationID: 7}, Attempted: true, Acknowledged: true, Verified: verified}
		var stderr bytes.Buffer
		ops := Operations{Apply: func(ApplyOptions) (app.ApplyResult, error) {
			return app.ApplyResult{Outcome: app.ApplyCompleted, CollaboratorDeletions: []settings.CollaboratorDeleteResult{result}}, nil
		}}
		status := Run([]string{"apply"}, brokenWriter{}, &stderr, "dev", ops)
		want := "deletion acknowledged; resulting access unverified"
		if verified {
			want = "invitation cancelled"
		}
		if status == 0 || !strings.Contains(stderr.String(), want) || !strings.Contains(stderr.String(), "invitation id 7") {
			t.Fatal(stderr.String())
		}
	}
}
