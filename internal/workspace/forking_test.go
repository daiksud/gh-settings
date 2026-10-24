package workspace

import (
	"reflect"
	"strings"
	"testing"
)

func TestForkingBooleanRoundTripAndComposition(t *testing.T) {
	for _, v := range []string{"true", "false"} {
		d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {allow_forking: " + v + "}}\n"))
		if err != nil || d.Repository.AllowForking == nil || *d.Repository.AllowForking != (v == "true") {
			t.Fatal(d, err)
		}
		b, err := EncodeConfig(d)
		if err != nil || !strings.Contains(string(b), "allow_forking: "+v) {
			t.Fatal(string(b), err)
		}
		round, err := DecodeConfig(b)
		if err != nil || !reflect.DeepEqual(round, d) {
			t.Fatal(round, err)
		}
	}
	for _, v := range []string{"null", "'false'", "0", "[]", "{}", "yes", "!!bool on"} {
		if _, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {allow_forking: " + v + "}}\n")); err == nil {
			t.Fatal(v)
		}
	}
	if _, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {allow_forking: false, archived: true}}\n")); err == nil {
		t.Fatal("accepted archive-after-forking composition")
	}
	if _, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {allow_forking: false, archived: false}}\n")); err != nil {
		t.Fatal("rejected explicit unarchive-first composition", err)
	}
	d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {has_issues: false}}\n"))
	if err != nil || d.Repository.AllowForking != nil {
		t.Fatal(d, err)
	}
	b, err := EncodeConfig(d)
	if err != nil || strings.Contains(string(b), "allow_forking") {
		t.Fatal(string(b), err)
	}
}
