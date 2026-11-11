package settings

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestPullRequestExemptionDeclarations(t *testing.T) {
	input := []string{"Z-user", "Alice"}
	d, err := NewPullRequestExemptionsDeclaration(input)
	if err != nil || !reflect.DeepEqual(d.Logins(), []string{"alice", "z-user"}) {
		t.Fatalf("canonical %#v %v", d, err)
	}
	input[0] = "changed"
	values := d.Logins()
	values[0] = "changed"
	if d.Logins()[0] != "alice" {
		t.Fatal("declaration aliases caller")
	}
	for _, names := range [][]string{{""}, {"alice", "ALICE"}, {"-a"}, {"a-"}, {"a--b"}, {"a_b"}, {"app[bot]"}, {" Alice"}, {"日本語"}, {"K"}, {strings.Repeat("a", 38) + "K"}, {strings.Repeat("a", 40)}} {
		if _, err := NewPullRequestExemptionsDeclaration(names); err == nil {
			t.Fatalf("accepted %#v", names)
		}
	}
	names := make([]string, 100)
	for i := range names {
		names[i] = fmt.Sprintf("user%d", i)
	}
	if _, err := NewPullRequestExemptionsDeclaration(names); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPullRequestExemptionsDeclaration(append(names, "extra")); err == nil {
		t.Fatal("accepted >100")
	}
	empty, _ := NewPullRequestExemptionsDeclaration(nil)
	if empty.Logins() == nil || (Declaration{PullRequestExemptions: &empty}).Empty() {
		t.Fatal("explicit empty lost ownership")
	}
}

func TestPullRequestExemptionObservation(t *testing.T) {
	if _, err := (PullRequestExemptionsObservation{}).Users(); err == nil {
		t.Fatal("accepted missing acquisition")
	}
	for _, users := range [][]PullRequestExemptionUser{
		{{ID: 0, Login: "alice", Type: "User"}}, {{ID: 1, Login: "alice", Type: "Organization"}}, {{ID: 1, Login: "app", Type: "Bot"}},
		{{ID: 1, Login: "alice", Type: "User"}, {ID: 1, Login: "bob", Type: "User"}},
		{{ID: 1, Login: "alice", Type: "User"}, {ID: 2, Login: "ALICE", Type: "User"}},
	} {
		if _, err := NewPullRequestExemptionsObservation(users); err == nil {
			t.Fatalf("accepted %#v", users)
		}
	}
	input := []PullRequestExemptionUser{{ID: 2, Login: "Bob", Type: "User"}, {ID: 1, Login: "Alice", Type: "User"}}
	o, err := NewPullRequestExemptionsObservation(input)
	if err != nil {
		t.Fatal(err)
	}
	input[0].ID = 0
	users, _ := o.Users()
	users[0].ID = 0
	users, _ = o.Users()
	if users[0].ID != 1 || users[0].Login != "alice" {
		t.Fatal("observation alias or canonical identity lost", users)
	}
}

func TestComparePullRequestExemptions(t *testing.T) {
	if n, c, err := ComparePullRequestExemptions(PullRequestExemptionsObservation{}, nil); err != nil || n != 0 || len(c) != 0 {
		t.Fatal("omission acquired state")
	}
	d, _ := NewPullRequestExemptionsDeclaration([]string{"alice"})
	if _, _, err := ComparePullRequestExemptions(PullRequestExemptionsObservation{}, &d); err == nil {
		t.Fatal("missing acquisition")
	}
	o, _ := NewPullRequestExemptionsObservation([]PullRequestExemptionUser{{ID: 1, Login: "ALICE", Type: "User"}})
	n, c, err := ComparePullRequestExemptions(o, &d)
	if err != nil || n != 1 || len(c) != 0 {
		t.Fatalf("set match %d %#v %v", n, c, err)
	}
	empty, _ := NewPullRequestExemptionsDeclaration([]string{})
	n, c, err = ComparePullRequestExemptions(o, &empty)
	if err != nil || n != 1 || len(c) != 1 {
		t.Fatalf("clear %d %#v %v", n, c, err)
	}
	before, after := c[0].Values()
	if !reflect.DeepEqual(before, []string{"alice"}) || after == nil || len(after) != 0 || c[0].Field() != PullRequestExemptionsPath {
		t.Fatal(before, after)
	}
	before[0] = "changed"
	before, _ = c[0].Values()
	if before[0] != "alice" {
		t.Fatal("change alias")
	}
	if !(Comparison{PullRequestExemptions: c}).HasChanges() {
		t.Fatal("lost change")
	}
	if _, err := NewPullRequestExemptionsChange([]string{"Alice"}, []string{"alice"}); err == nil {
		t.Fatal("accepted unchanged")
	}
}
