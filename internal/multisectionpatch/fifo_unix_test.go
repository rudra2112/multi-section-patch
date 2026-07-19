//go:build darwin || linux

package multisectionpatch

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestEditRejectsFIFOTargetWithoutBlocking(t *testing.T) {
	target := filepath.Join(t.TempDir(), "target.fifo")
	if err := syscall.Mkfifo(target, 0o600); err != nil {
		t.Fatal(err)
	}
	replacement := "text\n"
	spec, err := json.Marshal(map[string]any{
		"edits": []map[string]any{{
			"file":        target,
			"start_line":  1,
			"end_line":    1,
			"replacement": replacement,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	result := make(chan struct {
		code   int
		stdout string
		stderr string
	}, 1)
	go func() {
		code, stdout, stderr := invoke([]string{"edit"}, string(spec))
		result <- struct {
			code   int
			stdout string
			stderr string
		}{code: code, stdout: stdout, stderr: stderr}
	}()

	select {
	case got := <-result:
		if got.code != 1 || got.stdout != "" {
			t.Fatalf("code = %d, stdout = %q, stderr = %q", got.code, got.stdout, got.stderr)
		}
		if !strings.Contains(got.stderr, "not a regular file") {
			t.Fatalf("stderr = %q", got.stderr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("edit blocked while opening a FIFO target")
	}
}
