package workspace

import (
	"reflect"
	"testing"
)

func TestUIConfigLabelsIdentityAndRoundTrip(t *testing.T) {
	input := `version: daiksud/gh-settings@v0
repository:
  labels:
    - id: 17
      name: renamed
      color: '#abcdef'
      description: ''
    - name: new
      color: '#123456'
`
	got, err := DecodeConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if got.Labels == nil || len(*got.Labels) != 2 {
		t.Fatalf("lost selected labels: %#v", got.Labels)
	}
	first, second := (*got.Labels)[0], (*got.Labels)[1]
	if first.ID == nil || *first.ID != 17 || first.Name != "renamed" || first.Color == nil || *first.Color != "#abcdef" || first.Description == nil || *first.Description != "" || second.ID != nil || second.Name != "new" || second.Color == nil || *second.Color != "#123456" || second.Description != nil {
		t.Fatalf("label identity or presence changed: %#v", *got.Labels)
	}
	data, err := EncodeConfig(got)
	if err != nil {
		t.Fatal(err)
	}
	again, err := DecodeConfig(data)
	if err != nil || !reflect.DeepEqual(again, got) {
		t.Fatalf("label round trip changed intent: %s, %v", data, err)
	}
}

func TestUIConfigLabelsCollectionPresence(t *testing.T) {
	for _, tc := range []struct {
		body string
		want bool
	}{{"{}", false}, {"{labels: []}", true}} {
		got, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: " + tc.body + "\n"))
		if err != nil || (got.Labels != nil) != tc.want || (got.Labels != nil && len(*got.Labels) != 0) {
			t.Fatalf("collection presence changed: %#v, %v", got.Labels, err)
		}
		encoded, err := EncodeConfig(got)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeConfig(encoded)
		if err != nil || (again.Labels != nil) != tc.want {
			t.Fatalf("encoded presence changed: %s, %v", encoded, err)
		}
	}
}

func TestUIConfigLabelsRejectsMistypedDeclarations(t *testing.T) {
	for _, value := range []string{
		"null", "{}", "[null]", "[{name: 2}]", "[{name: x, color: false}]",
		"[{name: x, description: null}]", "[{name: x, id: '17'}]",
		"[{name: x, id: 9223372036854775808}]", "[{name: x, future: true}]",
		"[{name: x, id: 17, id: 18}]",
	} {
		t.Run(value, func(t *testing.T) {
			input := "version: daiksud/gh-settings@v0\nrepository: {labels: " + value + "}\n"
			if _, err := DecodeConfig([]byte(input)); err == nil {
				t.Fatal("accepted unknown or mistyped label control")
			}
		})
	}
}
