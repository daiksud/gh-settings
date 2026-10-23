package settings

import (
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
)

func TestForkingObservationRequiresAcquisitionAndIdentity(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	if _, err := (ForkingObservation{}).Value(); err == nil {
		t.Fatal("unacquired became false")
	}
	for _, v := range []bool{false, true} {
		o, err := NewForkingObservation(repo, 42, 7, "owner/current", v)
		if err != nil {
			t.Fatal(err)
		}
		got, err := o.Value()
		if err != nil || got != v {
			t.Fatal(got, err)
		}
		if err := o.Validate(repository.Repository{Owner: "other", Name: "current"}); err == nil {
			t.Fatal("wrong target")
		}
	}
	for _, tc := range []struct {
		id, org int64
		name    string
	}{{0, 7, "owner/current"}, {42, 0, "owner/current"}, {42, 7, "other/current"}} {
		if _, err := NewForkingObservation(repo, tc.id, tc.org, tc.name, false); err == nil {
			t.Fatal(tc)
		}
	}
	for _, id := range AllRepositoryFields() {
		if id.Path() == "repository.allow_forking" {
			t.Fatal("ordinary export gained opt-in setting")
		}
	}
}
