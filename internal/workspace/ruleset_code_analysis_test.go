package workspace

import (
	"reflect"
	"strings"
	"testing"
)

const analysisConfig = `schemaVersion: 1
settings:
  rulesets:
    - id: 26024001
      name: default
      enforcement: active
      conditions: {ref_name: {include: ['~DEFAULT_BRANCH']}}
      rules:
        - type: code_scanning
          parameters:
            code_scanning_tools:
              - {tool: CodeQL, alerts_threshold: errors, security_alerts_threshold: high_or_higher}
        - type: code_quality
          parameters: {severity: errors}
`

func TestRulesetCodeAnalysisYAMLRoundTripAndStrictFields(t *testing.T) {
	d, err := DecodeConfig([]byte(analysisConfig))
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeConfig(d)
	if err != nil {
		t.Fatal(err)
	}
	round, err := DecodeConfig(data)
	if err != nil || !reflect.DeepEqual(d, round) {
		t.Fatal(string(data), err)
	}
	for _, pair := range [][2]string{
		{"severity: errors", "severity: null"}, {"severity: errors", "severity: true"}, {"severity: errors", "severity: none"},
		{"code_scanning_tools:", "Code_scanning_tools:"}, {"code_scanning_tools:\n              - {tool: CodeQL, alerts_threshold: errors, security_alerts_threshold: high_or_higher}", "code_scanning_tools: null"},
		{"tool: CodeQL", "tool: CodeQL, future: true"}, {"tool: CodeQL", "tool: 42"}, {"tool: CodeQL", "Tool: CodeQL"},
		{"alerts_threshold: errors", "alerts_threshold: null"}, {"alerts_threshold: errors", "alerts_threshold: warning"},
		{"security_alerts_threshold: high_or_higher", "security_alerts_threshold: high"},
	} {
		input := strings.Replace(analysisConfig, pair[0], pair[1], 1)
		if input == analysisConfig {
			t.Fatal(pair)
		}
		if _, err := DecodeConfig([]byte(input)); err == nil {
			t.Fatal("accepted invalid analysis YAML", pair)
		}
	}
}
