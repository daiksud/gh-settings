package settings

import "testing"

func TestRepositoryVariableEmptyDesiredValueRejected(t *testing.T) {
	if _, err := NewVariableDeclarations([]RepositoryVariable{{Name: "DISPLAY_NAME", Value: ""}}); err == nil {
		t.Fatal("accepted empty desired value rejected by GitHub API")
	}
	if _, err := NewVariableChange("DISPLAY_NAME", nil, ""); err == nil {
		t.Fatal("accepted empty desired report value")
	}
	observed, err := NewVariableObservation(RepositoryVariable{Name: "DISPLAY_NAME", Value: ""})
	value, valueErr := observed.Value()
	if err != nil || valueErr != nil || value == nil || value.Value != "" {
		t.Fatal("lost acquired empty observation", err, valueErr)
	}
}
