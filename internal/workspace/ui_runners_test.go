package workspace

import (
	"github.com/daiksud/gh-settings/internal/github"
	"strings"
	"testing"
)

func TestUIRunnerLabelsRoundTrip(t *testing.T) {
	input := "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {runners: [{id: 23, custom_labels: [GPU, accelerated]}, {id: 24, custom_labels: []}]}}}\n"
	c, err := DecodeConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := github.DeclarationFromUI(c); err != nil {
		t.Fatal(err)
	}
	data, err := EncodeConfig(c)
	if err != nil || !strings.Contains(string(data), "custom_labels: []") || !strings.Contains(string(data), "GPU") {
		t.Fatal(string(data), err)
	}
	if _, err := DecodeConfig(data); err != nil {
		t.Fatal(err)
	}
}

func TestUIRunnerLabelsBoundary(t *testing.T) {
	for _, body := range []string{"null", "{}", "[null]", "[{id: 0, custom_labels: []}]", "[{id: 1}]", "[{id: null, custom_labels: []}]", "[{id: 1, custom_labels: null}]", "[{id: 1, custom_labels: [null]}]", "[{id: 1, custom_labels: [2]}]", "[{id: 1, custom_labels: [gpu, GPU]}]", "[{id: 1, custom_labels: [], name: nope}]", "[{id: 1.2, custom_labels: []}]", "[{id: '1', custom_labels: []}]"} {
		c, e := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {runners: " + body + "}}}\n"))
		if e == nil {
			_, e = github.DeclarationFromUI(c)
		}
		if e == nil {
			t.Fatal("accepted", body)
		}
	}
	c, e := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {runners: []}}}\n"))
	if e != nil {
		t.Fatal(e)
	}
	d, e := github.DeclarationFromUI(c)
	if e != nil || !d.Empty() || d.Runners == nil {
		t.Fatal(d, e)
	}
}
