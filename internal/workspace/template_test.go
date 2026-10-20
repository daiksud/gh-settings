package workspace

import (
	"reflect"
	"strings"
	"testing"
)

func TestTemplateDeclaration(t *testing.T) {
	for _, value := range []string{"true", "false"} {
		d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {is_template: " + value + "}}"))
		if err != nil || d.Empty() || d.Repository.BaseEmpty() || d.Repository.IsTemplate == nil || *d.Repository.IsTemplate != (value == "true") {
			t.Fatalf("value=%s declaration=%+v err=%v", value, d, err)
		}
		data, err := EncodeConfig(d)
		if err != nil || !strings.Contains(string(data), "is_template: "+value) {
			t.Fatalf("encode=%s err=%v", data, err)
		}
		round, err := DecodeConfig(data)
		if err != nil || !reflect.DeepEqual(round, d) {
			t.Fatalf("round=%+v err=%v", round, err)
		}
	}
	for _, value := range []string{"null", "", "'false'", "0", "[]", "{}", "yes", "!!bool on"} {
		if d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {is_template: " + value + "}}")); err == nil || !d.Empty() {
			t.Fatalf("accepted %q: %+v %v", value, d, err)
		}
	}
	d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {has_issues: false}}"))
	if err != nil || d.Repository.IsTemplate != nil {
		t.Fatalf("omission=%+v err=%v", d, err)
	}
	data, err := EncodeConfig(d)
	if err != nil || strings.Contains(string(data), "is_template") {
		t.Fatalf("omitted field introduced: %s %v", data, err)
	}
}
