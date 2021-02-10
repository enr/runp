package qac

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/enr/go-files/files"
)

func (a *DirectoryAssertion) verify(context planContext) AssertionResult {
	result := AssertionResult{
		description: fmt.Sprintf(`directory %s`, a.Path),
	}
	actualPath, err := resolvePath(a.Path, context)
	if err != nil {
		result.addError(err)
		return result
	}
	fileExists := files.Exists(actualPath)
	shouldExist := a.Exists
	if shouldExist != fileExists {
		result.addErrorf(`directory %s exist expected %t but got %t`, actualPath, shouldExist, fileExists)
		return result
	}
	if !shouldExist {
		return result
	}
	isDir := files.IsDir(actualPath)
	if !isDir {
		result.addErrorf(`directory %s is not a directory`, actualPath)
		return result
	}
	files, err := a.files(actualPath)
	if err != nil {
		result.addError(err)
		return result
	}
	var f1 string
	var f2 string
	if a.EqualsTo != "" {
		eq, _ := resolvePath(a.EqualsTo, context)
		otherFiles, _ := a.files(eq)
		if len(files) != len(otherFiles) {
			result.addErrorf("directory %s differs from %s: it contains %d files, expected %d", actualPath, eq, len(files), len(otherFiles))
		}
		for _, f := range files {
			if !sliceContains(otherFiles, f) {
				result.addErrorf(`file %s in %s but not in %s`, f, actualPath, eq)
				continue
			}
			f1 = filepath.Join(actualPath, f)
			f2 = filepath.Join(eq, f)
			result.addErrors(verifyFilesEqual(f1, f2))
		}
		for _, f := range otherFiles {
			if !sliceContains(files, f) {
				result.addErrorf(`file %s in %s but not in %s`, f, eq, actualPath)
			}
		}
	}
	if len(a.ContainsExactly) > 0 {
		// if len(files) != len(a.ContainsExactly) {
		// 	result.addErrorf("expected %s contains %d files but got %d\n%v%v", actualPath, len(a.ContainsExactly), len(files),
		// 		a.ContainsExactly, files)
		// }
		// for _, f := range files {
		// 	if !sliceContains(a.ContainsExactly, f) {
		// 		result.addErrorf(`file %s not in expected %q`, f, a.ContainsExactly)
		// 	}
		// }
		// for _, f := range a.ContainsExactly {
		// 	if !sliceContains(files, f) {
		// 		result.addErrorf(`file %s not found in actual dir content %q`, f, files)
		// 	}
		// }
		a.verifyContainsExactly(actualPath, files, result)
	}
	if len(a.ContainsAny) > 0 {
		// miss := true
		// for _, f := range a.ContainsAny {
		// 	if sliceContains(files, f) {
		// 		miss = false
		// 	}
		// }
		// if miss {
		// 	result.addErrorf(`directory does not contain any of %q`, files)
		// }
		a.verifyContainsAny(actualPath, files, result)
	}
	if len(a.ContainsAll) > 0 {
		// missing := []string{}
		// for _, f := range a.ContainsAll {
		// 	if !sliceContains(files, f) {
		// 		missing = append(missing, f)
		// 	}
		// }
		// if len(missing) > 0 {
		// 	result.addErrorf(`missing files: %q`, missing)
		// }
		a.verifyContainsAll(actualPath, files, result)
	}
	return result
}

func (a *DirectoryAssertion) verifyContainsAll(actualPath string, files []string, result AssertionResult) {
	missing := []string{}
	for _, f := range a.ContainsAll {
		if !sliceContains(files, f) {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		result.addErrorf(`missing files: %q`, missing)
	}
}

func (a *DirectoryAssertion) verifyContainsAny(actualPath string, files []string, result AssertionResult) {
	miss := true
	for _, f := range a.ContainsAny {
		if sliceContains(files, f) {
			miss = false
		}
	}
	if miss {
		result.addErrorf(`directory does not contain any of %q`, files)
	}
}

func (a *DirectoryAssertion) verifyContainsExactly(actualPath string, files []string, result AssertionResult) {
	if len(files) != len(a.ContainsExactly) {
		result.addErrorf("expected %s contains %d files but got %d\n%v%v", actualPath, len(a.ContainsExactly), len(files),
			a.ContainsExactly, files)
	}
	for _, f := range files {
		if !sliceContains(a.ContainsExactly, f) {
			result.addErrorf(`file %s not in expected %q`, f, a.ContainsExactly)
		}
	}
	for _, f := range a.ContainsExactly {
		if !sliceContains(files, f) {
			result.addErrorf(`file %s not found in actual dir content %q`, f, files)
		}
	}
}

func (a *DirectoryAssertion) files(dir string) ([]string, error) {
	aa := []string{}
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		r := strings.TrimPrefix(path, dir)
		rs := strings.Replace(r, string(os.PathSeparator), "/", -1)
		aa = append(aa, strings.TrimPrefix(rs, "/"))
		return err
	})
	return aa, err
}

func sliceContains(slice []string, element string) bool {
	for _, f := range slice {
		if f == element {
			return true
		}
	}
	return false
}
