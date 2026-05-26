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
	Include       []string          `yaml:"include"`
	Vars          map[string]string `yaml:"vars"`
	Setup         []Command         `yaml:"setup"`
	Teardown      []Command         `yaml:"teardown"`
	Preconditions Preconditions     `yaml:"preconditions"`
	Specs         map[string]Spec   `yaml:"specs"`
	specOrder     []string
}

// UnmarshalYAML preserves the declaration order of specs from the YAML source
// and rejects unknown fields.
func (tp *TestPlan) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: expected mapping for plan", value.Line)
	}
	known := map[string]bool{"include": true, "preconditions": true, "specs": true, "vars": true, "setup": true, "teardown": true}
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
		case "include":
			if err := strictDecodeNode(valNode, &tp.Include); err != nil {
				return err
			}
		case "vars":
			if err := strictDecodeNode(valNode, &tp.Vars); err != nil {
				return err
			}
		case "setup":
			if err := strictDecodeNode(valNode, &tp.Setup); err != nil {
				return err
			}
		case "teardown":
			if err := strictDecodeNode(valNode, &tp.Teardown); err != nil {
				return err
			}
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

// SkipCondition holds the conditions under which a spec is skipped.
type SkipCondition struct {
	// EnvSet skips the spec when the named environment variable is defined (regardless of its value).
	EnvSet string `yaml:"env_set"`
	// EnvValue skips the spec when any named variable equals its specified value.
	EnvValue map[string]string `yaml:"env_value"`
}

// Spec is the single test.
type Spec struct {
	id          string
	Description string        `yaml:"description"`
	Tags        []string      `yaml:"tags"`
	Skip        bool          `yaml:"skip"`
	SkipIf      SkipCondition `yaml:"skip_if"`
	// Retries is the number of additional attempts after the first failure.
	// Zero (default) means no retry.
	Retries       int           `yaml:"retries"`
	RetryDelay    string        `yaml:"retry_delay"`
	Setup         []Command     `yaml:"setup"`
	Teardown      []Command     `yaml:"teardown"`
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
	// Exists defaults to true when omitted (nil): the assertion checks the path exists.
	// Set explicitly to false to assert the path must not exist.
	Exists   *bool  `yaml:"exists"`
	EqualsTo string `yaml:"equals_to"`
	// Only for files
	TextEqualsTo     string   `yaml:"text_equals_to"`
	ContainsAny      []string `yaml:"contains_any"`
	ContainsAll      []string `yaml:"contains_all"`
	ContainsExactly  []string `yaml:"contains_exactly"`
	ContainsMatching string   `yaml:"contains_matching"`
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
	Path      string        `yaml:"path"`
	Extension FileExtension `yaml:"ext"`
	// Exists defaults to true (nil): check that the file exists.
	// Set to false to assert the file must not exist.
	Exists           *bool    `yaml:"exists"`
	EqualsTo         string   `yaml:"equals_to"`
	TextEqualsTo     string   `yaml:"text_equals_to"`
	ContainsAny      []string `yaml:"contains_any"`
	ContainsAll      []string `yaml:"contains_all"`
	ContainsMatching string   `yaml:"contains_matching"`
}

// DirectoryAssertion is an assertion on a given directory.
type DirectoryAssertion struct {
	Path string `yaml:"path"`
	// Exists defaults to true (nil): check that the directory exists.
	// Set to false to assert the directory must not exist.
	Exists          *bool    `yaml:"exists"`
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
//
// Use either cli or exe+args — setting both is a configuration error.
//
//   - cli runs the value through the system shell ($SHELL -c on Unix, cmd /C on
//     Windows). Shell features such as pipes (|), redirects (>), globs (*), and
//     variable expansion are available. Convenient for simple one-liners.
//     Avoid when any part of the command line comes from untrusted input:
//     shell injection is possible.
//
//   - exe + args starts the process directly without a shell. Arguments are
//     passed verbatim to the OS; no quoting, globbing, or expansion occurs.
//     Prefer this form when shell features are not needed, especially with
//     values that originate from user input or external data.
type Command struct {
	WorkingDir string `yaml:"working_dir"`
	// Cli is a shell command line. The system shell interprets it, so pipes,
	// redirects, and globs work. Mutually exclusive with exe.
	Cli string `yaml:"cli"`
	// Exe is the path or name of the executable. The process is started
	// directly without a shell. Mutually exclusive with cli.
	Exe string            `yaml:"exe"`
	Env map[string]string `yaml:"env"`
	// Extension is appended to Exe based on the runtime OS (e.g. ".exe" on Windows).
	Extension FileExtension `yaml:"ext"`
	// Args holds arguments passed directly to Exe. Only meaningful when Exe is set.
	Args []string `yaml:"args"`
	// Timeout is the maximum wall-clock time to wait; parsed by time.ParseDuration
	// (e.g. "30s", "1m"). Zero or empty means no timeout.
	Timeout string `yaml:"timeout"`
	// Stdin is an inline string piped to the command's standard input.
	Stdin string `yaml:"stdin"`
	// StdinFile is a path to a file whose contents are piped to standard input.
	StdinFile string `yaml:"stdin_file"`
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
	LessThan    *int `yaml:"less_than"`
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
	// ContainsLine requires at least one line to equal the given string exactly (after trimming).
	ContainsLine string `yaml:"contains_line"`
	// LineCount requires the output to have exactly N non-empty lines.
	LineCount *int `yaml:"line_count"`
	// LineCountGte requires the output to have at least N non-empty lines.
	LineCountGte *int `yaml:"line_count_gte"`
	// Matches requires the entire trimmed output to match the regular expression.
	Matches string `yaml:"matches"`
	// NotMatches requires the entire trimmed output NOT to match the regular expression.
	NotMatches string `yaml:"not_matches"`
}

// OutputAssertions is the aggregate of stdout and stderr assertions.
type OutputAssertions struct {
	Stdout OutputAssertion `yaml:"stdout"`
	Stderr OutputAssertion `yaml:"stderr"`
}

// DurationAssertion asserts that the command's wall-clock execution time is
// within the specified bounds.  Both fields accept any string accepted by
// time.ParseDuration (e.g. "2s", "500ms", "1m30s").
type DurationAssertion struct {
	Max string `yaml:"max"`
	Min string `yaml:"min"`
}

// Expectations is the aggregate of the final assertions on the command executed.
type Expectations struct {
	StatusAssertion      StatusAssertion       `yaml:"status"`
	OutputAssertions     OutputAssertions      `yaml:"output"`
	FileSystemAssertions []FileSystemAssertion `yaml:"fs"`
	DurationAssertion    DurationAssertion     `yaml:"duration"`
}
