package settings

import "testing"

func TestCollaboratorDeclarations(t *testing.T) {
	id := int64(3)
	for _, rows := range [][]CollaboratorDeclaration{{{Username: "a", ID: &id}, {Username: "A"}}, {{Username: "a", ID: &id}, {Username: "b", ID: &id}}, {{Username: "a/b"}}, {{Username: "a@example.test"}}, {{Username: "a--b"}}, {{Username: "-a"}}, {{Username: "a-"}}, {{}}} {
		if _, err := NewCollaboratorDeclarations(rows); err == nil {
			t.Fatal("invalid declaration accepted", rows)
		}
	}
	d, err := NewCollaboratorDeclarations([]CollaboratorDeclaration{{Username: "Z", ID: &id}, {Username: "a"}})
	if err != nil {
		t.Fatal(err)
	}
	id = 9
	values := d.Values()
	*values[1].ID = 10
	if d.Values()[0].Username != "a" || *d.Values()[1].ID != 3 {
		t.Fatal("declaration does not own identities")
	}
	zero := int64(0)
	if _, err := NewCollaboratorDeclarations([]CollaboratorDeclaration{{Username: "a", ID: &zero}}); err == nil {
		t.Fatal("zero id")
	}
}
func TestCollaboratorReconciliation(t *testing.T) {
	target := CollaboratorTarget{ID: 1, OwnerID: 2, FullName: "owner/current"}
	person := CollaboratorIdentity{ID: 3, Username: "Person"}
	for _, status := range []string{"active", "pending", "absent"} {
		for _, bound := range []bool{false, true} {
			rows := []CollaboratorState{}
			if status != "absent" {
				row := CollaboratorState{Person: person, Status: status}
				if status == "pending" {
					row.InvitationID = 7
				}
				rows = append(rows, row)
			}
			observed, err := NewCollaboratorObservation(target, rows)
			if err != nil {
				t.Fatal(err)
			}
			entry := CollaboratorDeclaration{Username: "person"}
			if bound {
				entry.ID = &person.ID
			}
			desired, _ := NewCollaboratorDeclarations([]CollaboratorDeclaration{entry})
			selected, err := ReconcileCollaborators(observed, desired, []CollaboratorIdentity{person})
			if status == "absent" && bound {
				if err == nil {
					t.Fatal("reinvitation allowed")
				}
				continue
			}
			if err != nil || len(selected) != 1 || selected[0].Status != status || selected[0].Bind == bound {
				t.Fatal(selected, err)
			}
			comparison := Comparison{Managed: 1, Collaborators: selected}
			if comparison.HasChanges() == bound {
				t.Fatal("binding must be distinct from matching state", comparison)
			}
		}
	}
	observed, _ := NewCollaboratorObservation(target, nil)
	for _, person := range []CollaboratorIdentity{{ID: 2, Username: "owner"}, {ID: 3, Username: "renamed"}, {ID: 4, Username: "person"}} {
		id := int64(3)
		desired, _ := NewCollaboratorDeclarations([]CollaboratorDeclaration{{Username: "person", ID: &id}})
		if _, err := ReconcileCollaborators(observed, desired, []CollaboratorIdentity{person}); err == nil {
			t.Fatal("invalid actor accepted", person)
		}
	}
	if _, err := (CollaboratorObservation{}).Declaration(); err == nil {
		t.Fatal("missing observation exported empty")
	}
	for _, rows := range [][]CollaboratorState{{{Person: person, Status: "active"}, {Person: person, Status: "pending", InvitationID: 7}}, {{Person: person, Status: "pending"}}, {{Person: CollaboratorIdentity{ID: 2, Username: "owner"}, Status: "active"}}} {
		if _, err := NewCollaboratorObservation(target, rows); err == nil {
			t.Fatal("ambiguous snapshot", rows)
		}
	}
}
