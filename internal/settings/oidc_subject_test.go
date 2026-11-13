package settings

import "testing"

func oidcValue(defaults bool, keys ...string) OIDCSubjectTemplate {
	v := OIDCSubjectTemplate{UseDefaultTemplate: &defaults}
	if !defaults {
		if keys == nil {
			keys = []string{}
		}
		v.SubjectClaimTemplate = &keys
	}
	return v
}
func TestOIDCSubjectUnionAndOrder(t *testing.T) {
	for _, v := range []OIDCSubjectTemplate{oidcValue(true), oidcValue(false), oidcValue(false, "repo", "context"), oidcValue(false, "Repo", "repo")} {
		if err := ValidateOIDCSubject(v); err != nil {
			t.Fatal(v, err)
		}
	}
	yes, no := true, false
	keys := []string{"repo"}
	for _, v := range []OIDCSubjectTemplate{{}, {SubjectClaimTemplate: &keys}, {UseDefaultTemplate: &no}, {UseDefaultTemplate: &yes, SubjectClaimTemplate: &keys}, oidcValue(false, ""), oidcValue(false, "repo", "repo"), oidcValue(false, "repo-name"), oidcValue(false, "日本語"), oidcValue(false, " repo")} {
		if ValidateOIDCSubject(v) == nil {
			t.Fatal("invalid union accepted", v)
		}
	}
	v := oidcValue(false, "repo", "context")
	o, err := NewOIDCSubjectObservation(v)
	if err != nil {
		t.Fatal(err)
	}
	(*v.SubjectClaimTemplate)[0] = "mutated"
	before, _ := o.Value()
	if (*before.SubjectClaimTemplate)[0] != "repo" {
		t.Fatal("aliased observation")
	}
	(*before.SubjectClaimTemplate)[0] = "again"
	after := oidcValue(false, "context", "repo")
	n, c, err := CompareOIDCSubject(o, &after)
	if err != nil || n != 1 || len(c) != 1 {
		t.Fatal(n, c, err)
	}
	b, a := c[0].Values()
	if (*b.SubjectClaimTemplate)[0] != "repo" || (*a.SubjectClaimTemplate)[0] != "context" {
		t.Fatal(b, a)
	}
	match := oidcValue(false, "repo", "context")
	if n, c, e := CompareOIDCSubject(o, &match); n != 1 || len(c) != 0 || e != nil {
		t.Fatal(n, c, e)
	}
	if _, _, e := CompareOIDCSubject(OIDCSubjectObservation{}, &match); e == nil {
		t.Fatal("unacquired accepted")
	}
}
