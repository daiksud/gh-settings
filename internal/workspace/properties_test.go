package workspace

import (
	"reflect"
	"testing"
)

func TestPropertyYAMLPresenceAndStrictUnion(t *testing.T) {
	d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings:\n  custom_properties:\n    clear: null\n    empty: ''\n    list: []\n    flag: 'false'\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Properties) != 4 || d.Properties["clear"] != nil || d.Properties["empty"] != "" || d.Properties["flag"] != "false" || !reflect.DeepEqual(d.Properties["list"], []string{}) {
		t.Fatal(d)
	}
	encoded, err := EncodeConfig(d)
	if err != nil {
		t.Fatal(err)
	}
	round, err := DecodeConfig(encoded)
	if err != nil || !reflect.DeepEqual(round.Properties, d.Properties) {
		t.Fatal(string(encoded), round, err)
	}
	for _, value := range []string{"null", "[]", "{bad: false}", "{bad: 1}", "{bad: {nested: value}}", "{bad: [a, a]}", "{bad: [null]}", "{bad: [false]}", "{'bad name': value}", "{a: x, a: y}"} {
		if _, err := DecodeConfig([]byte("schemaVersion: 1\nsettings:\n  custom_properties: " + value + "\n")); err == nil {
			t.Fatal("invalid property YAML", value)
		}
	}
	for _, value := range []string{"{}", "{custom_properties: {}}"} {
		d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: " + value))
		if err != nil || !d.Empty() {
			t.Fatal(d, err)
		}
	}
}
