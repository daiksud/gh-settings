package workspace

import (
	"strings"
	"testing"
)

func TestPagesYAMLNullableDomainAndStrictSource(t *testing.T) {
	for _, tc := range []struct {
		yaml  string
		valid bool
	}{
		{"{}", true}, {"{cname: null}", true}, {"{cname:}", true}, {"{https_enforced: false}", true}, {"{cname: EXAMPLE.COM, build_type: legacy, source: {branch: main, path: /docs}}", true},
		{"null", false}, {"[]", false}, {"{cname: ''}", false}, {"{cname: 'null'}", false}, {"{https_enforced: null}", false}, {"{https_enforced: yes}", false}, {"{source: {branch: main}}", false}, {"{source: {branch: main, path: null}}", false}, {"{source: {}}", false}, {"{build_type: workflow, source: {branch: main, path: /}}", false}, {"{enabled: false}", false}, {"{public: false}", false}, {"{cname: null, cname: example.com}", false},
	} {
		t.Run(tc.yaml, func(t *testing.T) {
			d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings:\n  pages: " + tc.yaml))
			if (err == nil) != tc.valid {
				t.Fatal(err)
			}
			if !tc.valid {
				return
			}
			data, err := EncodeConfig(d)
			if err != nil {
				t.Fatal(err)
			}
			round, err := DecodeConfig(data)
			if err != nil {
				t.Fatal(string(data), err)
			}
			if d.Pages.CNAME != nil && d.Pages.CNAME.Value == nil && (round.Pages.CNAME == nil || round.Pages.CNAME.Value != nil || !strings.Contains(string(data), "cname: null")) {
				t.Fatal("clear intent lost", string(data))
			}
			if strings.Contains(tc.yaml, "false") && (round.Pages.HTTPSEnforced == nil || *round.Pages.HTTPSEnforced) {
				t.Fatal("false lost")
			}
		})
	}
}
