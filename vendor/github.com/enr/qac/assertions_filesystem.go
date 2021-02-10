package qac

import "fmt"

func (a *FileSystemAssertion) actualAssertion(context planContext) (assertion, error) {
	fa := a.File != ""
	da := a.Directory != ""
	if fa && da {
		return nil, fmt.Errorf("Invalid file system assertion: file %s directory %s", a.File, a.Directory)
	}
	shouldExists := a.Exists == nil || *a.Exists
	if fa {
		// TODO check invalid fields
		return &FileAssertion{
			Path:         a.File,
			Extension:    a.Extension,
			Exists:       shouldExists,
			ContainsAll:  a.ContainsAll,
			ContainsAny:  a.ContainsAny,
			EqualsTo:     a.EqualsTo,
			TextEqualsTo: a.TextEqualsTo,
		}, nil
	}
	// TODO check invalid fields
	return &DirectoryAssertion{
		Path:            a.Directory,
		Exists:          shouldExists,
		ContainsAll:     a.ContainsAll,
		ContainsAny:     a.ContainsAny,
		EqualsTo:        a.EqualsTo,
		ContainsExactly: a.ContainsExactly,
	}, nil
}
