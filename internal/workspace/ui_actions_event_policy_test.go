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
