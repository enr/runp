package qac

import (
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
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
		result.addError(err)
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
			result.addError(err)
			// block
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
			result.addError(err)
			// block
			return result
		}
		if !files.Exists(exp) {
			result.addErrorf(`file not found %s`, exp)
			return result
		}

		result.addErrors(verifyFilesEqualText(actualPath, exp))
	}

	if len(a.ContainsAll) > 0 {
		content, err := ioutil.ReadFile(actualPath)
		if err != nil {
			result.addError(err)
			return result
		}
		cf := string(content)
		for _, t := range a.ContainsAll {
			if !strings.Contains(cf, t) {
				result.addError(fmt.Errorf("%s file\n%s\ndoes not contain:\n%s", actualPath, cf, t))
			}
		}
	}
	if len(a.ContainsAny) > 0 {
		content, err := ioutil.ReadFile(actualPath)
		if err != nil {
			result.addError(err)
			return result
		}
		cf := string(content)
		// fail := true
		// for _, t := range a.ContainsAny {
		// 	if strings.Contains(cf, t) {
		// 		fail = false
		// 		break
		// 	}
		// }
		if a.failContainsAny(cf) {
			result.addError(fmt.Errorf("%s file\n%s\ndoes not contain any of :\n%q", actualPath, cf, a.ContainsAny))
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
	errs := []error{}
	hash1, _ := hash(actualPath)
	hash2, _ := hash(other)
	if hash1 != hash2 {
		errs = append(errs, fmt.Errorf("File %s [%s] differs from\n%s [%s]", actualPath, hash1, other, hash2))
	}
	return errs
}

func verifyFilesEqualText(actualPath string, exp string) []error {
	errs := []error{}
	filelines := []string{}
	files.EachLine(actualPath, func(line string) error {
		filelines = append(filelines, line)
		return nil
	})
	expectedlines := []string{}
	files.EachLine(exp, func(line string) error {
		expectedlines = append(expectedlines, line)
		return nil
	})
	if len(filelines) != len(expectedlines) {
		errs = append(errs, fmt.Errorf("EachLine(%s), expected %d lines but got %d", actualPath, len(expectedlines), len(filelines)))
	}
	if len(filelines) == 0 || len(expectedlines) == 0 {
		// probably a missing/unexpected file
		return errs
	}
	for index, actual := range filelines {
		if len(expectedlines) <= index {
			errs = append(errs, fmt.Errorf(`unexpected line %d in file %s`, (index+1), actual))
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
	defer fh.Close()
	if err != nil {
		return "", err
	}
	h := sha1.New()
	io.Copy(h, fh)
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func isBinary(path string) bool {
	file, _ := os.Open(path)
	defer file.Close()
	return isBinaryFile(file)
}

// isBinary guesses whether a file is binary by reading the first X bytes and seeing if there are any nulls.
// Assumes the file starts seeked the beginning.
func isBinaryFile(file *os.File) bool {
	defer file.Seek(0, 0)
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
