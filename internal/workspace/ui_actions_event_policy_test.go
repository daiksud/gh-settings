package workspace

import (
	"reflect"
	"strings"
	"testing"
)

func TestUIActionsPoliciesRoundTrip(t *testing.T) {
	input := `version: daiksud/gh-settings@v0
repository:
    settings:
        actions:
            policies:
                - id: 7
                  policy_name: Restrict workflow events
                  enforcement_status: disabled
                  restrict_events:
                    allowed_events:
                        - push
                        - pull_request
                - policy_name: Another policy
                  enforcement_status: active
                  restrict_events:
                    allowed_events:
                        - workflow_dispatch
`
	value, err := DecodeConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	policies := value.Settings.Actions.Policies
	if policies == nil || len(*policies) != 2 || (*policies)[1].ID != nil || *(*policies)[0].ID != 7 {
		t.Fatalf("lost policy presence: %#v", policies)
	}
	encoded, err := EncodeConfig(value)
	if err != nil {
		t.Fatal(err)
	}
	again, err := DecodeConfig(encoded)
	if err != nil || !reflect.DeepEqual(value, again) {
		t.Fatalf("round trip lost policy values: %s, %v", encoded, err)
	}
	// The canonical encoder has a stable exact representation after one pass.
	second, err := EncodeConfig(again)
	if err != nil || string(second) != string(encoded) {
		t.Fatalf("unstable encoding: %s, %v", second, err)
	}
}

func TestUIActionsPoliciesPresence(t *testing.T) {
	for _, body := range []string{"{}", "{policies: []}", "{policies: [{policy_name: '', enforcement_status: '', restrict_events: {allowed_events: []}}]}"} {
		value, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {actions: " + body + "}}\n"))
		if err != nil {
			t.Fatal(err)
		}
		if (value.Settings.Actions.Policies != nil) != (body != "{}") {
			t.Fatalf("lost collection presence: %#v", value.Settings.Actions)
		}
		encoded, err := EncodeConfig(value)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeConfig(encoded)
		if err != nil || !reflect.DeepEqual(value, again) {
			t.Fatalf("lost explicit empty intent: %s, %v", encoded, err)
		}
	}
}

func TestUIActionsPoliciesRejectUnsafeValues(t *testing.T) {
	for _, body := range []string{
		"null", "{}", "[null]", "[{id: '7'}]", "[{id: null}]", "[{policy_name: null}]", "[{policy_name: false}]",
		"[{enforcement_status: null}]", "[{restrict_events: null}]", "[{restrict_events: []}]",
		"[{restrict_events: {allowed_events: null}}]", "[{restrict_events: {allowed_events: [null]}}]",
		"[{restrict_events: {allowed_events: [1]}}]", "[{restrict_events: {allowed_events: push}}]",
		"[{name: old}]", "[{enforcement: active}]", "[{target: all}]", "[{delete: true}]",
		"[{target_patterns: ['*']}]", "[{restrict_events: {events: [push]}}]", "[{policy_name: one, policy_name: two}]",
		"[{restrict_events: {allowed_events: !!str [push]}}]", "!!str []",
	} {
		t.Run(body, func(t *testing.T) {
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {policies: " + body + "}}}\n"
			if _, err := DecodeConfig([]byte(input)); err == nil {
				t.Fatal("accepted unsafe policy shape")
			}
		})
	}
}

func TestUIActionsPoliciesMergedUnknownKeyRejected(t *testing.T) {
	input := "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {policies: [{policy_name: policy, <<: {target_patterns: ['*']}}]}}}\n"
	if _, err := DecodeConfig([]byte(input)); err == nil || !strings.Contains(err.Error(), "target_patterns") {
		t.Fatalf("unknown merged key accepted: %v", err)
	}
}

func TestUIActionsWorkflowTargetRoundTripAndStrictTypes(t *testing.T) {
	prefix := "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {policies: [{policy_name: events, enforcement_status: disabled, restrict_events: {allowed_events: [push]}, target_workflows: "
	for _, target := range []string{`{include: ['.github/workflows/*.yml'], exclude: ['**/skip.yml']}`, `{include: [], exclude: ['skip.yml']}`, `{include: ['~ALL'], exclude: []}`} {
		value, err := DecodeConfig([]byte(prefix + target + "}]}}}\n"))
		if err != nil {
			t.Fatal(err)
		}
		data, err := EncodeConfig(value)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeConfig(data)
		if err != nil || !reflect.DeepEqual(value, again) {
			t.Fatal(string(data), err)
		}
	}
	for _, target := range []string{`null`, `[]`, `{include: null, exclude: []}`, `{include: [], exclude: null}`, `{include: [null], exclude: []}`, `{include: [true], exclude: []}`, `{include: [], exclude: [], actor: []}`} {
		if _, err := DecodeConfig([]byte(prefix + target + "}]}}}\n")); err == nil {
			t.Fatal("accepted", target)
		}
	}
}

func TestUIActionsActorRestrictionsRoundTripAndPresence(t *testing.T) {
	for _, body := range []string{
		`{}`, `{restrict_actors: {enabled: false}}`, `{restrict_events: {enabled: false}}`,
		`{restrict_actors: {allowed_actors: []}}`, `{restrict_actors: {enabled: true, allowed_actors: []}}`,
		`{restrict_actors: {allowed_actors: [{user: 123}, {role: write}, {role: maintain}, {role: repository_admin}]}, restrict_events: {allowed_events: [push]}}`,
	} {
		input := "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {policies: [" + body + "]}}}\n"
		value, err := DecodeConfig([]byte(input))
		if err != nil {
			t.Fatal(body, err)
		}
		data, err := EncodeConfig(value)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeConfig(data)
		if err != nil || !reflect.DeepEqual(value, again) {
			t.Fatalf("lost restriction presence: %s: %v", data, err)
		}
		stable, err := EncodeConfig(again)
		if err != nil || string(stable) != string(data) {
			t.Fatalf("unstable restriction encoding: %s: %v", stable, err)
		}
	}
}

func TestUIActionsActorRestrictionsRejectUnsafeYAML(t *testing.T) {
	for _, body := range []string{
		`{restrict_actors: null}`, `{restrict_actors: []}`, `{restrict_actors: {enabled: null}}`,
		`{restrict_actors: {enabled: 'false'}}`, `{restrict_actors: {enabled: 0}}`,
		`{restrict_actors: {allowed_actors: null}}`, `{restrict_actors: {allowed_actors: {}}}`,
		`{restrict_actors: {allowed_actors: [null]}}`, `{restrict_actors: {allowed_actors: [123]}}`,
		`{restrict_actors: {allowed_actors: [{user: null}]}}`, `{restrict_actors: {allowed_actors: [{user: '123'}]}}`,
		`{restrict_actors: {allowed_actors: [{user: true}]}}`, `{restrict_actors: {allowed_actors: [{user: 1.0}]}}`,
		`{restrict_actors: {allowed_actors: [{user: 9223372036854775808}]}}`,
		`{restrict_actors: {allowed_actors: [{role: null}]}}`, `{restrict_actors: {allowed_actors: [{role: 4}]}}`,
		`{restrict_actors: {allowed_actors: [{team: 1}]}}`, `{restrict_actors: {allowed_actors: [{type: User, id: 1}]}}`,
		`{restrict_actors: {allowed_actors: [{user: 1, user: 2}]}}`,
		`{restrict_actors: {allowed_actors: [], allowed_actors: []}}`,
		`{restrict_actors: {enabled: true, enabled: false}}`,
		`{restrict_actors: {allowed_actors: !!str []}}`,
		`{restrict_events: {enabled: null}}`, `{restrict_events: {enabled: 'true'}}`,
		`{restrict_actors: {allowed_actors: [&bad {user: null}, *bad]}}`,
		`{restrict_actors: {allowed_actors: [{<<: {team: 1}}]}}`,
	} {
		t.Run(body, func(t *testing.T) {
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {policies: [" + body + "]}}}\n"
			if _, err := DecodeConfig([]byte(input)); err == nil {
				t.Fatal("accepted unsafe actor shape")
			}
		})
	}
}
