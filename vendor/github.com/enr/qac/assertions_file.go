package qac

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/enr/go-files/files"
)

const (
	binaryDetectionBytes = 8000 // Same as git
)

func (a *FileAssertion) verify(context planContext) AssertionResult {
	result := AssertionResult{
		description: fmt.Sprintf(`file %s`, a.Path),
	}
	fp := a.Path
	if a.Extension.isSet() {
		fp = fmt.Sprintf(`%s%s`, a.Path, a.Extension.get())
	}
	actualPath, err := resolvePath(fp, context)
	if err != nil {
		result.addInfraError(fmt.Errorf("resolving file path %q: %w", fp, err))
		return result
	}
	fileExists := files.Exists(actualPath)
	shouldExist := a.Exists
	if shouldExist != fileExists {
		err := fmt.Errorf(`file %s exist expected %t but got %t`, actualPath, shouldExist, fileExists)
		result.addError(err)
		return result
	}
	if !shouldExist {
		return result
	}
	if a.EqualsTo != "" {
		other, err := resolvePath(a.EqualsTo, context)
		if err != nil {
			result.addInfraError(fmt.Errorf("resolving equals_to path %q: %w", a.EqualsTo, err))
			return result
		}
		if !files.Exists(other) {
			result.addErrorf(`file not found %s`, other)
			return result
		}
		errs := []error{}
		if isBinary(actualPath) {
			errs = verifyFilesEqualHash(actualPath, other)
		} else {
			errs = verifyFilesEqualText(actualPath, other)
		}
		result.addErrors(errs)

	}
	if a.TextEqualsTo != "" {
		exp, err := resolvePath(a.TextEqualsTo, context)
		if err != nil {
			result.addInfraError(fmt.Errorf("resolving text_equals_to path %q: %w", a.TextEqualsTo, err))
			return result
		}
		if !files.Exists(exp) {
			result.addErrorf(`file not found %s`, exp)
			return result
		}

		result.addErrors(verifyFilesEqualText(actualPath, exp))
	}

	if len(a.ContainsAll) > 0 {
		content, err := os.ReadFile(actualPath)
		if err != nil {
			result.addInfraError(fmt.Errorf("reading %q: %w", actualPath, err))
			return result
		}
		cf := string(content)
		for _, t := range a.ContainsAll {
			if !strings.Contains(cf, t) {
				result.addError(fmt.Errorf("%s file\n%s\ndoes not contain:\n%s", actualPath, snippet(cf), t))
			}
		}
	}
	if len(a.ContainsAny) > 0 {
		content, err := os.ReadFile(actualPath)
		if err != nil {
			result.addInfraError(fmt.Errorf("reading %q: %w", actualPath, err))
			return result
		}
		cf := string(content)
		if a.failContainsAny(cf) {
			result.addError(fmt.Errorf("%s file\n%s\ndoes not contain any of:\n%q", actualPath, snippet(cf), a.ContainsAny))
		}
	}

	return result
}

func (a *FileAssertion) failContainsAny(cf string) bool {
	fail := true
	for _, t := range a.ContainsAny {
		if strings.Contains(cf, t) {
			fail = false
			break
		}
	}
	return fail
}

func verifyFilesEqual(actualPath string, other string) []error {
	if isBinary(actualPath) {
		return verifyFilesEqualHash(actualPath, other)
	}
	return verifyFilesEqualText(actualPath, other)
}

func verifyFilesEqualHash(actualPath string, other string) []error {
	hash1, err := hash(actualPath)
	if err != nil {
		return []error{asInfraError(fmt.Errorf("hashing actual file %q: %w", actualPath, err))}
	}
	hash2, err := hash(other)
	if err != nil {
		return []error{asInfraError(fmt.Errorf("hashing expected file %q: %w", other, err))}
	}
	if hash1 != hash2 {
		return []error{&Error{
			Kind: KindAssertionFailure,
			msg:  fmt.Sprintf("File %s [%s] differs from\n%s [%s]", actualPath, hash1, other, hash2),
		}}
	}
	return nil
}

func verifyFilesEqualText(actualPath string, exp string) []error {
	filelines := []string{}
	if err := files.EachLine(actualPath, func(line string) error {
		filelines = append(filelines, line)
		return nil
	}); err != nil {
		return []error{asInfraError(fmt.Errorf("reading %q: %w", actualPath, err))}
	}
	expectedlines := []string{}
	if err := files.EachLine(exp, func(line string) error {
		expectedlines = append(expectedlines, line)
		return nil
	}); err != nil {
		return []error{asInfraError(fmt.Errorf("reading %q: %w", exp, err))}
	}
	errs := []error{}
	if len(filelines) != len(expectedlines) {
		errs = append(errs, fmt.Errorf("EachLine(%s), expected %d lines but got %d", actualPath, len(expectedlines), len(filelines)))
	}
	for index, actual := range filelines {
		if len(expectedlines) <= index {
			errs = append(errs, fmt.Errorf(`unexpected extra line %d in %s: %q`, index+1, actualPath, actual))
			continue
		}
		expected := expectedlines[index]
		if actual != expected {
			errs = append(errs, fmt.Errorf(`line %d expected %q but got %q`, (index+1), expected, actual))
		}
	}
	return errs
}

func hash(fullpath string) (string, error) {
	fh, err := os.Open(fullpath)
	if err != nil {
		return "", fmt.Errorf("opening %q: %w", fullpath, err)
	}
	defer fh.Close()
	h := sha1.New()
	if _, err := io.Copy(h, fh); err != nil {
		return "", fmt.Errorf("reading %q: %w", fullpath, err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func isBinary(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	return isBinaryFile(file)
}

// isBinaryFile guesses whether a file is binary by reading the first X bytes and seeing if there are any nulls.
// Assumes the file is seeked to the beginning by the caller.
func isBinaryFile(file *os.File) bool {
	buf := make([]byte, binaryDetectionBytes)
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return false
		}
		if n == 0 {
			break
		}
		for i := 0; i < n; i++ {
			if buf[i] == 0x00 {
				return true
			}
		}
		buf = buf[n:]
	}
	return false
}
