package settings

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// CollaboratorDeclaration binds a person, never an invitation or accepted access.
type CollaboratorDeclaration struct {
	Username string
	ID       *int64
}

type CollaboratorDeclarations struct{ entries []CollaboratorDeclaration }

var collaboratorUsername = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?$`)

func ValidateCollaboratorUsername(value string) error {
	if !collaboratorUsername.MatchString(value) || strings.Contains(value, "--") {
		return fmt.Errorf("collaborator username must be a GitHub username (email-only invitations are unsupported)")
	}
	return nil
}

func NewCollaboratorDeclarations(values []CollaboratorDeclaration) (CollaboratorDeclarations, error) {
	result := CollaboratorDeclarations{entries: make([]CollaboratorDeclaration, len(values))}
	seen, ids := map[string]bool{}, map[int64]bool{}
	for i, value := range values {
		if err := ValidateCollaboratorUsername(value.Username); err != nil {
			return CollaboratorDeclarations{}, err
		}
		key := strings.ToLower(value.Username)
		if seen[key] {
			return CollaboratorDeclarations{}, fmt.Errorf("duplicate collaborator username %q", value.Username)
		}
		seen[key] = true
		if value.ID != nil {
			id := *value.ID
			if id <= 0 || ids[id] {
				return CollaboratorDeclarations{}, fmt.Errorf("invalid or duplicate collaborator user id")
			}
			ids[id] = true
			value.ID = &id
		}
		result.entries[i] = value
	}
	slices.SortFunc(result.entries, func(a, b CollaboratorDeclaration) int {
		return strings.Compare(strings.ToLower(a.Username), strings.ToLower(b.Username))
	})
	return result, nil
}
func (d CollaboratorDeclarations) Len() int { return len(d.entries) }
func (d CollaboratorDeclarations) Values() []CollaboratorDeclaration {
	result := slices.Clone(d.entries)
	for i := range result {
		if result[i].ID != nil {
			id := *result[i].ID
			result[i].ID = &id
		}
	}
	return result
}

type CollaboratorIdentity struct {
	ID       int64
	Username string
}

func (person CollaboratorIdentity) Validate() error {
	if person.ID <= 0 {
		return fmt.Errorf("missing positive collaborator user id")
	}
	return ValidateCollaboratorUsername(person.Username)
}

// InvitationID is private observation evidence, never persisted as a YAML ID.
type CollaboratorState struct {
	Person       CollaboratorIdentity
	Status       string
	InvitationID int64
}
type CollaboratorTarget struct {
	ID, OwnerID int64
	FullName    string
	Archived    bool
}
type CollaboratorObservation struct {
	target   CollaboratorTarget
	entries  []CollaboratorState
	complete bool
}

func NewCollaboratorObservation(target CollaboratorTarget, entries []CollaboratorState) (CollaboratorObservation, error) {
	if target.ID <= 0 || target.OwnerID <= 0 || len(strings.Split(target.FullName, "/")) != 2 {
		return CollaboratorObservation{}, fmt.Errorf("incomplete collaborator target identity")
	}
	ids, names, invitations := map[int64]bool{}, map[string]bool{}, map[int64]bool{}
	for _, row := range entries {
		if err := row.Person.Validate(); err != nil {
			return CollaboratorObservation{}, err
		}
		key := strings.ToLower(row.Person.Username)
		if row.Person.ID == target.OwnerID || ids[row.Person.ID] || names[key] {
			return CollaboratorObservation{}, fmt.Errorf("ambiguous collaborator identity or repository owner")
		}
		ids[row.Person.ID], names[key] = true, true
		if row.Status == "pending" {
			if row.InvitationID <= 0 || invitations[row.InvitationID] {
				return CollaboratorObservation{}, fmt.Errorf("invalid or duplicate invitation id")
			}
			invitations[row.InvitationID] = true
		} else if row.Status != "active" || row.InvitationID != 0 {
			return CollaboratorObservation{}, fmt.Errorf("unsupported collaborator state")
		}
	}
	owned := slices.Clone(entries)
	slices.SortFunc(owned, func(a, b CollaboratorState) int {
		return strings.Compare(strings.ToLower(a.Person.Username), strings.ToLower(b.Person.Username))
	})
	return CollaboratorObservation{target: target, entries: owned, complete: true}, nil
}
func (o CollaboratorObservation) Values() ([]CollaboratorState, error) {
	if !o.complete {
		return nil, fmt.Errorf("collaborators and invitations were not completely observed")
	}
	return slices.Clone(o.entries), nil
}
func (o CollaboratorObservation) Target() (CollaboratorTarget, error) {
	if !o.complete {
		return CollaboratorTarget{}, fmt.Errorf("collaborator target was not observed")
	}
	return o.target, nil
}
func (o CollaboratorObservation) Declaration() (CollaboratorDeclarations, error) {
	rows, err := o.Values()
	if err != nil {
		return CollaboratorDeclarations{}, err
	}
	values := make([]CollaboratorDeclaration, 0, len(rows))
	for _, row := range rows {
		id := row.Person.ID
		values = append(values, CollaboratorDeclaration{Username: row.Person.Username, ID: &id})
	}
	return NewCollaboratorDeclarations(values)
}

// CollaboratorSelection retains pre-write identity and state. Bind records a
// verified user identity locally; it does not claim a remote access change.
type CollaboratorSelection struct {
	Target           CollaboratorTarget
	Person           CollaboratorIdentity
	DeclaredUsername string
	Status           string
	InvitationID     int64
	Bind             bool
}

func (s CollaboratorSelection) Changes() bool { return s.Status == "absent" || s.Bind }
func (s CollaboratorSelection) Validate() error {
	if err := s.Person.Validate(); err != nil {
		return err
	}
	if err := ValidateCollaboratorUsername(s.DeclaredUsername); err != nil {
		return err
	}
	if !strings.EqualFold(s.Person.Username, s.DeclaredUsername) {
		return fmt.Errorf("collaborator username changed; update the declaration")
	}
	if s.Status == "pending" && s.InvitationID > 0 || s.Status == "active" && s.InvitationID == 0 || s.Status == "absent" && s.InvitationID == 0 && s.Bind {
		return nil
	}
	return fmt.Errorf("invalid collaborator selection")
}

func ReconcileCollaborators(observed CollaboratorObservation, desired CollaboratorDeclarations, resolved []CollaboratorIdentity) ([]CollaboratorSelection, error) {
	rows, err := observed.Values()
	if err != nil {
		return nil, err
	}
	if len(resolved) != desired.Len() {
		return nil, fmt.Errorf("incomplete selected collaborator identities")
	}
	result := make([]CollaboratorSelection, 0, desired.Len())
	for i, declared := range desired.Values() {
		person := resolved[i]
		if err := person.Validate(); err != nil {
			return nil, err
		}
		if !strings.EqualFold(declared.Username, person.Username) {
			return nil, fmt.Errorf("collaborator %q is now %q; update the username deliberately", declared.Username, person.Username)
		}
		if declared.ID != nil && *declared.ID != person.ID {
			return nil, fmt.Errorf("collaborator %q no longer identifies user id %d", declared.Username, *declared.ID)
		}
		if person.ID == observed.target.OwnerID {
			return nil, fmt.Errorf("repository owner cannot be declared as a collaborator")
		}
		selected := CollaboratorSelection{Target: observed.target, Person: person, DeclaredUsername: declared.Username, Status: "absent", Bind: declared.ID == nil}
		for _, row := range rows {
			if strings.EqualFold(row.Person.Username, person.Username) && row.Person.ID != person.ID {
				return nil, fmt.Errorf("collaborator username belongs to another user id")
			}
			if row.Person.ID == person.ID {
				if !strings.EqualFold(row.Person.Username, person.Username) {
					return nil, fmt.Errorf("collaborator identity changed during observation")
				}
				selected.Status, selected.InvitationID = row.Status, row.InvitationID
			}
		}
		if selected.Status == "absent" && declared.ID != nil {
			return nil, fmt.Errorf("collaborator user id %d is neither active nor pending; refusing to reinvite", person.ID)
		}
		result = append(result, selected)
	}
	return result, nil
}

type CollaboratorWriteResult struct {
	Person             CollaboratorIdentity
	Status             string
	InvitationID       int64
	Attempted, Invited bool
}
