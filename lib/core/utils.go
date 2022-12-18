package core

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v3"

	"github.com/enr/go-files/files"
)

// ErrFmtCreateProcess format used for error in process creation.
const ErrFmtCreateProcess = "Unable to create Process for %s. Check configuration: missing Host, SSHTunnel or Container key?"

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
	ui.Debugf("Using configuration file %s", configurationFile)
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

// IsRunpfileValid returns a boolean rapresenting if any Runpfile value is valid and a list of errors.
func IsRunpfileValid(runpfile *Runpfile) (bool, []error) {
	errs := []error{}
	if len(runpfile.Units) == 0 {
		errs = append(errs, errors.New("No unit in Runpfile"))
	}
	for id, unit := range runpfile.Units {
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
			errs = append(errs, errors.New("Host, Container and SSHTunnel in "+id))
		}
		if len(modes) < 1 {
			errs = append(errs, errors.New("Host, SSHTunnel or Container missing in "+id))
		}
	}
	return (len(errs) == 0), errs
}

type runpfileSource struct {
	path       string
	importedBy string
}

// LoadRunpfileFromPath returns an Runpfile object reading file from path.
func LoadRunpfileFromPath(runpfilePath string) (*Runpfile, error) {
	rps := runpfileSource{
		path: runpfilePath,
	}
	visited := make(map[string]runpfileSource)
	return loadRunpfileFromPath(rps, visited)
}

func loadRunpfileFromPath(runpfile runpfileSource, visited map[string]runpfileSource) (*Runpfile, error) {
	if val, ok := visited[runpfile.path]; ok {
		return nil, fmt.Errorf(`Circular dependency for runpfile %s required from %s and %s`, runpfile.path, runpfile.importedBy, val.importedBy)
	}
	visited[runpfile.path] = runpfile
	data, err := ioutil.ReadFile(runpfile.path)
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
		if unit.Name == "" {
			unit.Name = id
		}
		if unit.Process() == nil {
			return nil, errors.New(fmt.Sprintf(ErrFmtCreateProcess, id))
		}
		wd, fail := resolveWorkingDir(rf, unit)
		if fail != nil {
			ui.WriteLinef("Failed resolving working directory %s:%s %v", unit.Name, unit.Process().Dir(), fail)
			return nil, fail
		}
		ui.Debugf(`Resolved directory %s: from "%s" to "%s"`, id, unit.Process().Dir(), wd)
		unit.Process().SetPreconditions(unit.Preconditions)
		unit.Process().SetDir(wd)
		unit.Process().SetID(unit.Name)
	}
	for _, inc := range rf.Include {
		rpp := filepath.ToSlash(filepath.Join(rf.Root, inc))
		ui.Debugf("%s include: %s", runpfile.path, rpp)
		if !files.Exists(rpp) {
			return nil, fmt.Errorf(`Imported Runpfile "%s" not found`, rpp)
		}
		source := runpfileSource{
			path:       rpp,
			importedBy: runpfile.path,
		}
		if rf.Units == nil {
			rf.Units = map[string]*RunpUnit{}
		}
		included, err := loadRunpfileFromPath(source, visited)
		if err != nil {
			return nil, err
		}
		for k, v := range included.Units {
			if _, ok := rf.Units[k]; ok {
				return nil, fmt.Errorf(`Duplicate unit id "%s"`, k)
			}
			rf.Units[k] = v
		}
	}
	if runpfile.importedBy == "" {
		ui.WriteLinef("Runp Root %v", rf.Root)
	}
	return rf, nil
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
		out = append(out, fmt.Sprintf("%s=%s", name, os.ExpandEnv(val)))
	}
	return out
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
