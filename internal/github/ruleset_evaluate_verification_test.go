package github

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestEvaluateTransitionsRequireIndependentReadback(t *testing.T) {
	for _, modes := range [][2]string{{"disabled", "evaluate"}, {"evaluate", "active"}, {"evaluate", "disabled"}} {
		for _, stale := range []bool{false, true} {
			t.Run(modes[0]+"-"+modes[1]+"-stale="+map[bool]string{true: "true", false: "false"}[stale], func(t *testing.T) {
				before, err := decodeRulesetPolicyFromTest([]byte(strings.Replace(basicRulesetJSON, `"enforcement":"active"`, `"enforcement":"`+modes[0]+`"`, 1)))
				if err != nil {
					t.Fatal(err)
				}
				after := before
				after.Enforcement = rulesetPtr(modes[1])
				change, err := settings.NewRulesetChange("update", 7, &before, &after)
				if err != nil {
					t.Fatal(err)
				}
				var methods []string
				client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
					methods = append(methods, req.Method)
					mode := modes[1]
					if req.Method == "GET" && stale {
						mode = modes[0]
					}
					return labelsResponse(req, 200, strings.Replace(basicRulesetJSON, `"enforcement":"active"`, `"enforcement":"`+mode+`"`, 1)), nil
				}))
				id, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
				if id != 7 || (err != nil) != stale || !reflect.DeepEqual(methods, []string{"PUT", "GET"}) {
					t.Fatal(id, err, methods)
				}
			})
		}
	}
}
