package github

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPrepareForkApprovalObservation(t *testing.T) {
	for _, raw := range []string{"first_time_contributors_new_to_github", "first_time_contributors", "all_external_contributors", "FUTURE", "", " \t"} {
		t.Run(raw, func(t *testing.T) {
			observation, err := prepareForkApprovalObservation(&raw)
			if err != nil {
				t.Fatal(err)
			}
			value, err := observation.Value()
			if err != nil {
				t.Fatal(err)
			}
			wantValue := raw
			wantRecord := settings.ObservationNormalization{}
			if !settings.ValidForkApprovalPolicy(raw) {
				wantValue = "first_time_contributors"
				wantRecord = settings.ObservationNormalization{Path: settings.ForkApprovalPath, RawValue: raw, Fallback: string(settings.DefaultForkApprovalChoice)}
			}
			if value != wantValue || observation.Normalization() != wantRecord {
				t.Fatalf("value=%q record=%+v; want %q %+v", value, observation.Normalization(), wantValue, wantRecord)
			}
			wire, err := observation.RawValue()
			if err != nil || wire != raw {
				t.Fatalf("raw=%q err=%v; want %q", wire, err, raw)
			}
			original := raw
			raw = "changed caller storage"
			wire, err = observation.RawValue()
			if err != nil || wire != original {
				t.Fatalf("raw ownership=%q err=%v", wire, err)
			}
		})
	}
	observation, err := prepareForkApprovalObservation(nil)
	if err != nil || observation != (settings.ForkApprovalObservation{}) {
		t.Fatalf("nil projection=%+v err=%v", observation, err)
	}
	if _, err := observation.Value(); err == nil {
		t.Fatal("nil became acquired")
	}
}

func TestForkApprovalCompleteGetNormalizationAndRawVerification(t *testing.T) {
	for _, raw := range []string{"FUTURE", "", " \t", "first_time_contributors", "all_external_contributors"} {
		for _, desired := range []string{"first_time_contributors", "all_external_contributors", "first_time_contributors_new_to_github"} {
			for _, retainRaw := range []bool{false, true} {
				t.Run(raw+" to "+desired+map[bool]string{false: " applied", true: " ignored"}[retainRaw], func(t *testing.T) {
					policy := raw
					writes, visibilityReads, policyReads := 0, 0, 0
					client := &ForkWorkflowsClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
						if req.Method == "GET" && req.URL.Path == "/repos/owner/current" {
							visibilityReads++
							return actionsResponse(req, 200, `{"visibility":"public"}`), nil
						}
						if req.URL.Path != "/repos/owner/current/actions/permissions/fork-pr-contributor-approval" {
							t.Fatal(req.URL)
						}
						if req.Method == "GET" {
							policyReads++
							data, _ := json.Marshal(map[string]string{"approval_policy": policy})
							return actionsResponse(req, 200, string(data)), nil
						}
						if req.Method != "PUT" {
							t.Fatal(req.Method)
						}
						writes++
						var payload map[string]string
						if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
							t.Fatal(err)
						}
						if !reflect.DeepEqual(payload, map[string]string{"approval_policy": desired}) {
							t.Fatal(payload)
						}
						if !retainRaw {
							policy = desired
						}
						return actionsResponse(req, 204, ""), nil
					}).rest}
					var warnings []settings.ObservationNormalization
					client.NormalizationWarning = func(record settings.ObservationNormalization) { warnings = append(warnings, record) }
					repo := repository.Repository{Owner: "owner", Name: "current"}
					observed, err := client.ReadApproval(repo)
					if err != nil {
						t.Fatal(err)
					}
					wire, err := observed.RawValue()
					if err != nil || wire != raw || len(warnings) != 0 {
						t.Fatal(wire, err, warnings)
					}
					wrote, err := client.UpdateApproval(repo, desired)
					needsWrite := raw != desired
					wantErr := needsWrite && retainRaw
					var pre interface{ ReadFailedBeforeWrite() }
					if (err != nil) != wantErr || wrote != (needsWrite && !wantErr) || errors.As(err, &pre) {
						t.Fatal(wrote, err)
					}
					wantWrites, wantReads := 0, 2
					if needsWrite {
						wantWrites, wantReads = 1, 3
					}
					wantWarnings := 0
					if !settings.ValidForkApprovalPolicy(raw) {
						wantWarnings = 1
						if retainRaw {
							wantWarnings++
						}
					}
					if writes != wantWrites || policyReads != wantReads || visibilityReads != wantReads || len(warnings) != wantWarnings {
						t.Fatal(writes, policyReads, visibilityReads, warnings)
					}
					for _, record := range warnings {
						if record != observed.Normalization() {
							t.Fatal(record)
						}
					}
				})
			}
		}
	}
}

func TestForkApprovalWarningRequiresAcquiredNormalization(t *testing.T) {
	for _, raw := range []*string{nil, actionsUIPtr("first_time_contributors"), actionsUIPtr(""), actionsUIPtr("FUTURE")} {
		observed, err := prepareForkApprovalObservation(raw)
		if err != nil {
			t.Fatal(err)
		}
		var warnings []settings.ObservationNormalization
		client := &ForkWorkflowsClient{NormalizationWarning: func(record settings.ObservationNormalization) { warnings = append(warnings, record) }}
		client.warnApprovalNormalization(observed)
		want := 0
		if raw != nil && !settings.ValidForkApprovalPolicy(*raw) {
			want = 1
		}
		if len(warnings) != want || want == 1 && warnings[0] != observed.Normalization() {
			t.Fatalf("warnings=%+v observation=%+v", warnings, observed)
		}
		client.NormalizationWarning = nil
		client.warnApprovalNormalization(observed)
	}
}
