package github

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
)

func TestCodeQLExportSingleResponseKnownAndUnknown(t *testing.T) {
	for _, scenario := range []string{"disabled", "configured", "incomplete-details", "unknown-threat", "403", "network", "missing-state", "invalid-state", "malformed"} {
		t.Run(scenario, func(t *testing.T) {
			reads := 0
			c := &CodeQLDefaultSetupClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" {
					t.Fatal("mutation", r.Method)
				}
				if r.URL.Path == "/repos/owner/current" {
					return actionsResponse(r, 200, `{"visibility":"public","owner":{"type":"User"}}`), nil
				}
				reads++
				raw := codeQLConfigRaw()
				switch scenario {
				case "disabled":
					return actionsResponse(r, 200, `{"state":"not-configured"}`), nil
				case "403":
					return actionsResponse(r, 403, `{}`), nil
				case "network":
					return nil, errors.New("connection failed")
				case "missing-state":
					delete(raw, "state")
				case "invalid-state":
					raw["state"] = "unknown"
				case "malformed":
					return actionsResponse(r, 200, `{"state":`), nil
				case "incomplete-details":
					delete(raw, "languages")
				case "unknown-threat":
					raw["threat_model"] = "future"
				}
				b, _ := json.Marshal(raw)
				return actionsResponse(r, 200, string(b)), nil
			}).rest}
			setup, configuration, err := c.ReadForExport(repository.Repository{Owner: "owner", Name: "current"})
			enabled, stateErr := setup.Value()
			if reads != 1 {
				t.Fatal("expected single setup GET", reads)
			}
			switch scenario {
			case "disabled":
				if err != nil || stateErr != nil || enabled {
					t.Fatal(enabled, stateErr, err)
				}
			case "configured":
				if err != nil || stateErr != nil || !enabled {
					t.Fatal(enabled, stateErr, err)
				}
				if _, err := configuration.Declaration(); err != nil {
					t.Fatal(err)
				}
			case "incomplete-details", "unknown-threat":
				if err == nil || stateErr != nil || !enabled {
					t.Fatal(enabled, stateErr, err)
				}
			default:
				if err == nil || stateErr == nil {
					t.Fatal("unknown became known", enabled, stateErr, err)
				}
			}
		})
	}
}
