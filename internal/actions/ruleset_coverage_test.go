package actions

import (
	"encoding/json"
	"testing"
)

func TestRulesetCoveragePrivateReportsRequireCompleteNumbers(t *testing.T) {
	for _, test := range []struct {
		parameters string
		valid      bool
	}{
		{`{"minimum_coverage":80.5,"max_coverage_drop":0}`, true},
		{`{"minimum_coverage":0,"max_coverage_drop":100}`, true},
		{`{"minimum_coverage":0}`, false},
		{`{"minimum_coverage":null,"max_coverage_drop":0}`, false},
		{`{"minimum_coverage":"80","max_coverage_drop":0}`, false},
		{`{"minimum_coverage":false,"max_coverage_drop":0}`, false},
		{`{"minimum_coverage":101,"max_coverage_drop":0}`, false},
		{`{"minimum_coverage":80,"max_coverage_drop":-1}`, false},
		{`{"minimum_coverage":80,"max_coverage_drop":0,"severity":"errors"}`, false},
		{`{"minimum_coverage":80,"max_coverage_drop":0,"future":true}`, false},
	} {
		body := `{"operation":"create","after":{"name":"coverage","target":"branch","enforcement":"active","conditions":{"ref_name":{"include":["~DEFAULT_BRANCH"],"exclude":[]}},"bypass_actors":[],"rules":[{"type":"code_coverage","parameters":` + test.parameters + `}]}}`
		var entry RulesetPlanChange
		err := json.Unmarshal([]byte(body), &entry)
		if err == nil {
			_, err = validateRulesetReport([]RulesetPlanChange{entry})
		}
		if (err == nil) != test.valid {
			t.Fatal(test.parameters, err)
		}
	}
}
