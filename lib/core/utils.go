package core

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"

	"errors"

	yaml "gopkg.in/yaml.v3"

	"github.com/enr/go-files/files"
)

// ErrFmtCreateProcess format used for error in process creation.
const ErrFmtCreateProcess = "Unable to create process for unit %s: exactly one of Host, SSHTunnel, or Container must be defined"

var (
	ui                         Logger
	processLoggerConfiguration LoggerConfig
)

// ConfigureUI allows to the main package to set main logger instance and configure the process logger instances.
func ConfigureUI(mainLogger Logger, processLoggerConfig LoggerConfig) {
	ui = mainLogger
	processLoggerConfiguration = processLoggerConfig
}

// ResolveRunpfilePath Returns the path to the Runpfile and error
func ResolveRunpfilePath(rp string) (string, error) {
	configurationFile, err := normalizePath(rp)
	if err != nil {
		return configurationFile, err
	}
	ui.Debugf("Resolved configuration file path: %s", configurationFile)
	if !files.Exists(configurationFile) {
		return configurationFile, errors.New("Runpfile not found: " + configurationFile)
	}
	return configurationFile, nil
}

func normalizePath(dirpath string) (string, error) {
	p, err := filepath.Abs(dirpath)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(filepath.ToSlash(p), "/"), nil
}

// IsRunpfileValid returns a boolean indicating whether the Runpfile is valid and a list of validation errors.
func IsRunpfileValid(runpfile *Runpfile) (bool, []error) {
	errs := []error{}
	if len(runpfile.Units) == 0 {
		errs = append(errs, errors.New("No units defined in Runpfile"))
	}
	for id, unit := range runpfile.Units {
		if err := validateUnitName(id); err != nil {
			errs = append(errs, err)
		}
		if unit != nil && unit.Name != "" && unit.Name != id {
			if err := validateUnitName(unit.Name); err != nil {
				errs = append(errs, err)
			}
		}
		if unit == nil {
			errs = append(errs, errors.New("Unit "+id+" must define exactly one process type: Host, SSHTunnel, or Container"))
			continue
		}
		modes := []string{}
		if unit.Container != nil {
			modes = append(modes, "container")
		}
		if unit.Host != nil {
			modes = append(modes, "host")
		}
		if unit.SSHTunnel != nil {
			modes = append(modes, "ssh_tunnel")
		}
		if len(modes) > 1 {
			errs = append(errs, errors.New("Unit "+id+" cannot have multiple process types: Host, Container, and SSHTunnel are mutually exclusive"))
		}
		if len(modes) < 1 {
			errs = append(errs, errors.New("Unit "+id+" must define exactly one process type: Host, SSHTunnel, or Container"))
		}
	}
	return (len(errs) == 0), errs
}

// validateUnitName rejects unit names that cannot be used safely as file
// names (PID files) or container names.
func validateUnitName(name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("invalid unit name %q: it must not be empty, \".\", \"..\" or contain path separators", name)
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("invalid unit name %q: it must not contain control characters", name)
		}
	}
	return nil
}

type runpfileSource struct {
	path       string
	importedBy string
	chain      []string // ordered list of paths from root to this file's importer
}

// LoadRunpfileFromPath returns an Runpfile object reading file from path.
func LoadRunpfileFromPath(runpfilePath string) (*Runpfile, error) {
	rps := runpfileSource{
		path: runpfilePath,
	}
	visited := make(map[string]runpfileSource)
	return loadRunpfileFromPath(rps, visited, false)
}

// LoadRunpfileForValidation loads a Runpfile without failing on units that
// lack a process block — allowing ValidateRunpfile to report all errors at once.
func LoadRunpfileForValidation(runpfilePath string) (*Runpfile, error) {
	rps := runpfileSource{
		path: runpfilePath,
	}
	visited := make(map[string]runpfileSource)
	return loadRunpfileFromPath(rps, visited, true)
}

func circularImportError(runpfile runpfileSource) error {
	cycleStart := -1
	for i, p := range runpfile.chain {
		if p == runpfile.path {
			cycleStart = i
			break
		}
	}
	var cycleNodes []string
	if cycleStart >= 0 {
		cycleNodes = append(runpfile.chain[cycleStart:], runpfile.path)
	} else {
		cycleNodes = append(runpfile.chain, runpfile.path)
	}
	return fmt.Errorf("circular dependency detected: %s", strings.Join(cycleNodes, " -> "))
}

func loadRunpfileFromPath(runpfile runpfileSource, visited map[string]runpfileSource, forValidation bool) (*Runpfile, error) {
	// visited holds the files on the current include chain only: a file
	// included twice through different branches (A->B->D, A->C->D) is not a
	// cycle; it is reported as duplicate units when merging.
	key := runpfile.path
	if abs, err := filepath.Abs(key); err == nil {
		key = filepath.ToSlash(abs)
	}
	if _, ok := visited[key]; ok {
		return nil, circularImportError(runpfile)
	}
	visited[key] = runpfile
	defer delete(visited, key)
	data, err := os.ReadFile(runpfile.path)
	if err != nil {
		return nil, err
	}
	rf, err := loadRunpfileFromData(data)
	if err != nil {
		return nil, err
	}
	rf.Root, err = filepath.Abs(filepath.Dir(runpfile.path))
	if err != nil {
		return nil, err
	}
	for id, unit := range rf.Units {
		unit.vars = rf.Vars
		unit.root = rf.Root
		if unit.Name == "" {
			unit.Name = id
		}
		if unit.Process() == nil {
			if !forValidation {
				return nil, fmt.Errorf(ErrFmtCreateProcess, id)
			}
			continue
		}
		wd, fail := resolveWorkingDir(rf, unit)
		if fail != nil {
			ui.WriteLinef("Failed to resolve working directory for unit %s (path: %s): %v", unit.Name, unit.Process().Dir(), fail)
			return nil, fail
		}
		ui.Debugf("Resolved working directory for unit %s: %s -> %s", id, unit.Process().Dir(), wd)
		unit.Process().SetPreconditions(unit.Preconditions)
		unit.Process().SetDir(wd)
		unit.Process().SetID(unit.Name)
	}
	for _, inc := range rf.Include {
		err = merge(runpfile, rf, inc, visited, forValidation)
		if err != nil {
			return nil, err
		}
	}
	if runpfile.importedBy == "" {
		ui.WriteLinef("Runpfile root directory: %s", rf.Root)
		project := ProjectID(rf.Root)
		for _, unit := range rf.Units {
			if unit.Container != nil {
				unit.Container.project = project
			}
		}
	}
	return rf, nil
}

func merge(runpfile runpfileSource, rf *Runpfile, inc string, visited map[string]runpfileSource, forValidation bool) error {
	rpp := filepath.ToSlash(filepath.Join(rf.Root, inc))
	ui.Debugf("Including Runpfile from %s: %s", runpfile.path, rpp)
	if !files.Exists(rpp) {
		return fmt.Errorf("included Runpfile not found: %s", rpp)
	}
	newChain := make([]string, len(runpfile.chain)+1)
	copy(newChain, runpfile.chain)
	newChain[len(runpfile.chain)] = runpfile.path
	source := runpfileSource{
		path:       rpp,
		importedBy: runpfile.path,
		chain:      newChain,
	}
	if rf.Units == nil {
		rf.Units = map[string]*RunpUnit{}
	}
	included, err := loadRunpfileFromPath(source, visited, forValidation)
	if err != nil {
		return err
	}
	for k, v := range included.Units {
		if _, ok := rf.Units[k]; ok {
			return fmt.Errorf("duplicate unit identifier: %s", k)
		}
		rf.Units[k] = v
	}
	return nil
}

func sliceContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func envAsArray(in map[string]string) (out []string) {
	out = []string{}
	for name, val := range in {
		out = append(out, fmt.Sprintf("%s=%s", name, expandEnvValue(val)))
	}
	return out
}

// expandEnvValue expands $VAR and ${VAR} references to the environment of
// runp; "$$" is an escape for a literal "$" (e.g. in passwords).
func expandEnvValue(s string) string {
	if !strings.Contains(s, "$") {
		return s
	}
	parts := strings.Split(s, "$$")
	for i, part := range parts {
		parts[i] = os.ExpandEnv(part)
	}
	return strings.Join(parts, "$")
}

func loadRunpfileFromData(data []byte) (*Runpfile, error) {
	rf := &Runpfile{}
	err := unmarshalStrict(data, &rf)
	return rf, err
}

func unmarshalStrict(data []byte, out interface{}) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil && err != io.EOF {
		return err
	}
	return nil
}

func resolveWorkingDir(rf *Runpfile, unit *RunpUnit) (string, error) {
	process := unit.Process()
	pd := process.Dir()
	if unit.SkipDirResolution() {
		return pd, nil
	}
	if pd == "" {
		return rf.Root, nil
	}
	if varsRegexp.MatchString(pd) {
		// Resolved when the unit starts, once the final vars are known.
		return pd, nil
	}
	return resolvePath(pd, rf.Root)
}

func resolvePath(pd string, root string) (string, error) {
	pwd := os.ExpandEnv(pd)
	if strings.HasPrefix(pwd, "~") {
		home, err := homedir.Dir()
		if err != nil {
			return "", err
		}
		relpath := strings.TrimPrefix(pwd, "~")
		return filepath.FromSlash(path.Join(home, relpath)), nil
	}
	if filepath.IsAbs(pwd) {
		return filepath.FromSlash(pwd), nil
	}
	return filepath.Abs(path.Join(filepath.FromSlash(root), filepath.FromSlash(pwd)))
}

type multiError []error

func (e multiError) Error() string {
	var sb strings.Builder
	for _, err := range e {
		sb.WriteString(err.Error())
		sb.WriteString("\n")
	}
	return sb.String()
}

func cmd(commandLine string) (*exec.Cmd, error) {
	shell := defaultShell()
	exe := shell.Path
	args := shell.Args
	args = append(args, commandLine)
	return exec.Command(exe, args...), nil
}

// splitCommandLine splits s into words following POSIX shell quoting rules
// (single quotes, double quotes, backslash escapes) without performing any
// expansion. It is used to turn a command string into an argument list that
// can be executed without a shell.
func splitCommandLine(s string) ([]string, error) {
	var words []string
	var cur strings.Builder
	inWord := false
	const (
		none = iota
		single
		double
	)
	quote := none
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch quote {
		case single:
			if r == '\'' {
				quote = none
			} else {
				cur.WriteRune(r)
			}
		case double:
			switch {
			case r == '"':
				quote = none
			case r == '\\' && i+1 < len(runes) && strings.ContainsRune("\"\\$`\n", runes[i+1]):
				i++
				if runes[i] != '\n' {
					cur.WriteRune(runes[i])
				}
			default:
				cur.WriteRune(r)
			}
		default:
			switch {
			case r == ' ' || r == '\t' || r == '\n' || r == '\r':
				if inWord {
					words = append(words, cur.String())
					cur.Reset()
					inWord = false
				}
			case r == '\'':
				quote, inWord = single, true
			case r == '"':
				quote, inWord = double, true
			case r == '\\':
				inWord = true
				if i+1 < len(runes) {
					i++
					if runes[i] != '\n' {
						cur.WriteRune(runes[i])
					}
				}
			default:
				inWord = true
				cur.WriteRune(r)
			}
		}
	}
	if quote != none {
		return nil, errors.New("unterminated quote in command")
	}
	if inWord {
		words = append(words, cur.String())
	}
	return words, nil
}
