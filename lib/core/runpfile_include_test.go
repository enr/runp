package core

import (
	"strings"
	"testing"
)

type includesTestCase struct {
	runpfilePath  string
	expectedUnits []string
	errors        []string
}

var okTestCases = []includesTestCase{
	{
		runpfilePath:  "../../testdata/runpfiles/include/01-02-03.yml",
		expectedUnits: []string{"unit-in-01", "unit-in-02", "unit-in-03"},
	},
}

var koTestCases = []includesTestCase{
	{
		runpfilePath: "../../testdata/runpfiles/include/04-02-05.yml",
		errors:       []string{"circular dependency", "02.yml"},
	},
	{
		runpfilePath: "../../testdata/runpfiles/include/06-07.yml",
		errors:       []string{"duplicate unit", "unit-in-06-and-07"},
	},
}

func TestIncludeOk(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})
	for _, it := range okTestCases {
		rp, err := LoadRunpfileFromPath(it.runpfilePath)
		if err != nil {
			t.Fatalf("Runppfile %s, load error %v", it.runpfilePath, err)
		}
		isActuallyValid, _ := IsRunpfileValid(rp)
		if !isActuallyValid {
			t.Errorf("Expected runpfile valid but it is not\n")
		}
		units := rp.Units
		if len(units) != len(it.expectedUnits) {
			t.Errorf("Expected units #%d but got #%d\n", len(it.expectedUnits), len(units))
		}
		for _, u := range it.expectedUnits {
			_, ok := rp.Units[u]
			if !ok {
				t.Errorf("Expected unit %s in %v\n", u, units)
			}
		}
	}

}

func TestIncludeKo(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})
	for _, it := range koTestCases {
		_, err := LoadRunpfileFromPath(it.runpfilePath)
		if err == nil {
			t.Fatalf("Runppfile %s: expected error but got nil", it.runpfilePath)
		}
		found := false
		for _, e := range it.errors {
			if strings.Contains(strings.ToLower(err.Error()), e) {
				found = true
			}
		}
		if !found {
			t.Errorf("Error %v doesn't contain any of %v\n", err, it.errors)
		}
	}

}
