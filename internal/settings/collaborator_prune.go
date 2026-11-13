package settings

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"time"
)

// These acquisition inputs contain only the fields needed for fixed access
// deletion guards. They are copied into private snapshots, never reports/YAML.
type CollaboratorInviter struct {
	ID          int64
	Login, Type string
}
type CollaboratorPruneEntry struct {
	State       CollaboratorState
	Permissions map[string]bool
	CreatedAt   string
	Expired     *bool
	Inviter     *CollaboratorInviter
}
type CollaboratorOperator struct {
	Kind string
	User CollaboratorIdentity
}

type CollaboratorDeletion struct {
	Person       CollaboratorIdentity
	InvitationID int64
}

func (d CollaboratorDeletion) Operation() string {
	if d.InvitationID > 0 {
		return "cancel-invitation"
	}
	return "remove-collaborator"
}
func (d CollaboratorDeletion) Validate() error {
	if err := d.Person.Validate(); err != nil {
		return err
	}
	if d.InvitationID < 0 {
		return fmt.Errorf("invalid invitation deletion identity")
	}
	return nil
}

type CollaboratorDeleteResult struct {
	Deletion                          CollaboratorDeletion
	Attempted, Acknowledged, Verified bool
}

// A preparation owns the retained user union, complete inventories and operator
// evidence. Refreshes can account only for the operation explicitly supplied.
type CollaboratorPruneObservation struct {
	observed  CollaboratorObservation
	entries   []CollaboratorPruneEntry
	operator  CollaboratorOperator
	selected  []CollaboratorSelection
	deletions []CollaboratorDeletion
	prepared  bool
}

func cloneCollaboratorPruneEntry(v CollaboratorPruneEntry) CollaboratorPruneEntry {
	v.Permissions = maps.Clone(v.Permissions)
	if v.Expired != nil {
		value := *v.Expired
		v.Expired = &value
	}
	if v.Inviter != nil {
		value := *v.Inviter
		v.Inviter = &value
	}
	return v
}
func NewCollaboratorPruneObservation(observed CollaboratorObservation, entries []CollaboratorPruneEntry, operator CollaboratorOperator) (CollaboratorPruneObservation, error) {
	rows, err := observed.Values()
	if err != nil {
		return CollaboratorPruneObservation{}, err
	}
	if len(rows) != len(entries) {
		return CollaboratorPruneObservation{}, fmt.Errorf("incomplete private collaborator evidence")
	}
	owned := make([]CollaboratorPruneEntry, len(entries))
	for i, entry := range entries {
		owned[i] = cloneCollaboratorPruneEntry(entry)
	}
	slices.SortFunc(owned, func(a, b CollaboratorPruneEntry) int {
		return strings.Compare(strings.ToLower(a.State.Person.Username), strings.ToLower(b.State.Person.Username))
	})
	for i, entry := range owned {
		if entry.State != rows[i] {
			return CollaboratorPruneObservation{}, fmt.Errorf("mismatched private collaborator evidence")
		}
		if entry.State.Status == "active" {
			if entry.Permissions != nil {
				for _, key := range []string{"pull", "push", "admin"} {
					if _, ok := entry.Permissions[key]; !ok {
						return CollaboratorPruneObservation{}, fmt.Errorf("incomplete active role evidence")
					}
				}
				for key, value := range entry.Permissions {
					want := key == "pull" || key == "push" || key == "triage"
					if (key != "pull" && key != "push" && key != "admin" && key != "triage" && key != "maintain") || value != want {
						return CollaboratorPruneObservation{}, fmt.Errorf("unsupported active role evidence")
					}
				}
			}
			if entry.CreatedAt != "" || entry.Expired != nil || entry.Inviter != nil {
				return CollaboratorPruneObservation{}, fmt.Errorf("invalid active collaborator evidence")
			}
		} else {
			if _, err := time.Parse(time.RFC3339, entry.CreatedAt); err != nil || entry.Permissions != nil || entry.Expired != nil && *entry.Expired {
				return CollaboratorPruneObservation{}, fmt.Errorf("incomplete invitation evidence")
			}
			if v := entry.Inviter; v != nil {
				if v.ID <= 0 || v.Login == "" || strings.ContainsAny(v.Login, "/\\\r\n\t ") || (v.Type != "User" && v.Type != "Bot") {
					return CollaboratorPruneObservation{}, fmt.Errorf("invalid invitation inviter evidence")
				}
			}
		}
	}
	return CollaboratorPruneObservation{observed: observed, entries: owned, operator: operator}, nil
}
func (o CollaboratorPruneObservation) Prepare(desired CollaboratorDeclarations, resolved []CollaboratorIdentity) (CollaboratorPruneObservation, error) {
	selected, err := ReconcileCollaborators(o.observed, desired, resolved)
	if err != nil {
		return CollaboratorPruneObservation{}, err
	}
	keep := map[int64]bool{o.observed.target.OwnerID: true}
	for _, p := range selected {
		if keep[p.Person.ID] {
			return CollaboratorPruneObservation{}, fmt.Errorf("duplicate retained collaborator identity")
		}
		keep[p.Person.ID] = true
	}
	o.selected = selected
	o.deletions = nil
	for _, row := range o.entries {
		if !keep[row.State.Person.ID] {
			o.deletions = append(o.deletions, CollaboratorDeletion{Person: row.State.Person, InvitationID: row.State.InvitationID})
		}
	}
	slices.SortStableFunc(o.deletions, func(a, b CollaboratorDeletion) int {
		if (a.InvitationID > 0) != (b.InvitationID > 0) {
			if a.InvitationID > 0 {
				return -1
			}
			return 1
		}
		return strings.Compare(strings.ToLower(a.Person.Username), strings.ToLower(b.Person.Username))
	})
	o.prepared = true
	if len(o.deletions) > 0 {
		if err := o.validateOperator(); err != nil {
			return CollaboratorPruneObservation{}, err
		}
	}
	if err := o.checkDependencies(false); err != nil {
		return CollaboratorPruneObservation{}, err
	}
	return o, nil
}
func (o CollaboratorPruneObservation) validateOperator() error {
	switch o.operator.Kind {
	case "owner":
		if err := o.operator.User.Validate(); err != nil {
			return err
		}
		if o.operator.User.ID != o.observed.target.OwnerID || !strings.EqualFold(o.operator.User.Username, strings.Split(o.observed.target.FullName, "/")[0]) {
			return fmt.Errorf("authenticated user must be the personal repository owner")
		}
	case "installation":
		if o.operator.User != (CollaboratorIdentity{}) {
			return fmt.Errorf("ambiguous installation operator evidence")
		}
	default:
		return fmt.Errorf("collaborator deletion management path unavailable")
	}
	return nil
}
func (o CollaboratorPruneObservation) ValidateApply(unarchive bool) error {
	if !o.prepared {
		return fmt.Errorf("collaborator prune was not prepared")
	}
	if len(o.deletions) > 0 && o.observed.target.Archived && !unarchive {
		return fmt.Errorf("unarchive explicitly before collaborator deletion")
	}
	return nil
}
func (o CollaboratorPruneObservation) Selections() []CollaboratorSelection {
	return slices.Clone(o.selected)
}
func (o CollaboratorPruneObservation) Deletions() []CollaboratorDeletion {
	return slices.Clone(o.deletions)
}
func (o CollaboratorPruneObservation) NeedsOperator() bool { return len(o.deletions) > 0 }
func (o CollaboratorPruneObservation) AfterUnarchive() CollaboratorPruneObservation {
	o.observed.target.Archived = false
	o.selected = slices.Clone(o.selected)
	for i := range o.selected {
		o.selected[i].Target.Archived = false
	}
	return o
}
func (o CollaboratorPruneObservation) checkDependencies(allPending bool) error {
	removals, cancellations := map[int64]bool{}, map[int64]bool{}
	for _, d := range o.deletions {
		if d.InvitationID == 0 {
			removals[d.Person.ID] = true
		} else {
			cancellations[d.InvitationID] = true
		}
	}
	if len(removals) == 0 {
		return nil
	}
	for _, row := range o.entries {
		if row.State.Status != "pending" || !allPending && cancellations[row.State.InvitationID] {
			continue
		}
		if row.Inviter == nil || removals[row.Inviter.ID] {
			return fmt.Errorf("active collaborator deletion could cancel a retained or newly observed invitation")
		}
	}
	return nil
}

// Refresh allows selected pending-to-active progress but never a candidate state
// transition. A supplied successful reconciliation owns only that person's row.
func (o CollaboratorPruneObservation) Refresh(fresh CollaboratorPruneObservation, reconciled *CollaboratorWriteResult) (CollaboratorPruneObservation, error) {
	if !o.prepared || o.observed.target != fresh.observed.target || o.operator != fresh.operator {
		return CollaboratorPruneObservation{}, fmt.Errorf("collaborator repository or management path changed")
	}
	expected := map[int64]CollaboratorPruneEntry{}
	retained := map[int64]bool{}
	for _, p := range o.selected {
		retained[p.Person.ID] = true
	}
	for _, row := range o.entries {
		expected[row.State.Person.ID] = row
	}
	owned := int64(0)
	if reconciled != nil {
		owned = reconciled.Person.ID
		if !retained[owned] || (reconciled.Status != "active" && reconciled.Status != "pending") {
			return CollaboratorPruneObservation{}, fmt.Errorf("unverified collaborator reconciliation")
		}
	}
	foundOwned := false
	for _, row := range fresh.entries {
		id := row.State.Person.ID
		before, exists := expected[id]
		if id == owned {
			if row.State.Person != reconciled.Person || row.State.Status != reconciled.Status || row.State.Status == "pending" && row.State.InvitationID != reconciled.InvitationID {
				return CollaboratorPruneObservation{}, fmt.Errorf("reconciled collaborator changed before prune")
			}
			if !exists && !reconciled.Attempted {
				return CollaboratorPruneObservation{}, fmt.Errorf("unowned collaborator addition")
			}
			if exists && (before.State.Person != row.State.Person || before.State.Status == "active" && row.State.Status != "active") {
				return CollaboratorPruneObservation{}, fmt.Errorf("retained collaborator identity or access regressed")
			}
			if exists && before.State.Status == row.State.Status && !reflect.DeepEqual(before, row) {
				return CollaboratorPruneObservation{}, fmt.Errorf("retained collaborator evidence changed")
			}
			foundOwned = true
		} else if !exists || !reflect.DeepEqual(before, row) {
			if !exists || !retained[id] || before.State.Person != row.State.Person || before.State.Status != "pending" || row.State.Status != "active" {
				return CollaboratorPruneObservation{}, fmt.Errorf("fixed collaborator or invitation inventory changed")
			}
		}
		delete(expected, id)
	}
	if len(expected) != 0 || owned > 0 && !foundOwned {
		return CollaboratorPruneObservation{}, fmt.Errorf("prepared collaborator or invitation disappeared")
	}
	fresh.selected = slices.Clone(o.selected)
	fresh.deletions = slices.Clone(o.deletions)
	fresh.prepared = true
	for i, p := range fresh.selected {
		for _, row := range fresh.entries {
			if row.State.Person.ID == p.Person.ID {
				fresh.selected[i].Status = row.State.Status
				fresh.selected[i].InvitationID = row.State.InvitationID
			}
		}
	}
	if err := fresh.checkDependencies(false); err != nil {
		return CollaboratorPruneObservation{}, err
	}
	return fresh, nil
}
func (o CollaboratorPruneObservation) BeforeDelete(d CollaboratorDeletion) error {
	if !o.prepared || len(o.deletions) == 0 || o.deletions[0] != d {
		return fmt.Errorf("deletion is not the next fixed collaborator candidate")
	}
	if err := o.ValidateApply(false); err != nil {
		return err
	}
	if err := o.validateOperator(); err != nil {
		return err
	}
	if d.InvitationID == 0 {
		return o.checkDependencies(true)
	}
	return nil
}
func (o CollaboratorPruneObservation) AfterDelete(fresh CollaboratorPruneObservation, d CollaboratorDeletion) (CollaboratorPruneObservation, error) {
	if err := o.BeforeDelete(d); err != nil {
		return CollaboratorPruneObservation{}, err
	}
	o.entries = slices.Clone(o.entries)
	o.entries = slices.DeleteFunc(o.entries, func(v CollaboratorPruneEntry) bool { return v.State.Person.ID == d.Person.ID })
	o.deletions = slices.Clone(o.deletions[1:])
	// Reuse exact remaining-state checks. A replacement row for the deleted user
	// is not retained and therefore cannot be adopted.
	return o.Refresh(fresh, nil)
}
