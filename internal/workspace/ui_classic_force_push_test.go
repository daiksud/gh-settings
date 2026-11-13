package workspace

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/github"
)

func TestClassicForcePushLiteralCodec(t *testing.T) {
	const prefix = "version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: [{id: 7, branch_name_pattern: 'release/**', allow_force_pushes: "
	for _, value := range []string{"{}", "{enabled: false}", "{enabled: true, who_can_force_push: everyone}", "{enabled: true, who_can_force_push: specified_actors, actors: [{type: user, id: U_1}, {type: app, id: A_2}]}"} {
		ui, err := DecodeConfig([]byte(prefix + value + "}]}}\n"))
		if err != nil {
			t.Fatal(value, err)
		}
		if _, err = github.DeclarationFromUI(ui); err != nil {
			t.Fatal(value, err)
		}
		data, err := EncodeConfig(ui)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeConfig(data)
		if err != nil || !reflect.DeepEqual(ui, again) {
			t.Fatal(string(data), err)
		}
	}
}

func TestClassicForcePushInvalidLiteral(t *testing.T) {
	const prefix = "version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: [{branch_name_pattern: main, allow_force_pushes: "
	for _, value := range []string{"null", "true", "[]", "{enabled: null}", "{enabled: true}", "{who_can_force_push: everyone}", "{enabled: false, who_can_force_push: everyone}", "{enabled: true, who_can_force_push: Everyone}", "{enabled: true, who_can_force_push: everyone, actors: []}", "{enabled: true, who_can_force_push: specified_actors}", "{enabled: true, who_can_force_push: specified_actors, actors: []}", "{enabled: true, who_can_force_push: specified_actors, actors: null}", "{enabled: true, who_can_force_push: specified_actors, actors: [null]}", "{enabled: true, who_can_force_push: specified_actors, actors: [{type: team, id: T_1}]}", "{enabled: true, who_can_force_push: specified_actors, actors: [{type: User, id: U_1}]}", "{enabled: true, who_can_force_push: specified_actors, actors: [{type: user, id: 1}]}", "{enabled: true, who_can_force_push: specified_actors, actors: [{type: user, id: null}]}", "{enabled: true, who_can_force_push: specified_actors, actors: [{type: user, id: U_1, login: owner}]}", "{enabled: true, who_can_force_push: specified_actors, actors: [{type: user, id: U_1}, {type: app, id: U_1}]}"} {
		ui, err := DecodeConfig([]byte(prefix + value + "}]}}\n"))
		if err == nil {
			_, err = github.DeclarationFromUI(ui)
		}
		if err == nil {
			t.Fatal("accepted invalid group", value)
		}
	}
}
