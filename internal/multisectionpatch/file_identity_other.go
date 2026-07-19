//go:build !darwin && !linux && !windows

package multisectionpatch

import (
	"errors"
	"os"
)

// fileIdentityAndLinks reports that editing is unsupported when the current
// platform has no safe filesystem-identity implementation.
func fileIdentityAndLinks(_ *os.File, _ os.FileInfo) (string, uint64, error) {
	return "", 0, errors.New("editing is unsupported on this operating system")
}

// validateTargetForEdit adds no checks because unsupported platforms fail
// earlier while capturing the required filesystem identity.
func validateTargetForEdit(_ string, _ os.FileInfo) error {
	return nil
}
