package settings

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

const CodeQLConfigurationFieldCount = 4
const CodeQLConfigurationPath = "repository.settings.advanced_security.code_security.codeql_analysis"

type CodeQLRunner struct {
	Type  string `json:"type"`
	Label string `json:"label,omitempty"`
}
type CodeQLConfigurationDeclaration struct {
	Languages   *[]string
	QuerySuite  *string
	Runner      *CodeQLRunner
	ThreatModel *string
}

func (d CodeQLConfigurationDeclaration) Empty() bool {
	return d.Languages == nil && d.QuerySuite == nil && d.Runner == nil && d.ThreatModel == nil
}
func validateCodeQLLanguages(v []string) error {
	if len(v) == 0 {
		return fmt.Errorf("CodeQL languages must contain at least one language")
	}
	seen := map[string]bool{}
	for _, x := range v {
		if !slices.Contains([]string{"github_actions", "c_cpp", "csharp", "go", "java_kotlin", "javascript_typescript", "python", "ruby", "swift"}, x) || seen[x] {
			return fmt.Errorf("invalid or duplicate CodeQL language")
		}
		seen[x] = true
	}
	return nil
}
func (r CodeQLRunner) Validate() error {
	if r.Type == "standard_github_runner" && r.Label == "" {
		return nil
	}
	if r.Type == "labeled_runner" && strings.TrimSpace(r.Label) != "" && !strings.ContainsAny(r.Label, "\r\n") {
		return nil
	}
	return fmt.Errorf("CodeQL runner requires standard_github_runner without label, or labeled_runner with a nonblank label")
}
func (d CodeQLConfigurationDeclaration) Validate() error {
	if d.Languages != nil {
		if err := validateCodeQLLanguages(*d.Languages); err != nil {
			return err
		}
	}
	if d.QuerySuite != nil && *d.QuerySuite != "default" && *d.QuerySuite != "extended" {
		return fmt.Errorf("CodeQL query_suite must be default or extended")
	}
	if d.Runner != nil {
		if err := d.Runner.Validate(); err != nil {
			return err
		}
	}
	if d.ThreatModel != nil && *d.ThreatModel != "remote_sources" && *d.ThreatModel != "remote_and_local_sources" {
		return fmt.Errorf("invalid CodeQL threat_model")
	}
	return nil
}

type CodeQLConfigurationState struct {
	Languages   []string
	QuerySuite  string
	Runner      CodeQLRunner
	ThreatModel string
}

func (s CodeQLConfigurationState) declaration() CodeQLConfigurationDeclaration {
	return CodeQLConfigurationDeclaration{Languages: &s.Languages, QuerySuite: &s.QuerySuite, Runner: &s.Runner, ThreatModel: &s.ThreatModel}
}
func copyCodeQLState(s CodeQLConfigurationState) CodeQLConfigurationState {
	s.Languages = slices.Clone(s.Languages)
	slices.Sort(s.Languages)
	return s
}

type CodeQLConfigurationObservation struct {
	acquired bool
	state    CodeQLConfigurationState
}

func NewCodeQLConfigurationObservation(s CodeQLConfigurationState) (CodeQLConfigurationObservation, error) {
	if err := s.declaration().Validate(); err != nil {
		return CodeQLConfigurationObservation{}, err
	}
	return CodeQLConfigurationObservation{true, copyCodeQLState(s)}, nil
}
func (o CodeQLConfigurationObservation) State() (CodeQLConfigurationState, error) {
	if !o.acquired {
		return CodeQLConfigurationState{}, fmt.Errorf("CodeQL configuration was not acquired")
	}
	return copyCodeQLState(o.state), nil
}
func (o CodeQLConfigurationObservation) Declaration() (CodeQLConfigurationDeclaration, error) {
	s, err := o.State()
	if err != nil {
		return CodeQLConfigurationDeclaration{}, err
	}
	return s.declaration(), nil
}

type CodeQLConfigurationChange struct {
	field         string
	before, after any
}

func (c CodeQLConfigurationChange) Field() string { return c.field }
func copyCodeQLValue(v any) any {
	if a, ok := v.([]string); ok {
		return slices.Clone(a)
	}
	return v
}
func (c CodeQLConfigurationChange) Values() (any, any) {
	return copyCodeQLValue(c.before), copyCodeQLValue(c.after)
}
func NewCodeQLConfigurationChange(field string, before, after any) (CodeQLConfigurationChange, error) {
	normalize := func(v any) (any, error) {
		switch field {
		case "languages":
			a, ok := v.([]string)
			if !ok {
				return nil, fmt.Errorf("invalid CodeQL language value")
			}
			if err := validateCodeQLLanguages(a); err != nil {
				return nil, err
			}
			a = slices.Clone(a)
			slices.Sort(a)
			return a, nil
		case "runner":
			r, ok := v.(CodeQLRunner)
			if !ok {
				return nil, fmt.Errorf("invalid CodeQL runner value")
			}
			return r, r.Validate()
		case "query_suite", "threat_model":
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("invalid CodeQL choice")
			}
			d := CodeQLConfigurationDeclaration{}
			if field == "query_suite" {
				d.QuerySuite = &s
			} else {
				d.ThreatModel = &s
			}
			return s, d.Validate()
		default:
			return nil, fmt.Errorf("unknown CodeQL configuration field")
		}
	}
	b, err := normalize(before)
	if err != nil {
		return CodeQLConfigurationChange{}, err
	}
	a, err := normalize(after)
	if err != nil {
		return CodeQLConfigurationChange{}, err
	}
	if reflect.DeepEqual(a, b) {
		return CodeQLConfigurationChange{}, fmt.Errorf("equal CodeQL configuration change")
	}
	return CodeQLConfigurationChange{field, b, a}, nil
}
func CompareCodeQLConfiguration(o CodeQLConfigurationObservation, d CodeQLConfigurationDeclaration) (int, []CodeQLConfigurationChange, error) {
	if err := d.Validate(); err != nil {
		return 0, nil, err
	}
	s, err := o.State()
	if err != nil {
		return 0, nil, err
	}
	managed := 0
	changes := []CodeQLConfigurationChange{}
	add := func(field string, b, a any) error {
		managed++
		if reflect.DeepEqual(b, a) {
			return nil
		}
		c, err := NewCodeQLConfigurationChange(field, b, a)
		if err != nil {
			return err
		}
		changes = append(changes, c)
		return nil
	}
	if d.Languages != nil {
		a := slices.Clone(*d.Languages)
		slices.Sort(a)
		if err := add("languages", s.Languages, a); err != nil {
			return 0, nil, err
		}
	}
	if d.QuerySuite != nil {
		if err := add("query_suite", s.QuerySuite, *d.QuerySuite); err != nil {
			return 0, nil, err
		}
	}
	if d.Runner != nil {
		if err := add("runner", s.Runner, *d.Runner); err != nil {
			return 0, nil, err
		}
	}
	if d.ThreatModel != nil {
		if err := add("threat_model", s.ThreatModel, *d.ThreatModel); err != nil {
			return 0, nil, err
		}
	}
	return managed, changes, nil
}
func ExpectedCodeQLConfiguration(s CodeQLConfigurationState, d CodeQLConfigurationDeclaration) CodeQLConfigurationState {
	if d.Languages != nil {
		s.Languages = slices.Clone(*d.Languages)
	}
	if d.QuerySuite != nil {
		s.QuerySuite = *d.QuerySuite
	}
	if d.Runner != nil {
		s.Runner = *d.Runner
	}
	if d.ThreatModel != nil {
		s.ThreatModel = *d.ThreatModel
	}
	return copyCodeQLState(s)
}
