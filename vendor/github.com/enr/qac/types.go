package qac

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// TestPlan represents the full set of tests on a program.
type TestPlan struct {
	Preconditions Preconditions   `yaml:"preconditions"`
	Specs         map[string]Spec `yaml:"specs"`
	specOrder     []string
}

// UnmarshalYAML preserves the declaration order of specs from the YAML source
// and rejects unknown fields.
func (tp *TestPlan) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: expected mapping for plan", value.Line)
	}
	known := map[string]bool{"preconditions": true, "specs": true}
	for i := 0; i < len(value.Content)-1; i += 2 {
		k := value.Content[i].Value
		if !known[k] {
			return fmt.Errorf("line %d: unknown field %q in plan", value.Content[i].Line, k)
		}
	}
	for i := 0; i < len(value.Content)-1; i += 2 {
		keyNode := value.Content[i]
		valNode := value.Content[i+1]
		switch keyNode.Value {
		case "preconditions":
			if err := strictDecodeNode(valNode, &tp.Preconditions); err != nil {
				return err
			}
		case "specs":
			if valNode.Kind != yaml.MappingNode {
				return fmt.Errorf("line %d: specs must be a mapping", valNode.Line)
			}
			tp.Specs = make(map[string]Spec, len(valNode.Content)/2)
			for j := 0; j < len(valNode.Content)-1; j += 2 {
				specKeyNode := valNode.Content[j]
				specValNode := valNode.Content[j+1]
				key := specKeyNode.Value
				var spec Spec
				if err := strictDecodeNode(specValNode, &spec); err != nil {
					return fmt.Errorf("spec %q: %w", key, err)
				}
				tp.Specs[key] = spec
				tp.specOrder = append(tp.specOrder, key)
			}
		}
	}
	return nil
}

// strictDecodeNode decodes a YAML node into v, rejecting unknown fields.
func strictDecodeNode(node *yaml.Node, v interface{}) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(node); err != nil {
		return err
	}
	enc.Close()
	dec := yaml.NewDecoder(&buf)
	dec.KnownFields(true)
	return dec.Decode(v)
}

// Spec is the single test.
type Spec struct {
	id            string
	Description   string        `yaml:"description"`
	Preconditions Preconditions `yaml:"preconditions"`
	Command       Command       `yaml:"command"`
	Expectations  Expectations  `yaml:"expectations"`
}

// ID returns the dynamically created identifier for a spec.
func (s Spec) ID() string {
	return s.id
}

// FileSystemAssertion is an assertion on files and directories.
type FileSystemAssertion struct {
	File      string        `yaml:"file"`
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
	Path         string        `yaml:"path"`
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
	WorkingDir string            `yaml:"working_dir"`
	Cli        string            `yaml:"cli"`
	Exe        string            `yaml:"exe"`
	Env        map[string]string `yaml:"env"`
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
	EqualsTo    *int `yaml:"equals_to"`
	GreaterThan *int `yaml:"greater_than"`
	LesserThan  *int `yaml:"lesser_than"`
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
	EndsWith     string   `yaml:"ends_with"`
	IsEmpty      *bool    `yaml:"is_empty"`
	ContainsAny  []string `yaml:"contains_any"`
	ContainsAll  []string `yaml:"contains_all"`
	ContainsNone []string `yaml:"contains_none"`
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
