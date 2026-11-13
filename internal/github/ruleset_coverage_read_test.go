package github

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestCompleteRulesetCoverageObservation(t *testing.T) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(qualityRulesetJSON), &raw); err != nil {
		t.Fatal(err)
	}
	raw["rules"] = json.RawMessage(`[{"type":"code_coverage","parameters":{"minimum_coverage":80.5,"max_coverage_drop":0}}]`)
	policy, _, err := decodeObservedRulesetPolicy(raw)
	if err != nil {
		t.Fatal(err)
	}
	p := (*policy.Rules)[0].Parameters
	if p.MinimumCoverage == nil || *p.MinimumCoverage != 80.5 || p.MaxCoverageDrop == nil || *p.MaxCoverageDrop != 0 {
		t.Fatal("lost coverage", p)
	}
}

func TestCoverageDoesNotAbsorbQualityNormalization(t *testing.T) {
	body := coverageBody(t, 80.5, 0)
	analysisParams(t, body, "code_quality")["severity"] = "FUTURE_QUALITY"
	data, _ := json.Marshal(body)
	var raw map[string]json.RawMessage
	json.Unmarshal(data, &raw)
	policy, records, err := decodeObservedRulesetPolicy(raw)
	if err != nil || records.QualitySeverity.RawValue != "FUTURE_QUALITY" {
		t.Fatal("quality provenance lost", records, err)
	}
	snapshot := rulesetCoverageSnapshot(&policy)
	if snapshot == nil || *snapshot != [2]float64{80.5, 0} {
		t.Fatal("coverage changed during quality normalization", snapshot)
	}
	if err := verifyReviewRulesetResponse(policy, raw); err == nil {
		t.Fatal("normalized quality passed strict readback alongside coverage")
	}
}

func TestRulesetCoverageAcquisitionDoesNotDefaultOptionalFields(t *testing.T) {
	for _, rule := range []string{
		`{"type":"code_coverage"}`,
		`{"type":"code_coverage","parameters":null}`,
		`{"type":"code_coverage","parameters":{}}`,
		`{"type":"code_coverage","parameters":{"minimum_coverage":0}}`,
		`{"type":"code_coverage","parameters":{"max_coverage_drop":0}}`,
		`{"type":"code_coverage","parameters":{"minimum_coverage":null,"max_coverage_drop":0}}`,
		`{"type":"code_coverage","parameters":{"minimum_coverage":0,"max_coverage_drop":null}}`,
		`{"type":"code_coverage","parameters":{"minimum_coverage":false,"max_coverage_drop":0}}`,
		`{"type":"code_coverage","parameters":{"minimum_coverage":0,"max_coverage_drop":"1"}}`,
		`{"type":"code_coverage","parameters":{"minimum_coverage":-1,"max_coverage_drop":0}}`,
		`{"type":"code_coverage","parameters":{"minimum_coverage":0,"max_coverage_drop":101}}`,
		`{"type":"code_coverage","parameters":{"minimum_coverage":0,"max_coverage_drop":0,"future":true}}`,
		`{"type":"code_coverage","parameters":{"minimum_coverage":0,"max_coverage_drop":0,"severity":"errors"}}`,
	} {
		var raw map[string]json.RawMessage
		json.Unmarshal([]byte(qualityRulesetJSON), &raw)
		raw["rules"] = json.RawMessage("[" + rule + "]")
		if _, _, err := decodeObservedRulesetPolicy(raw); err == nil {
			t.Fatal("incomplete coverage escaped", rule)
		}
		if _, err := decodeRulesetPolicy(raw); err == nil {
			t.Fatal("strict read accepted incomplete coverage", rule)
		}
	}
	for _, pair := range [][2]float64{{0, 0}, {100, 100}, {80.5, 2.25}} {
		var raw map[string]json.RawMessage
		json.Unmarshal([]byte(qualityRulesetJSON), &raw)
		raw["rules"] = json.RawMessage(fmt.Sprintf(`[{"type":"code_coverage","parameters":{"minimum_coverage":%g,"max_coverage_drop":%g}}]`, pair[0], pair[1]))
		if _, _, err := decodeObservedRulesetPolicy(raw); err != nil {
			t.Fatal(pair, err)
		}
		raw["target"] = json.RawMessage(`"tag"`)
		if _, _, err := decodeObservedRulesetPolicy(raw); err == nil {
			t.Fatal("observed tag coverage accepted")
		}
	}
}
