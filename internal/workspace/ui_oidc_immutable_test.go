package workspace

import (
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/github"
)

func TestUIOIDCImmutableLiteral(t *testing.T) {
	input := "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {oidc: {subject_claim: {use_immutable_subject_claim: false}}}}}\n"
	v, err := DecodeConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := github.DeclarationFromUI(v); err != nil {
		t.Fatal(err)
	}
	b, err := EncodeConfig(v)
	if err != nil || !strings.Contains(string(b), "use_immutable_subject_claim: false") || strings.Contains(string(b), "use_default_template") {
		t.Fatal(string(b), err)
	}
}

func TestUIOIDCImmutableStrictUnion(t *testing.T) {
	for _, body := range []string{"use_immutable_subject_claim: null", "use_immutable_subject_claim: 'false'", "use_immutable_subject: true", "use_immutable_subject_claim: true, subject_claim_template: []", "use_immutable_subject_claim: true, use_default_template: false", "use_immutable_subject_claim: true, use_default_template: true, subject_claim_template: []"} {
		v, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {oidc: {subject_claim: {" + body + "}}}}}\n"))
		if err == nil {
			_, err = github.DeclarationFromUI(v)
		}
		if err == nil {
			t.Fatal("invalid immutable union accepted", body)
		}
	}
	for _, body := range []string{"use_immutable_subject_claim: true", "use_immutable_subject_claim: false, use_default_template: true", "use_immutable_subject_claim: false, use_default_template: false, subject_claim_template: []"} {
		v, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {oidc: {subject_claim: {" + body + "}}}}}\n"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := github.DeclarationFromUI(v); err != nil {
			t.Fatal(err)
		}
	}
}
