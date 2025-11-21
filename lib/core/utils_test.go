package core

import (
	"fmt"
	"path/filepath"
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
			"No units defined in Runpfile",
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

func TestResolveRunpfile(t *testing.T) {
	ui = CreateMainLogger(" ", 6, `TEST`, true, false)
	p := `../../examples/Runpfile-await-db.yml`
	rpp, err := ResolveRunpfilePath(p)
	if err != nil {
		t.Errorf("ResolveRunpfilePath %s, unexpected error %v", p, err)
	}
	if !filepath.IsAbs(rpp) {
		t.Errorf("ResolveRunpfilePath %s, expected absolute path but got %s", p, rpp)
	}
}

func TestRunpfileLoad(t *testing.T) {
	ui = CreateMainLogger(" ", 6, `TEST`, true, false)
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

func TestMultiError(t *testing.T) {
	expected := `e1
error 2
`
	var errors multiError
	errors = append(errors, fmt.Errorf(`e1`))
	errors = append(errors, fmt.Errorf(`error 2`))
	if errors.Error() != expected {
		t.Errorf(`multiError expected output "%s", got "%s"`, expected, errors.Error())
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
