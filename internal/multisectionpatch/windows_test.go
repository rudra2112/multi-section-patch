//go:build windows

package multisectionpatch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestWindowsEditRejectsReadOnlyTarget(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\n")
	if err := os.Chmod(target, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(target, 0o666); err != nil && !os.IsNotExist(err) {
			t.Errorf("restore target permissions: %v", err)
		}
	})
	spec, err := json.Marshal(map[string]any{
		"edits": []map[string]any{{
			"file":        target,
			"start_line":  1,
			"end_line":    1,
			"replacement": "ONE\n",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke([]string{"edit", "--apply"}, string(spec))

	if code != 1 || stdout != "" {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "target is read-only") {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
	assertFileContent(t, target, "one\n")
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o200 != 0 {
		t.Fatalf("target mode = %o, want read-only", info.Mode().Perm())
	}
}

func TestWindowsEditRejectsExclusivelyOpenedTarget(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\n")
	spec, err := json.Marshal(map[string]any{
		"edits": []map[string]any{{
			"file":        target,
			"start_line":  1,
			"end_line":    1,
			"replacement": "ONE\n",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	path, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := syscall.CreateFile(
		path,
		syscall.GENERIC_READ,
		0,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke([]string{"edit", "--apply"}, string(spec))
	if err := syscall.CloseHandle(handle); err != nil {
		t.Fatal(err)
	}

	if code != 1 || stdout != "" {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "cannot") {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
	assertFileContent(t, target, "one\n")
}
