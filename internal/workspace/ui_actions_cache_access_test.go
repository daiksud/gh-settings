package workspace

import (
	"reflect"
	"testing"
)

func TestUIActionsCacheAccessRoundTrip(t *testing.T) {
	for _, controls := range []string{"cache: {cache_retention: 7, cache_size_eviction_limit: 10}, access: not_accessible", "cache: {cache_retention: 1}", "cache: {cache_size_eviction_limit: 10000}", "access: accessible_from_repositories_owned_by_user"} {
		raw := "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {general: {" + controls + "}}}}"
		got, err := DecodeConfig([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := EncodeConfig(got)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeConfig(encoded)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatalf("roundtrip: %s %v", encoded, err)
		}
	}
}
func TestUIActionsCacheAccessRejectSyntax(t *testing.T) {
	for _, controls := range []string{"cache: null", "cache: {cache_retention: null}", "cache: {cache_size_eviction_limit: null}", "cache: {cache_retention: '7'}", "cache: {cache_retention: 1.5}", "cache: {cache_size_eviction_limit: true}", "cache: {cache_retention_days: 7}", "access: null", "access: true", "access_level: none"} {
		raw := "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {general: {" + controls + "}}}}"
		if _, err := DecodeConfig([]byte(raw)); err == nil {
			t.Fatalf("accepted %s", controls)
		}
	}
}
