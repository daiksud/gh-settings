package workspace

import (
	"reflect"
	"strings"
	"testing"
)

func TestPRAccessConfigRoundtripAndStrictTypes(t *testing.T) {
	input := `schemaVersion: 1
settings:
  repository:
    has_pull_requests: false
    pull_request_creation_policy: COLLABORATORS_ONLY
`
	d, err := DecodeConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if d.Repository.HasPullRequests == nil || *d.Repository.HasPullRequests || d.Repository.PullRequestCreationPolicy == nil || *d.Repository.PullRequestCreationPolicy != "COLLABORATORS_ONLY" {
		t.Fatal(d)
	}
	encoded, err := EncodeConfig(d)
	if err != nil {
		t.Fatal(err)
	}
	round, err := DecodeConfig(encoded)
	if err != nil || !reflect.DeepEqual(round, d) {
		t.Fatal(string(encoded), err)
	}
	for _, pair := range [][2]string{{"false", "null"}, {"false", "'false'"}, {"COLLABORATORS_ONLY", "all"}, {"COLLABORATORS_ONLY", "null"}, {"COLLABORATORS_ONLY", "false"}, {"COLLABORATORS_ONLY", "''"}} {
		if _, err := DecodeConfig([]byte(strings.Replace(input, pair[0], pair[1], 1))); err == nil {
			t.Fatal("invalid policy accepted", pair)
		}
	}
	d, err = DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {has_pull_requests: false}}\n"))
	if err != nil || d.Repository.PullRequestCreationPolicy != nil {
		t.Fatal("companion invented", d, err)
	}
}
