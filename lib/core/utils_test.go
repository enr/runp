package core

import (
	"fmt"
	"testing"
)

type testCase struct {
	path             string
	success          bool
	loadErrors       []string
	validationErrors []string
}

var executions = []testCase{
	{
		path:    "../../examples/Runpfile-await-db.yml",
		success: true,
		// loadErrors:       []string{},
		// validationErrors: []string{},
	},
	{
		path:    "../../testdata/runpfiles/validation-error-01.yml",
		success: false,
		validationErrors: []string{
			"No unit in Runpfile",
		},
	},
	{
		path:    "../../testdata/runpfiles/load-error-01.yml",
		success: false,
		loadErrors: []string{
			fmt.Sprintf(ErrFmtCreateProcess, `err`),
		},
	},
}

func TestRunpfileLoad(t *testing.T) {
	// LoadUI(clui.VerbosityLevelLow)
	i := ""
	for _, d := range executions {
		i = i + "."
		if d.success && len(d.loadErrors) > 0 {
			t.Errorf("Runppfile %s misconfiguration: expected success TRUE but load errors' number is not 0 \n", d.path)
		}
		if d.success && len(d.validationErrors) > 0 {
			t.Errorf("Runppfile %s misconfiguration: expected success TRUE but validation errors' number is not 0 \n", d.path)
		}

		rp, err := LoadRunpfileFromPath(d.path)
		if err != nil {
			actualLoadErrors := []error{err}
			compareErrors(d.path, actualLoadErrors, d.loadErrors, t)
			break
		}
		if len(d.loadErrors) > 0 {
			t.Errorf("Runppfile %s, expected load error %v not thrown\n", d.path, d.loadErrors)
		}

		isActuallyValid, errs := IsRunpfileValid(rp)
		if isActuallyValid != d.success {
			t.Errorf("Runppfile %s, expected valid %t but got %t\n", d.path, d.success, isActuallyValid)
		}
		if isActuallyValid {
			continue
		}
		compareErrors(d.path, errs, d.validationErrors, t)
	}
}

func compareErrors(rp string, actualErrors []error, expectedErrorMessages []string, t *testing.T) {
	if len(actualErrors) != len(expectedErrorMessages) {
		t.Errorf("Runppfile %s, expected errors size %d but got %d\n", rp, len(expectedErrorMessages), len(actualErrors))
	}
	for _, e := range expectedErrorMessages {
		for _, a := range actualErrors {
			if e == a.Error() {
				break
			}
			t.Errorf("Runppfile %s, expected error '%s' not found in actual errors %v\n", rp, e, actualErrors)
		}
	}
}
