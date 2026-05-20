package qac

import "fmt"

func (a *FileSystemAssertion) actualAssertion(context planContext) (assertion, error) {
	fa := a.File != ""
	da := a.Directory != ""
	if fa && da {
		return nil, asConfigError(fmt.Errorf("invalid file system assertion: file %s directory %s", a.File, a.Directory))
	}
	shouldExists := a.Exists == nil || *a.Exists
	if fa {
		if len(a.ContainsExactly) > 0 {
			return nil, asConfigError(fmt.Errorf("field contains_exactly is not valid for file assertion %q", a.File))
		}
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
	if a.TextEqualsTo != "" {
		return nil, asConfigError(fmt.Errorf("field text_equals_to is not valid for directory assertion %q", a.Directory))
	}
	return &DirectoryAssertion{
		Path:            a.Directory,
		Exists:          shouldExists,
		ContainsAll:     a.ContainsAll,
		ContainsAny:     a.ContainsAny,
		EqualsTo:        a.EqualsTo,
		ContainsExactly: a.ContainsExactly,
	}, nil
}
