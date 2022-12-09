package core

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"

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
		// if !files.IsDir(unit.WorkingDir) {
		// 	errs = append(errs, errors.New("error in command <"+id+">: no working dir "+unit.WorkingDir))
		// }
	}
	return (len(errs) == 0), errs
}

// LoadRunpfileFromPath returns an Runpfile object reading file from path.
func LoadRunpfileFromPath(runpfilePath string) (*Runpfile, error) {
	data, err := ioutil.ReadFile(runpfilePath)
	if err != nil {
		return &Runpfile{}, err
	}
	rf, err := LoadRunpfileFromData(data)
	if err != nil {
		return &Runpfile{}, err
	}
	rf.Root, _ = filepath.Abs(filepath.Dir(runpfilePath))
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
		ui.Debugf("Resolved directory %s: from %s to %s", id, unit.Process().Dir(), wd)
		unit.Process().SetPreconditions(unit.ToPreconditions())
		unit.Process().SetDir(wd)
		unit.Process().SetID(unit.Name)
	}
	ui.WriteLinef("Runp Root %v", rf.Root)
	return rf, nil
}

func envAsArray(in map[string]string) (out []string) {
	out = []string{}
	for name, val := range in {
		out = append(out, fmt.Sprintf("%s=%s", name, os.ExpandEnv(val)))
	}
	return out
}

// LoadRunpfileFromData returns an Runpfile object reading []byte.
func LoadRunpfileFromData(data []byte) (*Runpfile, error) {
	rf := &Runpfile{}
	err := yaml.UnmarshalStrict(data, &rf)
	return rf, err
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
