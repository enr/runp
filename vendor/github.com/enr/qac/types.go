package qac

import (
	"fmt"
	"runtime"
	"strings"
)

// TestPlan represents the full set of tests on a program.
type TestPlan struct {
	Preconditions Preconditions   `yaml:"preconditions"`
	Specs         map[string]Spec `yaml:"specs"`
}

// Spec is the single test.
type Spec struct {
	id            string
	Description   string        `yaml:"description"`
	Preconditions Preconditions `yaml:"preconditions"`
	Command       Command       `yaml:"command"`
	Expectations  Expectations  `yaml:"expectations"`
}

// ID returns the dinamically created identifier for a spec.
func (s Spec) ID() string {
	return s.id
}

// FileSystemAssertion is an assertion on files and directories.
type FileSystemAssertion struct {
	File string `yaml:"file"`
	// aggiunta a file
	Extension FileExtension `yaml:"ext"`
	Directory string        `yaml:"directory"`
	Exists    *bool         `yaml:"exists"`
	EqualsTo  string        `yaml:"equals_to"`
	// Only for files
	TextEqualsTo    string   `yaml:"text_equals_to"`
	ContainsAny     []string `yaml:"contains_any"`
	ContainsAll     []string `yaml:"contains_all"`
	ContainsExactly []string `yaml:"contains_exactly"`
}

// FileExtension is added as suffix to file assertions' path and command's exe values
// based on runtime.GOOS
type FileExtension struct {
	Windows string `yaml:"windows"`
	Unix    string `yaml:"unix"`
}

func (e FileExtension) isSet() bool {
	return len(e.Windows) > 0 || len(e.Unix) > 0
}
func (e FileExtension) get() string {
	if runtime.GOOS == "windows" {
		return e.Windows
	}
	return e.Unix
}

// FileAssertion is an assertion on a given file.
type FileAssertion struct {
	Path string `yaml:"path"`
	// aggiunta a path
	Extension    FileExtension `yaml:"ext"`
	Exists       bool          `yaml:"exists"`
	EqualsTo     string        `yaml:"equals_to"`
	TextEqualsTo string        `yaml:"text_equals_to"`
	ContainsAny  []string      `yaml:"contains_any"`
	ContainsAll  []string      `yaml:"contains_all"`
}

// DirectoryAssertion is an assertion on a given directory.
type DirectoryAssertion struct {
	Path            string   `yaml:"path"`
	Exists          bool     `yaml:"exists"`
	EqualsTo        string   `yaml:"equals_to"`
	ContainsAny     []string `yaml:"contains_any"`
	ContainsAll     []string `yaml:"contains_all"`
	ContainsExactly []string `yaml:"contains_exactly"`
}

// Preconditions represents the minimal requirements for a plan or a single spec to start.
type Preconditions struct {
	FileSystemAssertions []FileSystemAssertion `yaml:"fs"`
}

// Command represents the command under test.
type Command struct {
	WorkingDir string `yaml:"working_dir"`
	Cli        string `yaml:"cli"`
	Exe        string `yaml:"exe"`
	Env        map[string]string
	// added to exe
	Extension FileExtension `yaml:"ext"`
	Args      []string      `yaml:"args"`
}

func (c Command) String() string {
	fullCommand := c.Cli
	if fullCommand == "" {
		fullCommand = strings.TrimSpace(c.Exe + " " + strings.Join(c.Args, " "))
	}
	return fmt.Sprintf("%s# %s", c.WorkingDir, fullCommand)
}

// StatusAssertion represents an assertion on the status code returned from a command.
type StatusAssertion struct {
	EqualsTo    string `yaml:"equals_to"`
	GreaterThan string `yaml:"greater_than"`
	LesserThan  string `yaml:"lesser_than"`
}

// OutputAssertion is an assertion on the output of a command: namely standard output and standard error.
type OutputAssertion struct {
	// to identify as "stdout" or "stderr"
	id           string
	EqualsTo     string `yaml:"equals_to"`
	EqualsToFile string `yaml:"equals_to_file"`
	// output is trimmed
	StartsWith string `yaml:"starts_with"`
	// output is trimmed
	EndsWith    string   `yaml:"ends_with"`
	IsEmpty     *bool    `yaml:"is_empty"`
	ContainsAny []string `yaml:"contains_any"`
	ContainsAll []string `yaml:"contains_all"`
}

// OutputAssertions is the aggregate of stdout and stderr assertions.
type OutputAssertions struct {
	Stdout OutputAssertion `yaml:"stdout"`
	Stderr OutputAssertion `yaml:"stderr"`
}

// Expectations is the aggregate of the final assertions on the command executed.
type Expectations struct {
	StatusAssertion      StatusAssertion       `yaml:"status"`
	OutputAssertions     OutputAssertions      `yaml:"output"`
	FileSystemAssertions []FileSystemAssertion `yaml:"fs"`
}
