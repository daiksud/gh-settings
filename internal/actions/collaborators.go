package actions

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"strings"
)

type CollaboratorPlanEntry struct {
	Username     string `json:"username"`
	ID           int64  `json:"id"`
	State        string `json:"state"`
	InvitationID int64  `json:"invitation_id,omitempty"`
	BindIdentity *bool  `json:"bind_identity"`
}

func validateCollaboratorReport(rows []CollaboratorPlanEntry) ([]settings.CollaboratorSelection, error) {
	result := make([]settings.CollaboratorSelection, 0, len(rows))
	ids, names := map[int64]bool{}, map[string]bool{}
	for _, row := range rows {
		if row.BindIdentity == nil {
			return nil, fmt.Errorf("missing collaborator binding decision")
		}
		person := settings.CollaboratorSelection{Person: settings.CollaboratorIdentity{ID: row.ID, Username: row.Username}, DeclaredUsername: row.Username, Status: row.State, InvitationID: row.InvitationID, Bind: *row.BindIdentity}
		if err := person.Validate(); err != nil {
			return nil, err
		}
		key := strings.ToLower(row.Username)
		if ids[row.ID] || names[key] {
			return nil, fmt.Errorf("duplicate collaborator report identity")
		}
		ids[row.ID], names[key] = true, true
		result = append(result, person)
	}
	return result, nil
}

type CollaboratorDeletionEntry struct {
	Operation    string `json:"operation"`
	Username     string `json:"username"`
	ID           int64  `json:"id"`
	InvitationID int64  `json:"invitation_id,omitempty"`
}

func validateCollaboratorDeletions(rows []CollaboratorDeletionEntry, selected []settings.CollaboratorSelection) ([]settings.CollaboratorDeletion, error) {
	result := make([]settings.CollaboratorDeletion, 0, len(rows))
	ids, names, invites := map[int64]bool{}, map[string]bool{}, map[int64]bool{}
	for _, p := range selected {
		ids[p.Person.ID] = true
		names[strings.ToLower(p.Person.Username)] = true
		if p.InvitationID > 0 {
			invites[p.InvitationID] = true
		}
	}
	for _, row := range rows {
		d := settings.CollaboratorDeletion{Person: settings.CollaboratorIdentity{ID: row.ID, Username: row.Username}, InvitationID: row.InvitationID}
		if err := d.Validate(); err != nil {
			return nil, err
		}
		key := strings.ToLower(row.Username)
		if d.Operation() != row.Operation || ids[row.ID] || names[key] || row.InvitationID > 0 && invites[row.InvitationID] {
			return nil, fmt.Errorf("invalid or duplicate collaborator deletion report")
		}
		ids[row.ID], names[key] = true, true
		if row.InvitationID > 0 {
			invites[row.InvitationID] = true
		}
		result = append(result, d)
	}
	return result, nil
}
