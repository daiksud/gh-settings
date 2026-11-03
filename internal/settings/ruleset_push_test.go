package settings

import (
	"reflect"
	"strings"
	"testing"
)

func TestPushRulesetValidationAndOwnedPolicy(t *testing.T) {
	paths := []string{"z/**", "a/**"}
	exceptions := []string{"z/example"}
	policy := RulesetDeclaration{Name: "push", Target: rsText("push"), Enforcement: rsText("active"), Rules: &[]RulesetRule{{Type: "file_path_restriction", Parameters: &RuleParameters{RestrictedFilePaths: &paths, IgnoredFilePaths: &exceptions}}, {Type: "max_file_size", Parameters: &RuleParameters{MaxFileSize: paramInt(10), IgnoredFilePaths: rsList()}}}}
	declared, err := NewRulesetDeclarations([]RulesetDeclaration{policy})
	if err != nil {
		t.Fatal(err)
	}
	paths[0] = "changed"
	exceptions[0] = "changed"
	values := declared.Values()
	if values[0].Conditions != nil || !reflect.DeepEqual(*ruleOf(t, values[0], "file_path_restriction").RestrictedFilePaths, []string{"a/**", "z/**"}) || (*ruleOf(t, values[0], "file_path_restriction").IgnoredFilePaths)[0] != "z/example" || len(*ruleOf(t, values[0], "max_file_size").IgnoredFilePaths) != 0 {
		t.Fatal("policy lost or aliased", values)
	}
	for _, r := range []RulesetRule{
		{Type: "file_path_restriction"}, {Type: "file_extension_restriction", Parameters: &RuleParameters{}},
		{Type: "file_path_restriction", Parameters: &RuleParameters{RestrictedFilePaths: rsList("x", "x"), IgnoredFilePaths: rsList()}},
		{Type: "file_extension_restriction", Parameters: &RuleParameters{RestrictedFileExtensions: rsList(strings.Repeat("x", 201))}},
		{Type: "max_file_size", Parameters: &RuleParameters{MaxFileSize: paramInt(0), IgnoredFilePaths: rsList()}}, {Type: "max_file_size", Parameters: &RuleParameters{MaxFileSize: paramInt(101), IgnoredFilePaths: rsList()}},
		{Type: "max_file_path_length", Parameters: &RuleParameters{MaxFilePathLength: paramInt(32768)}},
		{Type: "max_file_path_length", Parameters: &RuleParameters{MaxFilePathLength: paramInt(10), IgnoredFilePaths: rsList()}},
		{Type: "max_file_size", Parameters: &RuleParameters{MaxFileSize: paramInt(10), IgnoredFilePaths: rsList(), Severity: rsText("errors")}},
	} {
		if err := validateRule(r, true); err == nil {
			t.Fatal("invalid push parameters accepted", r)
		}
	}
	for _, target := range []string{"branch", "tag"} {
		*policy.Target = target
		if _, err := NewRulesetDeclarations([]RulesetDeclaration{policy}); err == nil {
			t.Fatal("push rules on ref", target)
		}
	}
	*policy.Target = "push"
	policy.Rules = &[]RulesetRule{{Type: "deletion"}}
	if _, err := NewRulesetDeclarations([]RulesetDeclaration{policy}); err == nil {
		t.Fatal("ref rule on push")
	}
	policy.Rules = &[]RulesetRule{}
	policy.Conditions = &RulesetConditions{}
	if _, err := NewRulesetDeclarations([]RulesetDeclaration{policy}); err == nil {
		t.Fatal("conditions on push")
	}
}
