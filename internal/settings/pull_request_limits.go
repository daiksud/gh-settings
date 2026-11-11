package settings

import "fmt"

const PullRequestLimitsFieldCount = 3

const PullRequestLimitsPath = "repository.settings.interaction_limits.pull_request_limits"
const PullRequestLimitEnabledPath = PullRequestLimitsPath + ".limit_open_pull_requests_from_users_without_write_access"
const PullRequestLimitMaximumPath = PullRequestLimitsPath + ".maximum_open_pull_requests_per_user"
const PullRequestLimitDraftsPath = PullRequestLimitsPath + ".count_draft_pull_requests_toward_the_limit"

type PullRequestLimitsDeclaration struct {
	Enabled       *bool
	Maximum       *int
	IncludeDrafts *bool
}

func (d PullRequestLimitsDeclaration) Empty() bool {
	return d.Enabled == nil && d.Maximum == nil && d.IncludeDrafts == nil
}
func (d PullRequestLimitsDeclaration) Validate() error {
	if d.Maximum != nil && (*d.Maximum < 1 || *d.Maximum > 1000) {
		return fmt.Errorf("%s must be between 1 and 1000", PullRequestLimitMaximumPath)
	}
	return nil
}

type PullRequestLimitsValues struct {
	Enabled       bool
	Maximum       int
	IncludeDrafts bool
}
type PullRequestLimitsObservation struct {
	acquired bool
	values   PullRequestLimitsValues
}

func NewPullRequestLimitsObservation(v PullRequestLimitsValues) (PullRequestLimitsObservation, error) {
	if v.Maximum < 1 || v.Maximum > 1000 {
		return PullRequestLimitsObservation{}, fmt.Errorf("invalid pull request maximum")
	}
	return PullRequestLimitsObservation{true, v}, nil
}
func (o PullRequestLimitsObservation) Values() (PullRequestLimitsValues, error) {
	if !o.acquired {
		return PullRequestLimitsValues{}, fmt.Errorf("pull request limits observation was not acquired")
	}
	return o.values, nil
}

type PullRequestLimitsChange struct {
	field         string
	before, after any
}

func NewPullRequestLimitsChange(field string, before, after any) (PullRequestLimitsChange, error) {
	valid := false
	switch field {
	case PullRequestLimitEnabledPath, PullRequestLimitDraftsPath:
		_, a := before.(bool)
		_, b := after.(bool)
		valid = a && b
	case PullRequestLimitMaximumPath:
		a, ok := before.(int)
		b, ok2 := after.(int)
		valid = ok && ok2 && a >= 1 && a <= 1000 && b >= 1 && b <= 1000
	}
	if !valid || before == after {
		return PullRequestLimitsChange{}, fmt.Errorf("invalid pull request limits change")
	}
	return PullRequestLimitsChange{field, before, after}, nil
}
func (c PullRequestLimitsChange) Field() string      { return c.field }
func (c PullRequestLimitsChange) Values() (any, any) { return c.before, c.after }
func ComparePullRequestLimits(o PullRequestLimitsObservation, d PullRequestLimitsDeclaration) (int, []PullRequestLimitsChange, error) {
	if err := d.Validate(); err != nil {
		return 0, nil, err
	}
	if d.Empty() {
		return 0, nil, nil
	}
	v, err := o.Values()
	if err != nil {
		return 0, nil, err
	}
	managed := 0
	var changes []PullRequestLimitsChange
	add := func(path string, before, after any) {
		managed++
		if before != after {
			c, _ := NewPullRequestLimitsChange(path, before, after)
			changes = append(changes, c)
		}
	}
	if d.Enabled != nil {
		add(PullRequestLimitEnabledPath, v.Enabled, *d.Enabled)
	}
	if d.Maximum != nil {
		add(PullRequestLimitMaximumPath, v.Maximum, *d.Maximum)
	}
	if d.IncludeDrafts != nil {
		add(PullRequestLimitDraftsPath, v.IncludeDrafts, *d.IncludeDrafts)
	}
	return managed, changes, nil
}
