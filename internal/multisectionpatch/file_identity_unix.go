//go:build darwin || linux

package multisectionpatch

import (
	"fmt"
	"os"
	"syscall"
)

// fileIdentityAndLinks returns the stable filesystem identity and hard-link
// count captured by the same open handle used to read the file.
func fileIdentityAndLinks(_ *os.File, info os.FileInfo) (string, uint64, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", 0, fmt.Errorf("unexpected file information type %T", info.Sys())
	}
	identity := fmt.Sprintf("%d:%d", uint64(stat.Dev), uint64(stat.Ino))
	return identity, uint64(stat.Nlink), nil
}

func validateTargetForEdit(_ string, _ os.FileInfo) error {
	return nil
}
