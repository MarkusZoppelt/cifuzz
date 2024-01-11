//go:build freebsd || linux

package ldd

import (
	"github.com/pkg/errors"
	"github.com/u-root/u-root/pkg/ldd"

	"code-intelligence.com/cifuzz/util/fileutil"
)

func NonSystemSharedLibraries(executable string) ([]string, error) {
	var sharedObjects []string

	// ldd provides the complete list of dynamic dependencies of a dynamically linked file.
	// That is, we don't have to recursively query the transitive dynamic dependencies.
	dependencies, err := ldd.List(executable)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for _, dep := range dependencies {
		if fileutil.IsSharedLibrary(dep) && !fileutil.IsSystemLibrary(dep) {
			sharedObjects = append(sharedObjects, dep)
		}
	}

	return sharedObjects, nil
}
