package github

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func metadataPolicyJSON(kind, parameters string) string {
	return strings.Replace(basicRulesetJSON, `"rules":[{"type":"deletion"},{"type":"non_fast_forward"}]`, `"rules":[{"type":"deletion"},{"type":"`+kind+`","parameters":`+parameters+`}]`, 1)
}

func TestMetadataRequirementsLiteralMappings(t *testing.T) {
	for _, test := range []struct {
		choice, operator string
		negate           bool
	}{
		{"must_start_with_matching_pattern", "starts_with", false}, {"must_not_start_with_matching_pattern", "starts_with", true},
		{"must_end_with_matching_pattern", "ends_with", false}, {"must_not_end_with_matching_pattern", "ends_with", true},
		{"must_contain_matching_pattern", "contains", false}, {"must_not_contain_matching_pattern", "contains", true},
		{"must_match_given_regex_pattern", "regex", false}, {"must_not_match_given_regex_pattern", "regex", true},
	} {
		ui := uimodel.MetadataPatternRule{Description: rulesetPtr(" exact description "), Requirement: test.choice, Pattern: rulesetPtr("(?m)^test\\n?$")}
		p, err := declareUIMetadataPattern(ui)
		if err != nil {
			t.Fatal(err)
		}
		if *p.Operator != test.operator || (p.Negate != nil && *p.Negate) != test.negate {
			t.Fatal(test, p)
		}
		observed, err := observeUIMetadataPattern(p)
		if err != nil || !reflect.DeepEqual(observed, &ui) {
			t.Fatal(observed, err)
		}
	}
	if _, err := declareUIMetadataPattern(uimodel.MetadataPatternRule{Pattern: rulesetPtr("x"), Requirement: "regex"}); err == nil {
		t.Fatal("native alias accepted")
	}
}

func TestMetadataReadsRejectIncompleteUnknownMixedAndDuplicate(t *testing.T) {
	for _, params := range []string{`{}`, `{"operator":"regex"}`, `{"pattern":"x"}`, `{"operator":null,"pattern":"x"}`, `{"operator":"regex","pattern":null}`, `{"operator":"regex","pattern":3}`, `{"operator":"regex","pattern":"x","negate":null}`, `{"operator":"regex","pattern":"x","negate":"false"}`, `{"operator":"regex","pattern":"x","Name":"description"}`, `{"operator":"glob","pattern":"x"}`, `{"operator":"regex","pattern":"x","severity":"errors"}`} {
		if _, err := decodeRulesetPolicyFromTest([]byte(metadataPolicyJSON("commit_message_pattern", params))); err == nil {
			t.Fatal("accepted", params)
		}
	}
	for _, flag := range []string{"", `,"negate":false`, `,"negate":true`} {
		policy, err := decodeRulesetPolicyFromTest([]byte(metadataPolicyJSON("commit_message_pattern", `{"operator":"regex","pattern":"x","name":""`+flag+`}`)))
		if err != nil {
			t.Fatal(flag, err)
		}
		obs, err := settings.NewRulesetObservation([]settings.RulesetState{{ID: 7, Policy: policy}}, 0)
		if err != nil {
			t.Fatal(err)
		}
		exported, err := obs.ExportDeclarations()
		if err != nil {
			t.Fatal(err)
		}
		source := uiExportBase()
		source.Rulesets = &exported
		ui, err := UIFromExportDeclaration(source)
		if err != nil {
			t.Fatal(err)
		}
		data, err := workspace.EncodeConfig(ui)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := workspace.DecodeConfig(data)
		if err != nil {
			t.Fatal(string(data), err)
		}
		declared, err := DeclarationFromUI(decoded)
		if err != nil {
			t.Fatal(err)
		}
		changes, err := settings.ReconcileRulesets(obs, declared.Rulesets)
		if err != nil || len(changes) != 0 {
			t.Fatal("roundtrip drift", flag, changes, err)
		}
	}
	body := metadataPolicyJSON("commit_message_pattern", `{"operator":"regex","pattern":"x"}`)
	body = strings.Replace(body, `{"type":"deletion"}`, `{"type":"commit_message_pattern","parameters":{"operator":"contains","pattern":"y"}}`, 1)
	if _, err := decodeRulesetPolicyFromTest([]byte(body)); err == nil {
		t.Fatal("duplicate restriction discarded")
	}
}

func TestMetadataMutationRequiresIndependentReadbackIncludingRemoval(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	for _, kind := range []string{"commit_message_pattern", "commit_author_email_pattern", "committer_email_pattern", "branch_name_pattern", "tag_name_pattern"} {
		for _, operation := range []string{"create", "update", "remove"} {
			for _, drift := range []bool{false, true} {
				t.Run(kind+"/"+operation+"/"+map[bool]string{true: "drift", false: "ok"}[drift], func(t *testing.T) {
					body := metadataPolicyJSON(kind, `{"operator":"regex","pattern":"^before$","name":"exact","negate":true}`)
					if kind == "tag_name_pattern" {
						body = strings.Replace(body, `"target":"branch"`, `"target":"tag"`, 1)
					}
					before, err := decodeRulesetPolicyFromTest([]byte(body))
					if err != nil {
						t.Fatal(err)
					}
					afterBody := strings.Replace(body, "^before$", "^after$", 1)
					if operation == "remove" {
						afterBody = strings.Replace(afterBody, `,{"type":"`+kind+`","parameters":{"operator":"regex","pattern":"^after$","name":"exact","negate":true}}`, "", 1)
					}
					after, err := decodeRulesetPolicyFromTest([]byte(afterBody))
					if err != nil {
						t.Fatal(err)
					}
					var prior *settings.RulesetDeclaration
					op, id := "create", int64(0)
					if operation != "create" {
						prior, op, id = &before, "update", 7
					}
					change, err := settings.NewRulesetChange(op, id, prior, &after)
					if err != nil {
						t.Fatal(err)
					}
					var expected map[string]any
					json.Unmarshal([]byte(afterBody), &expected)
					if kind != "tag_name_pattern" && operation != "remove" {
						expected["rules"].([]any)[0], expected["rules"].([]any)[1] = expected["rules"].([]any)[1], expected["rules"].([]any)[0]
					}
					for _, key := range []string{"id", "source_type", "source", "current_user_can_bypass"} {
						delete(expected, key)
					}
					calls := 0
					client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
						calls++
						if r.Method == "GET" {
							if calls != 2 || r.URL.Path != "/repos/owner/current/rulesets/7" || r.URL.Query().Get("includes_parents") != "false" {
								t.Fatal("wrong readback", r.URL, calls)
							}
							result := afterBody
							if drift {
								result = body
							}
							return labelsResponse(r, 200, result), nil
						}
						gotBytes, _ := io.ReadAll(r.Body)
						var got any
						json.Unmarshal(gotBytes, &got)
						if !reflect.DeepEqual(got, expected) {
							t.Fatalf("payload %s != %v", gotBytes, expected)
						}
						return labelsResponse(r, 200, afterBody), nil
					}))
					_, err = client.ApplyRuleset(repo, change)
					if (err != nil) != drift || calls != 2 {
						t.Fatal("verification", err, calls)
					}
					if drift && !strings.Contains(err.Error(), "writes may already have applied") {
						t.Fatal("uncertainty lost", err)
					}
				})
			}
		}
	}
}
