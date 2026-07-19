//go:build !darwin && !linux && !windows

package multisectionpatch

import (
	"errors"
	"os"
)

func fileIdentityAndLinks(_ *os.File, _ os.FileInfo) (string, uint64, error) {
	return "", 0, errors.New("editing is unsupported on this operating system")
}

func validateTargetForEdit(_ string, _ os.FileInfo) error {
	return nil
}
