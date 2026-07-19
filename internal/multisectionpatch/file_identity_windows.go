//go:build windows

package multisectionpatch

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const fileIDInfoClass = 18

var getFileInformationByHandleEx = syscall.NewLazyDLL("kernel32.dll").
	NewProc("GetFileInformationByHandleEx")

type windowsFileIDInfo struct {
	volume uint64
	value  [16]byte
}

// fileIdentityAndLinks uses Windows' 128-bit file identifier so identities
// remain valid on NTFS and ReFS. It also returns the hard-link count.
func fileIdentityAndLinks(file *os.File, _ os.FileInfo) (string, uint64, error) {
	connection, err := file.SyscallConn()
	if err != nil {
		return "", 0, err
	}
	var basic syscall.ByHandleFileInformation
	var extended windowsFileIDInfo
	var callErr error
	if err := connection.Control(func(raw uintptr) {
		handle := syscall.Handle(raw)
		if callErr = syscall.GetFileInformationByHandle(handle, &basic); callErr != nil {
			return
		}
		var result uintptr
		result, _, callErr = getFileInformationByHandleEx.Call(
			uintptr(handle),
			fileIDInfoClass,
			uintptr(unsafe.Pointer(&extended)),
			unsafe.Sizeof(extended),
		)
		if result == 0 {
			if callErr == nil || callErr == syscall.Errno(0) {
				callErr = syscall.EINVAL
			}
			return
		}
		callErr = nil
	}); err != nil {
		return "", 0, err
	}
	if callErr != nil {
		return "", 0, callErr
	}
	identity := fmt.Sprintf(
		"%d:%x",
		extended.volume,
		extended.value,
	)
	return identity, uint64(basic.NumberOfLinks), nil
}

// validateTargetForEdit rejects the Windows read-only attribute before Multi
// Section Patch creates staged files. MoveFileEx cannot safely replace such a
// destination, and read-only staged recovery files cannot be removed reliably.
func validateTargetForEdit(path string, info os.FileInfo) error {
	if info.Mode().Perm()&0o200 == 0 {
		return fmt.Errorf("%s: target is read-only", path)
	}
	return nil
}
