package workspace

import (
	"reflect"
	"strings"
	"testing"
)

func TestRulesetsConfigStrictPresenceAndRoundTrip(t *testing.T) {
	for _, input := range []string{"schemaVersion: 1", "schemaVersion: 1\nsettings: {rulesets: []}", "schemaVersion: 1\nsettings: {rulesets: [{name: basic, target: branch, enforcement: disabled, conditions: {ref_name: {include: ['~ALL'], exclude: []}}, bypass_actors: [], rules: [{type: deletion, parameters: {}}]}]}", "schemaVersion: 1\nsettings: {rulesets: [{<<: {name: basic, enforcement: disabled}, conditions: {ref_name: {include: []}}, rules: []}]}"} {
		d, e := DecodeConfig([]byte(input))
		if e != nil {
			t.Fatal(input, e)
		}
		encoded, e := EncodeConfig(d)
		if e != nil {
			t.Fatal(e)
		}
		round, e := DecodeConfig(encoded)
		if e != nil || !reflect.DeepEqual(round, d) {
			t.Fatalf("%s %v", encoded, e)
		}
		if (d.Rulesets == nil) != !strings.Contains(input, "rulesets") {
			t.Fatal("presence lost")
		}
	}
}
func TestRulesetsConfigRejectsNullUnknownAndTypes(t *testing.T) {
	for _, input := range []string{"null", "{}", `[{name: a, Name: b}]`, `[{name: a, conditions: {Ref_name: {include: []}}}]`, `&x [*x]`, "[null]", "[{}]", "[{name: null}]", "[{name: 3}]", "[{name: a, target: null}]", "[{name: a, conditions: null}]", "[{name: a, conditions: {ref_name: null}}]", "[{name: a, conditions: {ref_name: {include: null}}}]", "[{name: a, conditions: {ref_name: {exclude: [null]}}}]", "[{name: a, rules: null}]", "[{name: a, rules: [{type: deletion, parameters: null}]}]", "[{name: a, rules: [{type: deletion, parameters: {unknown: true}}]}]", "[{name: a, rules: [{type: merge_queue}]}]", "[{name: a, bypass_actors: [{actor_type: Team, actor_id: 0}]}]", "[{name: a, bypass_actors: [{actor_type: DeployKey, actor_id: null}]}]", "[{name: a, bypass_actors: [{actor_type: Team, actor_id: '3'}]}]", "[{name: a, id: 0}]", "[{name: a, id: -1}]", "[{name: a, id: '1'}]", "[{name: a, id: 1.5}]", "[{name: a, id: null}]", "[{name: a, id: 7}, {name: b, id: 7}]", "[{name: a, conditions: {unknown: []}}]", "[{name: a, name: b}]", "[{name: a}, {name: a}]"} {
		if _, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {rulesets: " + input + "}\n")); err == nil {
			t.Fatal("accepted", input)
		}
	}
}

func TestRulesetIDRoundTrip(t *testing.T) {
	input := []byte("schemaVersion: 1\nsettings: {rulesets: [{id: 712, name: existing, enforcement: active, conditions: {ref_name: {include: ['~ALL']}}}]}\n")
	declaration, err := DecodeConfig(input)
	if err != nil {
		t.Fatal(err)
	}
	if declaration.Rulesets == nil || declaration.Rulesets.Values()[0].ID == nil || *declaration.Rulesets.Values()[0].ID != 712 {
		t.Fatal("lost selected REST ID", declaration)
	}
	encoded, err := EncodeConfig(declaration)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := DecodeConfig(encoded)
	if err != nil || !reflect.DeepEqual(declaration, roundTrip) || !strings.Contains(string(encoded), "id: 712") {
		t.Fatalf("round trip lost ID: %s %v", encoded, err)
	}
}
