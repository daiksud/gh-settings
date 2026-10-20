package settings

import "fmt"

func ValidForkApprovalPolicy(v string) bool {
	return v == "first_time_contributors_new_to_github" || v == "first_time_contributors" || v == "all_external_contributors"
}

// PrivateForkDeclaration never supplies an omitted required PUT parameter.
type PrivateForkDeclaration struct {
	Run         *bool `yaml:"run_workflows_from_fork_pull_requests,omitempty" json:"run_workflows_from_fork_pull_requests,omitempty"`
	WriteTokens *bool `yaml:"send_write_tokens_to_workflows,omitempty" json:"send_write_tokens_to_workflows,omitempty"`
	Secrets     *bool `yaml:"send_secrets_and_variables,omitempty" json:"send_secrets_and_variables,omitempty"`
	Approval    *bool `yaml:"require_approval_for_fork_pr_workflows,omitempty" json:"require_approval_for_fork_pr_workflows,omitempty"`
}

func (d PrivateForkDeclaration) Values() [4]*bool {
	return [4]*bool{d.Run, d.WriteTokens, d.Secrets, d.Approval}
}
func (d PrivateForkDeclaration) Empty() bool { return d == (PrivateForkDeclaration{}) }
func PrivateForkKeys() [4]string {
	return [4]string{"run_workflows_from_fork_pull_requests", "send_write_tokens_to_workflows", "send_secrets_and_variables", "require_approval_for_fork_pr_workflows"}
}

type ForkApprovalObservation struct {
	acquired bool
	policy   string
}

func NewForkApprovalObservation(v string) (ForkApprovalObservation, error) {
	if !ValidForkApprovalPolicy(v) {
		return ForkApprovalObservation{}, fmt.Errorf("unsupported fork PR approval policy %q", v)
	}
	return ForkApprovalObservation{true, v}, nil
}
func (o ForkApprovalObservation) Value() (string, error) {
	if !o.acquired {
		return "", fmt.Errorf("fork approval observation was not acquired")
	}
	return o.policy, nil
}
func CompareForkApproval(o ForkApprovalObservation, d ActionsPolicyDeclaration) (int, []ActionsPolicyChange, error) {
	if err := d.Validate(); err != nil {
		return 0, nil, err
	}
	v, err := o.Value()
	if err != nil {
		return 0, nil, err
	}
	changes := []ActionsPolicyChange{}
	if d.ForkApprovalPolicy == nil {
		return 0, changes, nil
	}
	if v != *d.ForkApprovalPolicy {
		c, _ := NewActionsPolicyChange("actions.fork_pr_approval_policy", v, *d.ForkApprovalPolicy)
		changes = append(changes, c)
	}
	return 1, changes, nil
}

type PrivateForkObservation struct {
	acquired bool
	values   [4]bool
}

func NewPrivateForkObservation(values [4]bool) PrivateForkObservation {
	return PrivateForkObservation{true, values}
}
func (o PrivateForkObservation) Values() ([4]bool, error) {
	if !o.acquired {
		return [4]bool{}, fmt.Errorf("private fork observation was not acquired")
	}
	return o.values, nil
}
func (o PrivateForkObservation) Declaration() (PrivateForkDeclaration, error) {
	v, err := o.Values()
	if err != nil {
		return PrivateForkDeclaration{}, err
	}
	return PrivateForkDeclaration{&v[0], &v[1], &v[2], &v[3]}, nil
}

// ComparePrivateFork checks write prerequisites only when a write is necessary.
// Enabling execution must also make latent credential/approval choices explicit.
func ComparePrivateFork(o PrivateForkObservation, d PrivateForkDeclaration) (int, []ActionsPolicyChange, error) {
	v, err := o.Values()
	if err != nil {
		return 0, nil, err
	}
	managed := 0
	changes := []ActionsPolicyChange{}
	for i, p := range d.Values() {
		if p != nil {
			managed++
			if v[i] != *p {
				c, _ := NewActionsPolicyChange("actions.private_fork_workflows."+PrivateForkKeys()[i], v[i], *p)
				changes = append(changes, c)
			}
		}
	}
	if len(changes) > 0 && d.Run == nil {
		return 0, nil, fmt.Errorf("private fork changes require explicitly declared run_workflows_from_fork_pull_requests; undeclared required parameters are never replayed")
	}
	if len(changes) > 0 && !v[0] && d.Run != nil && *d.Run && managed != 4 {
		return 0, nil, fmt.Errorf("enabling private fork workflows requires all four values explicitly declared, including token/secrets and approval choices")
	}
	return managed, changes, nil
}
