package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestOIDCImmutableExportIndependentEvidence(t *testing.T) {
	for _, mode := range []string{"first-unknown", "prior-unknown", "known-false", "known-true", "info-failure", "whole-failure", "leaf-check-failure"} {
		t.Run(mode, func(t *testing.T) {
			yes, no := true, false
			prior := appOIDC(false)
			prior.UseImmutableSubjectClaim = &yes
			var existing *settings.OIDCSubjectTemplate
			if mode != "first-unknown" {
				existing = &prior
			}
			var saved *settings.Declaration
			paths, checks := []string{}, []settings.OIDCSubjectTemplate{}
			reads := 0
			err := Export(ExportCheckout{Repository: checkoutRepository, Existing: settings.Declaration{OIDCSubject: existing}, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error { saved = &d; return nil }, nil
			}, Info: func(path string, _ error) error {
				if path == settings.OIDCSubjectPath || path == settings.OIDCTemplatePath || path == settings.OIDCImmutablePath {
					paths = append(paths, path)
					if mode == "info-failure" {
						return errors.New("diagnostic delivery")
					}
				}
				return nil
			}, Check: func(d settings.Declaration) error {
				if d.OIDCSubject == nil {
					return nil
				}
				checks = append(checks, settings.CloneOIDCSubject(*d.OIDCSubject))
				if mode == "leaf-check-failure" && d.OIDCSubject.UseDefaultTemplate == nil {
					return errors.New("leaf projection unavailable")
				}
				return settings.ValidateOIDCSubject(*d.OIDCSubject)
			}}, Reader{OIDCSubject: func(repository.Repository) (settings.OIDCSubjectObservation, error) {
				reads++
				if mode == "whole-failure" {
					return settings.OIDCSubjectObservation{}, errors.New("unavailable")
				}
				value := appOIDC(true)
				if mode == "known-false" || mode == "leaf-check-failure" {
					value.UseImmutableSubjectClaim = &no
				}
				if mode == "known-true" {
					value.UseImmutableSubjectClaim = &yes
				}
				return settings.NewOIDCSubjectObservation(value)
			}})
			if reads != 1 {
				t.Fatal("repeated raw observation", reads)
			}
			if mode == "info-failure" {
				if err == nil || saved != nil {
					t.Fatal("failed diagnostic still saved", err)
				}
				return
			}
			if err != nil || saved == nil || saved.OIDCSubject == nil {
				t.Fatal(saved, err)
			}
			if mode == "whole-failure" {
				if !reflect.DeepEqual(*saved.OIDCSubject, prior) || !reflect.DeepEqual(paths, []string{settings.OIDCSubjectPath}) {
					t.Fatal(saved, paths)
				}
				return
			}
			if !*saved.OIDCSubject.UseDefaultTemplate || saved.OIDCSubject.SubjectClaimTemplate != nil {
				t.Fatal("readable template stayed stale", saved)
			}
			if len(checks) == 0 || checks[0].UseImmutableSubjectClaim != nil {
				t.Fatal("retained intent certified as acquired", checks)
			}
			if mode == "first-unknown" {
				if saved.OIDCSubject.UseImmutableSubjectClaim != nil {
					t.Fatal("unknown fabricated")
				}
			} else if saved.OIDCSubject.UseImmutableSubjectClaim == nil || *saved.OIDCSubject.UseImmutableSubjectClaim != (mode != "known-false") {
				t.Fatal(saved.OIDCSubject)
			}
			unknown := mode != "known-false" && mode != "known-true"
			if unknown != reflect.DeepEqual(paths, []string{settings.OIDCImmutablePath}) {
				t.Fatal(paths)
			}
		})
	}
}

func TestOIDCImmutableFailureKeepsEarlierProgress(t *testing.T) {
	for _, earlier := range []bool{false, true} {
		no, description := false, "selected earlier setting"
		desired := settings.OIDCSubjectTemplate{UseImmutableSubjectClaim: &no}
		result, err := Apply(func() (settings.Declaration, error) {
			return settings.Declaration{OIDCSubject: &desired, Repository: settings.RepositoryDeclaration{Description: &description}}, nil
		}, checkoutRepository, Writer{Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) {
			return earlier, nil
		}, OIDCSubject: func(_ repository.Repository, original settings.OIDCSubjectTemplate) (bool, error) {
			if original.UseDefaultTemplate != nil || original.SubjectClaimTemplate != nil || original.UseImmutableSubjectClaim == nil || *original.UseImmutableSubjectClaim {
				t.Fatal("original partial intent replaced", original)
			}
			return false, errors.New("postwrite readback failed")
		}})
		want := ApplyFailed
		if earlier {
			want = ApplyPartialFailure
		}
		if err == nil || result.Outcome != want || result.Operations[len(result.Operations)-1].Status != OperationFailedUnknown {
			t.Fatal(earlier, result, err)
		}
	}
}
