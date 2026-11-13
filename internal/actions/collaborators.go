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
