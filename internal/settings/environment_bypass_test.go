package settings

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEnvironmentAdminBypassExplicitValuesAndUnknownOmission(t *testing.T) {
	for _, value := range []bool{false, true} {
		current := environmentPolicy(t, "production")
		current.CanAdminsBypass = &value
		observation, err := NewEnvironmentObservation([]EnvironmentState{{ID: 1, Policy: current}})
		if err != nil {
			t.Fatal(err)
		}
		desired, err := NewEnvironmentDeclarations([]EnvironmentDeclaration{current})
		if err != nil {
			t.Fatal(err)
		}
		if changes, err := ReconcileEnvironments(observation, &desired); err != nil || len(changes) != 0 {
			t.Fatal(changes, err)
		}
		opposite := !value
		request := current
		request.CanAdminsBypass = &opposite
		changed, _ := NewEnvironmentDeclarations([]EnvironmentDeclaration{request})
		changes, err := ReconcileEnvironments(observation, &changed)
		if err != nil || len(changes) != 1 || *changes[0].After().CanAdminsBypass != opposite || *changes[0].Before().CanAdminsBypass != value {
			t.Fatal(changes, err)
		}
		request.CanAdminsBypass = nil
		omitted, _ := NewEnvironmentDeclarations([]EnvironmentDeclaration{request})
		changes, err = ReconcileEnvironments(observation, &omitted)
		if err != nil || len(changes) != 1 || changes[0].After().CanAdminsBypass != nil || !EnvironmentHasAPIOmissions(*changes[0].After()) {
			t.Fatal(changes, err)
		}
		data, _ := json.Marshal(changes[0].After())
		if strings.Contains(string(data), "can_admins_bypass") {
			t.Fatal("omission acquired bypass default", string(data))
		}
		if _, err := NewEnvironmentObservation([]EnvironmentState{{ID: 1, Policy: request}}); err == nil {
			t.Fatal("missing bypass observation accepted")
		}
	}
}
