package settings

import (
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
)

// ForkingObservation is acquired only after the adapter proves applicability
// and complete organization policy visibility. Its zero value is not false.
type ForkingObservation struct {
	repositoryID, organizationID int64
	fullName                     string
	allowed, acquired            bool
}

func NewForkingObservation(repo repository.Repository, repositoryID, organizationID int64, fullName string, allowed bool) (ForkingObservation, error) {
	o := ForkingObservation{repositoryID, organizationID, fullName, allowed, true}
	if err := o.Validate(repo); err != nil {
		return ForkingObservation{}, err
	}
	return o, nil
}
func (o ForkingObservation) Validate(repo repository.Repository) error {
	if !o.acquired || o.repositoryID <= 0 || o.organizationID <= 0 || !strings.EqualFold(o.fullName, repo.Owner+"/"+repo.Name) {
		return fmt.Errorf("private forking observation is incomplete or does not match the target")
	}
	return nil
}
func (o ForkingObservation) Value() (bool, error) {
	if !o.acquired {
		return false, fmt.Errorf("private forking policy was not observed")
	}
	return o.allowed, nil
}

// WithAllowed preserves the observed identity for mutation response/readback checks.
func (o ForkingObservation) WithAllowed(value bool) ForkingObservation { o.allowed = value; return o }

var forkingBoolFields = [...]repositoryField[ForkingObservation, bool]{
	{name: "repository.allow_forking", current: func(o ForkingObservation) bool { return o.allowed },
		desired:    func(d RepositoryDeclaration) *bool { return d.AllowForking },
		setDesired: func(d *RepositoryDeclaration, v bool) { d.AllowForking = &v }},
}
