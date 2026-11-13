package cli

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
)

func collaboratorResultLine(person app.CollaboratorResult) string {
	state := "outcome unconfirmed; resolve access before retrying"
	if person.Status == "pending" {
		state = "acceptance pending"
	}
	if person.Status == "active" {
		state = "active access"
	}
	if person.Invited {
		state += "; invitation sent"
	} else if !person.Attempted {
		state += "; no invitation sent"
	}
	if person.InvitationID > 0 {
		state += fmt.Sprintf("; reported invitation id %d", person.InvitationID)
	}
	if person.BindingVerified {
		if person.BindingRecorded {
			state += "; user ID recorded locally"
		} else {
			state += "; record this verified user ID before retrying"
		}
	}
	return fmt.Sprintf("Collaborator %q: user id %d; %s", person.Username, person.Person.ID, state)
}
func collaboratorDiagnostic(err error, people []app.CollaboratorResult) error {
	for _, person := range people {
		err = fmt.Errorf("%w; %s", err, collaboratorResultLine(person))
	}
	return err
}

func collaboratorDeletionResultLine(result settings.CollaboratorDeleteResult) string {
	state := "no deletion request sent"
	if result.Attempted {
		state = "deletion attempted; effects unknown"
	}
	if result.Acknowledged {
		state = "deletion acknowledged; resulting access unverified"
	}
	if result.Verified {
		state = "collaborator grant removed"
		if result.Deletion.InvitationID > 0 {
			state = "invitation cancelled"
		}
	}
	suffix := ""
	if result.Deletion.InvitationID > 0 {
		suffix = fmt.Sprintf("; invitation id %d", result.Deletion.InvitationID)
	}
	return fmt.Sprintf("Collaborator %q: user id %d; %s%s", result.Deletion.Person.Username, result.Deletion.Person.ID, state, suffix)
}
func collaboratorDeletionDiagnostic(err error, results []settings.CollaboratorDeleteResult) error {
	for _, result := range results {
		err = fmt.Errorf("%w; %s", err, collaboratorDeletionResultLine(result))
	}
	return err
}
