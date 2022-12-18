package core

import (
	"testing"
)

func TestInclude(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})
	runpfilePath := "../../testdata/runpfiles/include/01-02-03.yml"
	expectedUnits := []string{"unit-in-01", "unit-in-02", "unit-in-03"}
	rp, err := LoadRunpfileFromPath(runpfilePath)
	if err != nil {
		t.Errorf("Runppfile %s, load error %v", runpfilePath, err)
	}
	isActuallyValid, _ := IsRunpfileValid(rp)
	if !isActuallyValid {
		t.Errorf("Expected runpfile valid but it is not\n")
	}
	units := rp.Units
	if len(units) != len(expectedUnits) {
		t.Errorf("Expected units #%d but got #%d\n", len(expectedUnits), len(units))
	}
	for _, u := range expectedUnits {
		_, ok := rp.Units[u]
		if !ok {
			t.Errorf("Expected unit %s in %v\n", u, units)
		}
	}
}
