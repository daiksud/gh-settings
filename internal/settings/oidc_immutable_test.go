package settings

import "testing"

func TestOIDCImmutableSelectionEvidenceAndReport(t *testing.T) {
	yes, no := true, false
	for _, value := range []bool{false, true} {
		d := OIDCSubjectTemplate{UseImmutableSubjectClaim: &value}
		if err := ValidateOIDCSubject(d); err != nil {
			t.Fatal(err)
		}
		v := oidcValue(false, "context", "repo")
		v.UseImmutableSubjectClaim = &value
		o, err := NewOIDCSubjectObservation(v)
		if err != nil {
			t.Fatal(err)
		}
		if n, changes, err := CompareOIDCSubject(o, &d); err != nil || n != 1 || len(changes) != 0 {
			t.Fatal(n, changes, err)
		}
		copy := CloneOIDCSubject(v)
		*copy.UseImmutableSubjectClaim = !value
		if *v.UseImmutableSubjectClaim != value {
			t.Fatal("immutable alias")
		}
		changed := OIDCSubjectTemplate{UseImmutableSubjectClaim: copy.UseImmutableSubjectClaim}
		_, changes, err := CompareOIDCSubject(o, &changed)
		if err != nil || len(changes) != 1 {
			t.Fatal(changes, err)
		}
		before, after := changes[0].Values()
		if before.UseDefaultTemplate != nil || before.SubjectClaimTemplate != nil || after.UseDefaultTemplate != nil {
			t.Fatal("replayed template became user intent", before, after)
		}
	}
	o, _ := NewOIDCSubjectObservation(oidcValue(true))
	if _, _, err := CompareOIDCSubject(o, &OIDCSubjectTemplate{UseImmutableSubjectClaim: &no}); err == nil {
		t.Fatal("unknown immutable became false")
	}
	matching := oidcValue(true)
	if _, changes, err := CompareOIDCSubject(o, &matching); err != nil || len(changes) != 0 {
		t.Fatal("matching template required unrelated evidence", err)
	}
	if _, err := NewOIDCSubjectObservation(OIDCSubjectTemplate{UseImmutableSubjectClaim: &yes}); err == nil {
		t.Fatal("incomplete observation accepted")
	}
	if _, err := NewOIDCSubjectChange(OIDCSubjectTemplate{UseDefaultTemplate: &yes, UseImmutableSubjectClaim: &yes}, OIDCSubjectTemplate{UseImmutableSubjectClaim: &no}); err == nil {
		t.Fatal("mismatched report units accepted")
	}
}
