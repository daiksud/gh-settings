package github

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestTeamApprovalsPrewriteWarningsKeepNumericIdentityOrder(t *testing.T) {
	for _, mode := range []string{"equal", "different", "reviewer removed", "replacement", "PR removed"} {
		change := teamApprovalsChange(t, mode)
		original := change.Normalizations()
		var seen []settings.RulesetTeamApprovalsNormalization
		want := []settings.RulesetTeamApprovalsNormalization{{ReviewerID: 2, Path: settings.RulesetTeamApprovalsPath, RawValue: -1, Fallback: 0}, {ReviewerID: 10, Path: settings.RulesetTeamApprovalsPath, RawValue: 11, Fallback: 0}}
		client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
			if !reflect.DeepEqual(seen, want) {
				t.Fatal("warning order before write", seen, want)
			}
			if req.Method == "PUT" {
				var got map[string]any
				if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, teamApprovalsPayload(mode)) {
					t.Fatal("warning mutated payload", got)
				}
			} else if req.Method != "GET" {
				t.Fatal(req.Method)
			}
			body := teamApprovalsPayload(mode)
			body["id"], body["source_type"], body["source"] = 7, "Repository", "owner/current"
			data, _ := json.Marshal(body)
			return labelsResponse(req, 200, string(data)), nil
		}))
		client.RulesetTeamApprovalsNormalizationWarning = func(id int64, name string, n settings.RulesetTeamApprovalsNormalization) {
			if id != 7 || name != "approvals" {
				t.Fatal(id, name)
			}
			seen = append(seen, n)
			n.ReviewerID = 99
			n.RawValue = 9
		}
		id, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
		if err != nil || id != 7 || !reflect.DeepEqual(seen, want) || !reflect.DeepEqual(change.Normalizations(), original) {
			t.Fatal(id, err, seen)
		}
	}
}
