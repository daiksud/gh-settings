package workspace

import (
 "reflect"
 "strings"
 "testing"
)

func TestArchiveConfigStrictBooleanAndOptInRoundTrip(t *testing.T) {
 for _, value := range []string{"true", "false"} {
  d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {archived: "+value+"}}\n"))
  if err != nil || d.Repository.Archived == nil || *d.Repository.Archived != (value == "true") || !d.Repository.BaseEmpty() || !d.Repository.PRAccessEmpty() {
   t.Fatal(value, d, err)
  }
  encoded, err := EncodeConfig(d)
  if err != nil || !strings.Contains(string(encoded), "archived: "+value) {t.Fatal(value, string(encoded), err)}
  round, err := DecodeConfig(encoded)
  if err != nil || !reflect.DeepEqual(round, d) {t.Fatal(value, round, err)}
 }
 for _, value := range []string{"null", "'false'", "0", "[]", "{}", "yes", "!!bool on"} {
  if _, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {archived: "+value+"}}\n")); err == nil {t.Fatal("invalid archived boolean accepted", value)}
 }
 omitted, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {has_issues: false}}\n"))
 if err != nil || omitted.Repository.Archived != nil {t.Fatal("synthesized archive", omitted, err)}
 encoded, err := EncodeConfig(omitted)
 if err != nil || strings.Contains(string(encoded), "archived:") {t.Fatal("archive omission lost", string(encoded), err)}
}
