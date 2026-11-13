package cli

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/app"
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
