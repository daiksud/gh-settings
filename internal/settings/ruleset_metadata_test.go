package settings

import "testing"

func TestMetadataRulesValidateApplicabilityAndCompleteParameters(t *testing.T) {
	for _, kind := range []string{"commit_message_pattern", "commit_author_email_pattern", "committer_email_pattern", "branch_name_pattern", "tag_name_pattern"} {
		for _, operator := range []string{"starts_with", "ends_with", "contains", "regex"} {
			policy := basicPolicy("metadata")
			if kind == "tag_name_pattern" {
				policy.Target = rsText("tag")
			}
			policy.Rules = &[]RulesetRule{{Type: kind, Parameters: &RuleParameters{Operator: rsText(operator), Pattern: rsText("pattern")}}}
			if _, err := NewRulesetDeclarations([]RulesetDeclaration{policy}); err != nil {
				t.Fatal(kind, operator, err)
			}
			for _, mode := range []string{"no-operator", "no-pattern", "bad-operator", "foreign-parameters"} {
				p := *(*policy.Rules)[0].Parameters
				switch mode {
				case "no-operator":
					p.Operator = nil
				case "no-pattern":
					p.Pattern = nil
				case "bad-operator":
					p.Operator = rsText("glob")
				case "foreign-parameters":
					p.Severity = rsText("errors")
				}
				bad := policy
				bad.Rules = &[]RulesetRule{{Type: kind, Parameters: &p}}
				if _, err := NewRulesetDeclarations([]RulesetDeclaration{bad}); err == nil {
					t.Fatal(kind, mode, "accepted")
				}
			}
		}
	}
	for _, kind := range []string{"branch_name_pattern", "tag_name_pattern"} {
		policy := basicPolicy("wrong-target")
		if kind == "branch_name_pattern" {
			policy.Target = rsText("tag")
		}
		policy.Rules = &[]RulesetRule{{Type: kind, Parameters: &RuleParameters{Operator: rsText("regex"), Pattern: rsText(".*")}}}
		if _, err := NewRulesetDeclarations([]RulesetDeclaration{policy}); err == nil {
			t.Fatal("wrong target accepted", kind)
		}
	}
	for _, kind := range []string{"deletion", "code_quality", "pull_request", "required_status_checks"} {
		p := basicPolicy("mixed")
		p.Rules = &[]RulesetRule{{Type: kind, Parameters: &RuleParameters{Negate: paramBool(false)}}}
		if _, err := NewRulesetDeclarations([]RulesetDeclaration{p}); err == nil {
			t.Fatal("foreign false negate accepted", kind)
		}
	}
}

func TestMetadataRulesOwnedValuesAndFalseAbsenceEquivalence(t *testing.T) {
	p := basicPolicy("metadata")
	p.Rules = &[]RulesetRule{{Type: "commit_message_pattern", Parameters: &RuleParameters{Name: rsText(""), Operator: rsText("regex"), Pattern: rsText("(?m)^test\n"), Negate: paramBool(false)}}}
	d, err := NewRulesetDeclarations([]RulesetDeclaration{p})
	if err != nil {
		t.Fatal(err)
	}
	*(*p.Rules)[0].Parameters.Pattern = "changed"
	v := d.Values()[0]
	if *(*v.Rules)[0].Parameters.Pattern != "(?m)^test\n" || (*v.Rules)[0].Parameters.Negate != nil || *(*v.Rules)[0].Parameters.Name != "" {
		t.Fatal("not owned/canonical", v)
	}
	*(*v.Rules)[0].Parameters.Name = "changed"
	if *(*d.Values()[0].Rules)[0].Parameters.Name != "" {
		t.Fatal("mutable name escaped")
	}
}
