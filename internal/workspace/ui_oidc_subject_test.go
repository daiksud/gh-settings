package workspace

import (
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/github"
)

func TestUIOIDCSubjectTemplateRoundtrip(t *testing.T) {
	for _, template := range []string{
		"use_default_template: true",
		"use_default_template: false, subject_claim_template: [repo, context, job_workflow_ref]",
	} {
		input := "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {oidc: {subject_claim: {" + template + "}}}}}\n"
		value, err := DecodeConfig([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		data, err := EncodeConfig(value)
		if err != nil || !strings.Contains(string(data), "use_default_template:") {
			t.Fatal(string(data), err)
		}
		if strings.Contains(template, "job_workflow_ref") {
			repo, context, workflow := strings.Index(string(data), "- repo"), strings.Index(string(data), "- context"), strings.Index(string(data), "- job_workflow_ref")
			if repo < 0 || context <= repo || workflow <= context {
				t.Fatal("claim order was not preserved", string(data))
			}
		}
		if _, err := DecodeConfig(data); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUIOIDCSubjectTemplateBoundary(t *testing.T) {
	for _, body := range []string{"use_default_template: false", "subject_claim_template: []", "use_default_template: true, subject_claim_template: []", "use_default_template: null", "use_default_template: 'false', subject_claim_template: []", "use_default_template: false, subject_claim_template: null", "use_default_template: false, subject_claim_template: [repo, repo]", "use_default_template: false, subject_claim_template: [repo-name]", "use_default_template: false, subject_claim_template: [1]", "use_default_template: false, subject_claim_template: [null]", "use_default_template: true, use_immutable_subject_claim: true"} {
		input := "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {oidc: {subject_claim: {" + body + "}}}}}\n"
		value, err := DecodeConfig([]byte(input))
		if err == nil {
			_, err = github.DeclarationFromUI(value)
		}
		if err == nil {
			t.Fatal("accepted", body)
		}
	}
	for _, body := range []string{"{}", "{subject_claim: {}}", "{subject_claim: {use_default_template: false, subject_claim_template: []}}"} {
		input := "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {oidc: " + body + "}}}\n"
		c, err := DecodeConfig([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		declaration, err := github.DeclarationFromUI(c)
		if err != nil || declaration.Empty() == strings.Contains(body, "false") {
			t.Fatal("omission versus explicit union", declaration, err)
		}
		b, err := EncodeConfig(c)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(body, "false") && !strings.Contains(string(b), "subject_claim_template: []") {
			t.Fatal(string(b))
		}
	}
}
