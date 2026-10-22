package workspace

import (
	"reflect"
	"strings"
	"testing"
)

func TestSponsorshipsConfigTypesAndRoundTrip(t *testing.T) {
	for _, value := range []string{"true", "false"} {
		d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {has_sponsorships: " + value + "}}\n"))
		if err != nil || d.Repository.HasSponsorships == nil || *d.Repository.HasSponsorships != (value == "true") || !d.Repository.BaseEmpty() || !d.Repository.PRAccessEmpty() {
			t.Fatal(d, err)
		}
		encoded, err := EncodeConfig(d)
		if err != nil || !strings.Contains(string(encoded), "has_sponsorships: "+value) {
			t.Fatal(string(encoded), err)
		}
		round, err := DecodeConfig(encoded)
		if err != nil || !reflect.DeepEqual(round, d) {
			t.Fatal(round, err)
		}
	}
	for _, value := range []string{"null", "'false'", "0", "[]", "{}", "yes", "!!bool on"} {
		if _, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {has_sponsorships: " + value + "}}\n")); err == nil {
			t.Fatal("invalid boolean accepted", value)
		}
	}
	d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {has_issues: false}}\n"))
	if err != nil || d.Repository.HasSponsorships != nil {
		t.Fatal("omission synthesized a sponsorship setting", d, err)
	}
	encoded, err := EncodeConfig(d)
	if err != nil || strings.Contains(string(encoded), "has_sponsorships") {
		t.Fatal("omission lost during encoding", string(encoded), err)
	}
}
