package settings

import (
	"fmt"
	"slices"
	"strings"
)

const PullRequestExemptionsPath = PullRequestLimitsPath + ".exemption_list"

// CanonicalPullRequestExemptionLogins validates an authoritative User login set.
// Identity is case-insensitive; the declaration does not accept IDs or bot names.
func CanonicalPullRequestExemptionLogins(logins []string) ([]string, error) {
	if len(logins) > 100 {
		return nil, fmt.Errorf("%s supports at most 100 users", PullRequestExemptionsPath)
	}
	result := make([]string, len(logins))
	seen := map[string]bool{}
	for i, login := range logins {
		if len(login) < 1 || len(login) > 39 || strings.HasPrefix(login, "-") || strings.HasSuffix(login, "-") || strings.Contains(login, "--") {
			return nil, fmt.Errorf("invalid pull request exemption User login %q", login)
		}
		for _, c := range login {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return nil, fmt.Errorf("invalid pull request exemption User login %q", login)
			}
		}
		canonical := strings.ToLower(login)
		if seen[canonical] {
			return nil, fmt.Errorf("duplicate pull request exemption User login %q", login)
		}
		seen[canonical] = true
		result[i] = canonical
	}
	slices.Sort(result)
	return result, nil
}

// PullRequestExemptionsDeclaration owns the complete list, including empty.
// Only a nil declaration pointer leaves the remote list unowned.
type PullRequestExemptionsDeclaration struct{ logins []string }

func NewPullRequestExemptionsDeclaration(logins []string) (PullRequestExemptionsDeclaration, error) {
	values, err := CanonicalPullRequestExemptionLogins(logins)
	if err != nil {
		return PullRequestExemptionsDeclaration{}, err
	}
	return PullRequestExemptionsDeclaration{logins: values}, nil
}
func (d PullRequestExemptionsDeclaration) Logins() []string { return append([]string{}, d.logins...) }

type PullRequestExemptionUser struct {
	ID    int64
	Login string
	Type  string
}
type PullRequestExemptionsObservation struct {
	acquired bool
	users    []PullRequestExemptionUser
}

func NewPullRequestExemptionsObservation(users []PullRequestExemptionUser) (PullRequestExemptionsObservation, error) {
	logins := make([]string, len(users))
	ids := map[int64]bool{}
	for i, user := range users {
		if user.ID <= 0 || ids[user.ID] || user.Type != "User" {
			return PullRequestExemptionsObservation{}, fmt.Errorf("unsupported or ambiguous pull request exemption User identity")
		}
		ids[user.ID] = true
		logins[i] = user.Login
	}
	if _, err := CanonicalPullRequestExemptionLogins(logins); err != nil {
		return PullRequestExemptionsObservation{}, err
	}
	values := append([]PullRequestExemptionUser{}, users...)
	for i := range values {
		values[i].Login = strings.ToLower(values[i].Login)
	}
	slices.SortFunc(values, func(a, b PullRequestExemptionUser) int { return strings.Compare(a.Login, b.Login) })
	return PullRequestExemptionsObservation{acquired: true, users: values}, nil
}
func (o PullRequestExemptionsObservation) Users() ([]PullRequestExemptionUser, error) {
	if !o.acquired {
		return nil, fmt.Errorf("pull request exemptions observation was not acquired")
	}
	return append([]PullRequestExemptionUser{}, o.users...), nil
}
func (o PullRequestExemptionsObservation) Logins() ([]string, error) {
	users, err := o.Users()
	if err != nil {
		return nil, err
	}
	result := make([]string, len(users))
	for i, user := range users {
		result[i] = user.Login
	}
	return result, nil
}

type PullRequestExemptionsChange struct{ before, after []string }

func NewPullRequestExemptionsChange(before, after []string) (PullRequestExemptionsChange, error) {
	b, err := CanonicalPullRequestExemptionLogins(before)
	if err != nil {
		return PullRequestExemptionsChange{}, err
	}
	a, err := CanonicalPullRequestExemptionLogins(after)
	if err != nil {
		return PullRequestExemptionsChange{}, err
	}
	if slices.Equal(b, a) {
		return PullRequestExemptionsChange{}, fmt.Errorf("unchanged pull request exemption list")
	}
	return PullRequestExemptionsChange{before: b, after: a}, nil
}
func (c PullRequestExemptionsChange) Values() ([]string, []string) {
	return append([]string{}, c.before...), append([]string{}, c.after...)
}
func (c PullRequestExemptionsChange) Field() string { return PullRequestExemptionsPath }
func ComparePullRequestExemptions(o PullRequestExemptionsObservation, d *PullRequestExemptionsDeclaration) (int, []PullRequestExemptionsChange, error) {
	if d == nil {
		return 0, nil, nil
	}
	before, err := o.Logins()
	if err != nil {
		return 0, nil, err
	}
	after := d.Logins()
	if slices.Equal(before, after) {
		return 1, nil, nil
	}
	change, err := NewPullRequestExemptionsChange(before, after)
	if err != nil {
		return 0, nil, err
	}
	return 1, []PullRequestExemptionsChange{change}, nil
}
