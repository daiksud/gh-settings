package settings

import (
	"encoding/json"
	"testing"
)

func modelAutolink(id int64, prefix string) Autolink {
	v := Autolink{ReferencePrefix: prefix, TargetURL: "https://example.test/<num>?Case=A%2FB", ReferenceFormat: "numeric"}
	if id > 0 {
		v.ID = &id
	}
	return v
}
func modelAutolinks() AutolinkObservation {
	return AutolinkObservation{Target: AutolinkTarget{ID: 1, OwnerID: 2, FullName: "owner/current"}, Entries: []AutolinkState{{Value: modelAutolink(7, "OLD-")}}}
}
func TestAutolinkValidationAndImmutableSelection(t *testing.T) {
	good := modelAutolink(0, "NEW-")
	for _, v := range []Autolink{good, modelAutolink(7, "OLD-")} {
		if err := ValidateAutolinks([]Autolink{v}); err != nil {
			t.Fatal(err)
		}
	}
	for _, mode := range []string{"empty-prefix", "space-prefix", "format", "no-placeholder", "relative", "scheme", "userinfo", "control", "zero", "overlap", "duplicate-id"} {
		t.Run(mode, func(t *testing.T) {
			v := good
			values := []Autolink{}
			switch mode {
			case "empty-prefix":
				v.ReferencePrefix = ""
			case "space-prefix":
				v.ReferencePrefix = " "
			case "format":
				v.ReferenceFormat = "false"
			case "no-placeholder":
				v.TargetURL = "https://example.test/id"
			case "relative":
				v.TargetURL = "/<num>"
			case "scheme":
				v.TargetURL = "file://host/<num>"
			case "userinfo":
				v.TargetURL = "https://name:private@example.test/<num>"
			case "control":
				v.TargetURL += "\n"
			case "zero":
				n := int64(0)
				v.ID = &n
			case "overlap":
				values = append(values, modelAutolink(8, "NEW"))
			case "duplicate-id":
				v = modelAutolink(7, "NEW-")
				values = append(values, modelAutolink(7, "OLD-"))
			}
			values = append(values, v)
			if ValidateAutolinks(values) == nil {
				t.Fatal("invalid accepted")
			}
		})
	}
	o := modelAutolinks()
	same := o.Entries[0].Value
	if c, e := CompareAutolinks(o, []Autolink{same}, true); e != nil || len(c) != 0 {
		t.Fatal(c, e)
	}
	changed := same
	changed.TargetURL += "changed"
	missing := modelAutolink(9, "MISS-")
	for _, v := range []Autolink{changed, missing, modelAutolink(0, "OLD-")} {
		if _, e := CompareAutolinks(o, []Autolink{v}, true); e == nil {
			t.Fatal("unsafe bound or colliding intent")
		}
	}
	c, e := CompareAutolinks(o, []Autolink{good}, true)
	if e != nil || len(c) != 2 || c[0].After == nil || c[1].Before == nil || ValidateAutolinkChanges(c) != nil {
		t.Fatal(c, e)
	}
	if c, e := CompareAutolinks(o, []Autolink{}, false); e != nil || len(c) != 0 {
		t.Fatal(c, e)
	}
}
func TestAutolinkPrivateInventoryAndOwnTransitions(t *testing.T) {
	o := modelAutolinks()
	for _, mode := range []string{"missing", "added", "changed", "metadata-null", "target", "owner", "archived"} {
		t.Run(mode, func(t *testing.T) {
			next := o.Clone()
			switch mode {
			case "missing":
				next.Entries = []AutolinkState{}
			case "added":
				next.Entries = append(next.Entries, AutolinkState{Value: modelAutolink(8, "NEW-")})
			case "changed":
				next.Entries[0].Value.TargetURL += "different"
			case "metadata-null":
				next.Entries[0].UpdatedAtPresent = true
			case "target":
				next.Target.ID++
			case "owner":
				next.Target.OwnerID++
			case "archived":
				next.Target.Archived = true
			}
			if VerifyAutolinkInventory(o, next) == nil {
				t.Fatal("drift accepted")
			}
		})
	}
	next, err := o.WithCreated(AutolinkState{Value: modelAutolink(8, "NEW-")})
	if err != nil {
		t.Fatal(err)
	}
	if err = VerifyAutolinkInventory(o, next.Without(8)); err != nil {
		t.Fatal(err)
	}
	values, err := ExportAutolinks(o)
	if err != nil || values[0].TargetURL != o.Entries[0].Value.TargetURL {
		t.Fatal(values, err)
	}
	*values[0].ID = 999
	if *o.Entries[0].Value.ID != 7 {
		t.Fatal("export aliases identity")
	}
	empty := o.Clone()
	empty.Entries = []AutolinkState{}
	values, err = ExportAutolinks(empty)
	if err != nil || values == nil || len(values) != 0 {
		t.Fatal(values, err)
	}
}
func TestAutolinkReportRejectsUnknownAndPrivateMetadata(t *testing.T) {
	for _, raw := range []string{`{"Before":{}}`, `{"before":null}`, `{"before":{"id":7,"reference_prefix":"A","target_url":"https://x/<num>","reference_format":"numeric","updated_at":null}}`, `{"after":{"ID":7}}`} {
		var c AutolinkChange
		if json.Unmarshal([]byte(raw), &c) == nil {
			t.Fatal("accepted", raw)
		}
	}
}
