package multisectionpatch

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var errForcedWrite = errors.New("forced output failure")

type failOnWrite struct {
	call   int
	failAt int
}

func (writer *failOnWrite) Write(data []byte) (int, error) {
	writer.call++
	if writer.call == writer.failAt {
		return 0, errForcedWrite
	}
	return len(data), nil
}

func TestReadReturnsFailureWhenHumanOutputFails(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\n")
	var stderr bytes.Buffer

	code := Run(
		[]string{"read", target},
		strings.NewReader(""),
		&failOnWrite{failAt: 1},
		&stderr,
	)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), errForcedWrite.Error()) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEditDryRunReturnsFailureWhenDiffOutputFails(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\n")
	spec := editSpec(t, target, "ONE\n")
	var stderr bytes.Buffer

	code := Run(
		[]string{"edit"},
		strings.NewReader(spec),
		&failOnWrite{failAt: 1},
		&stderr,
	)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), errForcedWrite.Error()) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEditApplyReturnsFailureWhenStatusOutputFails(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\n")
	spec := editSpec(t, target, "ONE\n")
	var stderr bytes.Buffer

	code := Run(
		[]string{"edit", "--apply"},
		strings.NewReader(spec),
		&failOnWrite{failAt: 2},
		&stderr,
	)

	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), errForcedWrite.Error()) {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if got, err := os.ReadFile(target); err != nil {
		t.Fatal(err)
	} else if string(got) != "ONE\n" {
		t.Fatalf("target = %q, want applied edit", got)
	}
}

func TestReadRejectsNonRegularInput(t *testing.T) {
	code, stdout, stderr := invoke([]string{"read", os.DevNull}, "")

	if code != 1 || stdout != "" {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "not a regular file") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestEditRejectsNonRegularReplacementFile(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\n")
	spec, err := json.Marshal(map[string]any{
		"edits": []map[string]any{{
			"file":             target,
			"start_line":       1,
			"end_line":         1,
			"replacement_file": os.DevNull,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke([]string{"edit"}, string(spec))

	if code != 1 || stdout != "" {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "not a regular file") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func editSpec(t *testing.T, target, replacement string) string {
	t.Helper()
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
	return string(spec)
}

var _ io.Writer = (*failOnWrite)(nil)
