package settings

import (
	"strings"
	"testing"
)

const deployKeyFixtureOne = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB"
const deployKeyFixtureTwo = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIC"

func deployKeyModelState(t *testing.T, id int64, input string) DeployKeyState {
	t.Helper()
	key, err := ParseDeployPublicKey(input)
	if err != nil {
		t.Fatal(err)
	}
	return DeployKeyState{ID: id, Title: "same title", Key: key, ReadOnly: true, CreatedAt: "2026-11-14T00:00:00Z", APIURL: "https://api.github.com/repos/owner/repo/keys/1"}
}
func TestDeployKeyPublicIdentity(t *testing.T) {
	canonical, err := ParseDeployPublicKey(deployKeyFixtureOne)
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{deployKeyFixtureOne, " \t" + deployKeyFixtureOne + " comment\r\n", deployKeyFixtureOne + " another comment\n"} {
		key, err := ParseDeployPublicKey(input)
		if err != nil || key != canonical || key.Canonical() != deployKeyFixtureOne || !strings.HasPrefix(key.Fingerprint(), "SHA256:") {
			t.Fatal("canonical identity", err)
		}
	}
	for _, input := range []string{"", "PRIVATE_CANARY", "-----BEGIN OPENSSH PRIVATE KEY-----\nPRIVATE_CANARY\n-----END OPENSSH PRIVATE KEY-----", "PRIVATE_CANARY\n" + deployKeyFixtureOne, deployKeyFixtureOne + "\n" + deployKeyFixtureTwo, "command=\"PRIVATE_CANARY\" " + deployKeyFixtureOne, "ssh-rsa " + strings.TrimPrefix(deployKeyFixtureOne, "ssh-ed25519 "), deployKeyFixtureOne + "\n\n", strings.Repeat("X", DeployPublicKeyLimit+1)} {
		_, err := ParseDeployPublicKey(input)
		if err == nil || strings.Contains(err.Error(), "PRIVATE_CANARY") {
			t.Fatal("unsafe invalid input handling")
		}
	}
}
func TestDeployKeyImmutableSelectionAndMembership(t *testing.T) {
	first, second := deployKeyModelState(t, 1, deployKeyFixtureOne), deployKeyModelState(t, 2, deployKeyFixtureTwo)
	observed, err := NewDeployKeyObservation(DeployKeyTarget{ID: 1, OwnerID: 2, FullName: "owner/repo"}, []DeployKeyState{first, second})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name             string
		values           []DeployKeyDeclaration
		prune            bool
		creates, deletes int
		bad              bool
	}{
		{name: "matching", values: []DeployKeyDeclaration{first.Declaration(), second.Declaration()}},
		{name: "ordinary empty"}, {name: "prune empty", prune: true, deletes: 2},
		{name: "retain selected", values: []DeployKeyDeclaration{second.Declaration()}, prune: true, deletes: 1},
		{name: "idless collision", values: []DeployKeyDeclaration{{Title: first.Title, Key: first.Key}}, bad: true},
		{name: "missing id", values: []DeployKeyDeclaration{{ID: deployTestID(3), Title: first.Title, Key: first.Key}}, bad: true},
		{name: "changed grant", values: []DeployKeyDeclaration{{ID: deployTestID(1), Title: first.Title, Key: first.Key, AllowWriteAccess: true}}, bad: true},
		{name: "changed title", values: []DeployKeyDeclaration{{ID: deployTestID(1), Title: "renamed", Key: first.Key}}, bad: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, e := NewDeployKeyDeclarations(tc.values)
			if e != nil {
				t.Fatal(e)
			}
			_, changes, e := PrepareDeployKeys(observed, &d, tc.prune)
			if (e != nil) != tc.bad {
				t.Fatal(e)
			}
			c, n := 0, 0
			for _, x := range changes {
				if x.Operation() == "create" {
					c++
				} else {
					n++
				}
			}
			if c != tc.creates || n != tc.deletes {
				t.Fatal(c, n)
			}
		})
	}
	for _, d := range [][]DeployKeyDeclaration{{first.Declaration(), first.Declaration()}, {first.Declaration(), {Title: "other", Key: first.Key}}} {
		if _, err := NewDeployKeyDeclarations(d); err == nil {
			t.Fatal("duplicate identity")
		}
	}
	// Same titles are valid; matching key comments never produce a replacement.
	if _, err := NewDeployKeyDeclarations([]DeployKeyDeclaration{first.Declaration(), second.Declaration()}); err != nil {
		t.Fatal(err)
	}
}
func deployTestID(id int64) *int64 { return &id }
func TestDeployKeySnapshotCompanions(t *testing.T) {
	before := deployKeyModelState(t, 1, deployKeyFixtureOne)
	after := before
	after.Verified = true
	after.AddedBy = DeployKeyOptionalString{Present: true, Value: "different"}
	after.LastUsed = DeployKeyOptionalString{Present: true, Null: true}
	if !DeployKeyStableEqual(before, after) {
		t.Fatal("dynamic metadata drift")
	}
	enabled := false
	after.Enabled = &enabled
	if DeployKeyStableEqual(before, after) {
		t.Fatal("unknown is not false")
	}
	before.Enabled = &enabled
	if !DeployKeyStableEqual(before, after) {
		t.Fatal("same false")
	}
	other := true
	after.Enabled = &other
	if DeployKeyStableEqual(before, after) {
		t.Fatal("grant companion drift")
	}
	observation, _ := NewDeployKeyObservation(DeployKeyTarget{ID: 1, OwnerID: 2, FullName: "owner/repo", Archived: true}, []DeployKeyState{before})
	active, err := observation.AfterOwnUnarchive()
	if err != nil {
		t.Fatal(err)
	}
	target, states, _ := active.Values()
	if target.Archived || target.ID != 1 || target.OwnerID != 2 || !DeployKeyStableEqual(before, states[0]) {
		t.Fatal("unarchive identity")
	}
	states[0].Enabled = nil
	_, again, _ := active.Values()
	if again[0].Enabled == nil {
		t.Fatal("snapshot aliases returned state")
	}
}

func TestDeployKeyPrunePresence(t *testing.T) {
	empty, _ := NewDeployKeyDeclarations(nil)
	for _, scope := range []string{"deploy-keys", "all"} {
		p, err := ParsePruneScopes(scope)
		if err != nil || !p.Has("deploy-keys") || !p.Manages(Declaration{DeployKeys: &empty}) || p.Manages(Declaration{}) {
			t.Fatal(scope, p, err)
		}
	}
	if (PruneScopes{}).Manages(Declaration{DeployKeys: &empty}) {
		t.Fatal("implicit pruning")
	}
}
