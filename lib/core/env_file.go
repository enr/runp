package core

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// EnvFile is a dotenv file listed in an env_file key.
type EnvFile struct {
	Path string `yaml:"path"`
	// Required reports whether a missing file is an error (the default).
	Required *bool `yaml:"required"`
}

// IsRequired reports whether a missing file is an error.
func (f EnvFile) IsRequired() bool {
	return f.Required == nil || *f.Required
}

// EnvFiles is the value of an env_file key: a path, a {path, required}
// mapping, or a list of them. Files are loaded in order: later files win.
type EnvFiles []EnvFile

// UnmarshalYAML accepts a string, a mapping or a list of strings and mappings.
func (l *EnvFiles) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		if node.Tag == "!!null" {
			*l = nil
			return nil
		}
		f, err := decodeEnvFile(node)
		if err != nil {
			return err
		}
		*l = EnvFiles{f}
		return nil
	case yaml.MappingNode:
		f, err := decodeEnvFile(node)
		if err != nil {
			return err
		}
		*l = EnvFiles{f}
		return nil
	case yaml.SequenceNode:
		out := make(EnvFiles, 0, len(node.Content))
		for _, item := range node.Content {
			f, err := decodeEnvFile(item)
			if err != nil {
				return err
			}
			out = append(out, f)
		}
		*l = out
		return nil
	}
	return fmt.Errorf("line %d: env_file must be a path, a {path, required} mapping or a list of them", node.Line)
}

func decodeEnvFile(node *yaml.Node) (EnvFile, error) {
	switch node.Kind {
	case yaml.ScalarNode:
		if node.Tag != "!!str" {
			return EnvFile{}, fmt.Errorf("line %d: env_file path must be a string", node.Line)
		}
		return EnvFile{Path: node.Value}, nil
	case yaml.MappingNode:
		// Reject unknown keys, as the rest of the Runpfile does.
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			if key != "path" && key != "required" {
				return EnvFile{}, fmt.Errorf("line %d: field %s not found in env_file", node.Content[i].Line, key)
			}
		}
		var f EnvFile
		if err := node.Decode(&f); err != nil {
			return EnvFile{}, err
		}
		if f.Path == "" {
			return EnvFile{}, fmt.Errorf("line %d: env_file requires a path", node.Line)
		}
		return f, nil
	}
	return EnvFile{}, fmt.Errorf("line %d: env_file entries must be a path or a {path, required} mapping", node.Line)
}

// envFileRef is an env_file entry with the directory it is relative to.
type envFileRef struct {
	file EnvFile
	root string
}

var envNameRegexp = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// resolveEnvFilePath returns the absolute path of an env file: vars are
// resolved, then a relative path is made absolute against root.
func resolveEnvFilePath(f EnvFile, root string, pre *cliPreprocessor) (string, error) {
	p := f.Path
	if pre != nil {
		p = pre.process(p)
	}
	return resolvePath(p, root)
}

// loadEnvFiles reads the env files in order and returns their variables:
// a later file overrides the earlier ones. A missing file is an error unless
// it is not required. Errors never include the values read.
func loadEnvFiles(files EnvFiles, root string, pre *cliPreprocessor) (map[string]string, error) {
	out := map[string]string{}
	for _, f := range files {
		path, err := resolveEnvFilePath(f, root, pre)
		if err != nil {
			return nil, fmt.Errorf("env_file %s: %w", f.Path, err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) && !f.IsRequired() {
				ui.Debugf("Optional env_file not found, skipped: %s", path)
				continue
			}
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("env_file not found: %s", path)
			}
			return nil, fmt.Errorf("env_file %s: %w", path, err)
		}
		vars, err := parseDotenv(data)
		if err != nil {
			return nil, fmt.Errorf("env_file %s: %w", path, err)
		}
		for k, v := range vars {
			out[k] = v
		}
	}
	return out, nil
}

// parseDotenv parses dotenv-style content: NAME=value lines, blank lines and
// "#" comments, an optional "export " prefix, values optionally in single
// quotes (literal) or double quotes (with \n, \t, \", \\ escapes). Nothing
// is expanded or evaluated. Errors report the line, never the value.
func parseDotenv(data []byte) (map[string]string, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	out := map[string]string{}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	for i := 0; i < len(lines); i++ {
		lineNo := i + 1
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if rest, ok := strings.CutPrefix(line, "export "); ok {
			line = strings.TrimSpace(rest)
		}
		name, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: expected NAME=value", lineNo)
		}
		name = strings.TrimSpace(name)
		if !envNameRegexp.MatchString(name) {
			return nil, fmt.Errorf("line %d: invalid variable name %q", lineNo, name)
		}
		value = strings.TrimLeft(value, " \t")
		if value != "" && (value[0] == '"' || value[0] == '\'') {
			v, next, err := parseQuotedDotenvValue(lines, i, value)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNo, err)
			}
			value, i = v, next
		} else {
			value = trimUnquotedDotenvValue(value)
		}
		out[name] = value
	}
	return out, nil
}

// parseQuotedDotenvValue parses the quoted value starting on lines[i], which
// may span several lines. It returns the value and the index of its last line.
func parseQuotedDotenvValue(lines []string, i int, value string) (string, int, error) {
	for {
		v, rest, closed := unquoteDotenv(value)
		if closed {
			rest = strings.TrimSpace(rest)
			if rest != "" && !strings.HasPrefix(rest, "#") {
				return "", i, errors.New("unexpected characters after the closing quote")
			}
			return v, i, nil
		}
		if i+1 >= len(lines) {
			return "", i, errors.New("unterminated quoted value")
		}
		i++
		value += "\n" + lines[i]
	}
}

// trimUnquotedDotenvValue removes a trailing " #" comment and the spaces
// around an unquoted value.
func trimUnquotedDotenvValue(value string) string {
	for _, sep := range []string{" #", "\t#"} {
		if idx := strings.Index(value, sep); idx >= 0 {
			value = value[:idx]
		}
	}
	return strings.TrimSpace(value)
}

// unquoteDotenv returns the content of the quoted value at the start of s,
// what follows the closing quote, and whether the quote is closed.
func unquoteDotenv(s string) (string, string, bool) {
	q := s[0]
	var sb strings.Builder
	for i := 1; i < len(s); i++ {
		c := s[i]
		if c == q {
			return sb.String(), s[i+1:], true
		}
		if q == '"' && c == '\\' && i+1 < len(s) {
			i++
			switch s[i] {
			case 'n':
				sb.WriteByte('\n')
			case 'r':
				sb.WriteByte('\r')
			case 't':
				sb.WriteByte('\t')
			case '"', '\\', '$':
				sb.WriteByte(s[i])
			default:
				sb.WriteByte('\\')
				sb.WriteByte(s[i])
			}
			continue
		}
		sb.WriteByte(c)
	}
	return "", "", false
}

// loadRootEnvFiles loads the Runpfile-level env_file of rf into the
// environment of runp, so that its variables can be referenced as ${NAME}
// in every unit. A variable already set by the env_file of an including
// Runpfile (tracked in set) is not overridden.
func loadRootEnvFiles(files EnvFiles, root string, set map[string]bool) error {
	vars, err := loadEnvFiles(files, root, nil)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(vars))
	for name := range vars {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if set[name] {
			continue
		}
		if err := os.Setenv(name, vars[name]); err != nil {
			return fmt.Errorf("env_file: cannot set %s: %w", name, err)
		}
		set[name] = true
	}
	return nil
}

// applyRootEnvFiles records the Runpfile-level env files of rf for runp
// validate and loads them (see loadRootEnvFiles), tracking the variables set
// in src, shared with the included files. When forValidation is true errors
// are ignored: runp validate reports them.
func applyRootEnvFiles(rf *Runpfile, src *runpfileSource, forValidation bool) error {
	if src.envSet == nil {
		src.envSet = map[string]bool{}
	}
	for _, f := range rf.EnvFile {
		rf.envFileRefs = append(rf.envFileRefs, envFileRef{file: f, root: rf.Root})
	}
	err := loadRootEnvFiles(rf.EnvFile, rf.Root, src.envSet)
	if forValidation {
		return nil
	}
	return err
}

// validateEnvFiles checks that the env files of the Runpfile and of its
// units exist (unless optional) and can be parsed. Values are never reported.
func validateEnvFiles(rf *Runpfile) []error {
	var errs []error
	for _, ref := range rf.envFileRefs {
		if varsRegexp.MatchString(ref.file.Path) {
			errs = append(errs, fmt.Errorf("env_file %s: variables are not supported in the Runpfile-level env_file", ref.file.Path))
			continue
		}
		if _, err := loadEnvFiles(EnvFiles{ref.file}, ref.root, nil); err != nil {
			errs = append(errs, err)
		}
	}
	errs = append(errs, validateUnitEnvFiles(rf, false)...)
	return errs
}

// validateUnitEnvFiles checks the env_file entries of the units of rf.
// With resolved false (runp validate) the paths containing {{vars}} are
// skipped; with resolved true the units must have their vars bound.
func validateUnitEnvFiles(rf *Runpfile, resolved bool) []error {
	var errs []error
	unitNames := make([]string, 0, len(rf.Units))
	for name := range rf.Units {
		unitNames = append(unitNames, name)
	}
	sort.Strings(unitNames)
	for _, name := range unitNames {
		unit := rf.Units[name]
		if unit == nil || len(unit.EnvFile) == 0 {
			continue
		}
		if unit.SSHTunnel != nil {
			errs = append(errs, fmt.Errorf("unit %q: env_file is not supported for ssh_tunnel units", name))
			continue
		}
		root := unitRoot(unit, rf.Root)
		for _, f := range unit.EnvFile {
			if !resolved && varsRegexp.MatchString(f.Path) {
				continue
			}
			if _, err := loadEnvFiles(EnvFiles{f}, root, newCliPreprocessor(unit.vars)); err != nil {
				errs = append(errs, fmt.Errorf("unit %q: %w", name, err))
			}
		}
	}
	return errs
}
