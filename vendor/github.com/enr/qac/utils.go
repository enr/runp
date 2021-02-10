package qac

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
)

func resolvePath(declared string, context planContext) (string, error) {
	bd := filepath.FromSlash(context.basedir)
	if declared == "" {
		return bd, nil
	}
	if strings.HasPrefix(declared, "~") {
		home, err := homedir.Dir()
		if err != nil {
			return "", err
		}
		relpath := strings.TrimPrefix(declared, "~")
		return filepath.FromSlash(path.Join(home, relpath)), nil
	}
	dec := filepath.FromSlash(declared)
	if path.IsAbs(dec) {
		return dec, nil
	}
	j := filepath.Join(bd, dec)
	return filepath.Abs(j)
}
