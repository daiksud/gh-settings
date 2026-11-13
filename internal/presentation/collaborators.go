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
