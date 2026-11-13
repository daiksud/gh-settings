package presentation

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

func CollaboratorLine(person settings.CollaboratorSelection) string {
	state := "active access; no invitation"
	if person.Status == "pending" {
		state = "acceptance pending; no new invitation"
	}
	if person.Status == "absent" {
		state = "send initial invitation; access requires acceptance"
	}
	if person.Bind {
		state += "; record verified user ID locally"
	}
	return fmt.Sprintf("repository.settings.collaborators[%q] (user id %d): %s", person.DeclaredUsername, person.Person.ID, state)
}

func CollaboratorDeletionLine(deletion settings.CollaboratorDeletion) string {
	action := "remove collaborator grant"
	if deletion.InvitationID > 0 {
		action = fmt.Sprintf("cancel pending invitation id %d", deletion.InvitationID)
	}
	return fmt.Sprintf("repository.settings.collaborators[%q] (user id %d): %s", deletion.Person.Username, deletion.Person.ID, action)
}
