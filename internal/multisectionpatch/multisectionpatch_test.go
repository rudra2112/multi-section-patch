package multisectionpatch

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func invoke(args []string, input string) (int, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(args, strings.NewReader(input), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

type jsonErrorEnvelope struct {
	Error struct {
		Code      string `json:"code"`
		Command   string `json:"command"`
		Message   string `json:"message"`
		ItemIndex int    `json:"item_index"`
		Name      string `json:"name"`
		File      string `json:"file"`
		Field     string `json:"field"`
	} `json:"error"`
}

func decodeJSONError(t *testing.T, stderr string) jsonErrorEnvelope {
	t.Helper()
	var payload jsonErrorEnvelope
	if err := json.Unmarshal([]byte(stderr), &payload); err != nil {
		t.Fatalf("stderr is not a JSON error envelope: %v\n%s", err, stderr)
	}
	if payload.Error.Code == "" || payload.Error.Command == "" || payload.Error.Message == "" {
		t.Fatalf("incomplete JSON error envelope: %#v", payload)
	}
	return payload
}

func dryRunPlanSHA256(t *testing.T, spec string) string {
	t.Helper()
	code, stdout, stderr := invoke([]string{"edit", "--json"}, spec)
	if code != 0 {
		t.Fatalf("dry-run code = %d, stderr = %q", code, stderr)
	}
	var payload struct {
		PlanSHA256 string `json:"plan_sha256"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("cannot decode dry-run JSON: %v\n%s", err, stdout)
	}
	if len(payload.PlanSHA256) != sha256.Size*2 {
		t.Fatalf("plan_sha256 = %q, want %d lowercase hex characters", payload.PlanSHA256, sha256.Size*2)
	}
	if decoded, err := hex.DecodeString(payload.PlanSHA256); err != nil || len(decoded) != sha256.Size {
		t.Fatalf("plan_sha256 = %q, want lowercase SHA-256: %v", payload.PlanSHA256, err)
	}
	if payload.PlanSHA256 != strings.ToLower(payload.PlanSHA256) {
		t.Fatalf("plan_sha256 = %q, want lowercase", payload.PlanSHA256)
	}
	return payload.PlanSHA256
}

func reviewedApplyArgs(t *testing.T, spec string, extra ...string) []string {
	t.Helper()
	args := []string{"edit", "--apply", "--expect-plan", dryRunPlanSHA256(t, spec)}
	return append(args, extra...)
}

func TestReadMultipleSections(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first.md")
	second := filepath.Join(root, "second.md")
	if err := os.WriteFile(first, []byte("zero\none\ntwo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("# Intro\nhello\n# End\nbye\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke([]string{
		"read",
		first + "@2:3",
		second + "@# Intro..# End",
	}, "")

	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	for _, want := range []string{"2| one", "3| two", "# Intro", "hello", "sha256="} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "bye") {
		t.Errorf("stdout contains unselected content:\n%s", stdout)
	}
}

func TestFileSnapshotCacheReusesCanonicalFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "sample.txt")
	writeTestFile(t, target, "before\n")
	cache := newFileSnapshotCache()

	first, err := cache.read(target)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, target, "after\n")
	second, err := cache.read(filepath.Join(root, ".", "sample.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(first.data, []byte("before\n")) ||
		!bytes.Equal(second.data, first.data) ||
		second.path != first.path ||
		second.identity != first.identity {
		t.Fatalf("cache returned different snapshots: first=%#v second=%#v", first, second)
	}
}

func TestReadRejectsInvalidOccurrence(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	if err := os.WriteFile(target, []byte("start\nbody\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	spec, err := json.Marshal(map[string]any{
		"sections": []map[string]any{{
			"file":       target,
			"start":      "start",
			"occurrence": 0,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke([]string{"read", "--json"}, string(spec))

	if code != 1 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "occurrence must be at least 1") {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
}

func TestReadRejectsRangePastEndOfFile(t *testing.T) {
	target := filepath.Join(t.TempDir(), "short.txt")
	writeTestFile(t, target, "one\ntwo\n")

	code, stdout, stderr := invoke([]string{"read", target + "@9:10"}, "")

	if code != 1 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "invalid line range 9:10") {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
}

func TestReadReportsInvalidRegexWithoutTrace(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\ntwo\n")

	code, stdout, stderr := invoke([]string{"read", target + "@/[/"}, "")

	if code != 1 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "multi-section-patch: error: invalid regex") {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
	if strings.Contains(stderr, "panic") || strings.Contains(stderr, "goroutine") {
		t.Fatalf("stderr contains a runtime trace: %q", stderr)
	}
}

func TestReadRegexAnchorsMatchLogicalLines(t *testing.T) {
	for name, newline := range map[string]string{
		"LF":   "\n",
		"CRLF": "\r\n",
	} {
		t.Run(name, func(t *testing.T) {
			target := filepath.Join(t.TempDir(), "sample.txt")
			writeTestFile(t, target, strings.Join(
				[]string{"alpha", "BEGIN", "body", "END", "omega", ""},
				newline,
			))
			spec, err := json.Marshal(map[string]any{
				"sections": []map[string]any{{
					"file":        target,
					"start_regex": "^BEGIN$",
					"end_regex":   "^END$",
				}},
			})
			if err != nil {
				t.Fatal(err)
			}

			code, stdout, stderr := invoke([]string{"read", "--json"}, string(spec))

			if code != 0 {
				t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
			}
			var payload struct {
				Sections []struct {
					Content string `json:"content"`
				} `json:"sections"`
			}
			if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
				t.Fatal(err)
			}
			want := "BEGIN" + newline + "body" + newline
			if len(payload.Sections) != 1 || payload.Sections[0].Content != want {
				t.Fatalf("unexpected payload: %#v", payload)
			}
		})
	}
}

func TestReadTreatsJSONMarkersAsLiteral(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "aab\n/a+b/\ntail\n")
	spec, err := json.Marshal(map[string]any{
		"sections": []map[string]any{{
			"file":  target,
			"start": "/a+b/",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke([]string{"read", "--json"}, string(spec))

	if code != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	var payload struct {
		Sections []struct {
			Content string `json:"content"`
		} `json:"sections"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Sections) != 1 || payload.Sections[0].Content != "/a+b/\ntail\n" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestReadRejectsMixedSelectorFamilies(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\ntwo\n")
	spec, err := json.Marshal(map[string]any{
		"sections": []map[string]any{{
			"file":        target,
			"start_line":  1,
			"end_line":    1,
			"start_regex": "[",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke([]string{"read", "--json"}, string(spec))

	if code != 1 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "selector") {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
}

func TestReadRejectsEditOnlyFields(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\n")
	for field, value := range map[string]any{
		"replacement":      "",
		"replacement_file": "",
		"expected_sha256":  "",
		"must_contain":     []string{},
	} {
		t.Run(field, func(t *testing.T) {
			item := map[string]any{"file": target}
			item[field] = value
			spec, err := json.Marshal(map[string]any{
				"sections": []map[string]any{item},
			})
			if err != nil {
				t.Fatal(err)
			}

			code, stdout, stderr := invoke([]string{"read", "--json"}, string(spec))

			if code != 1 || stdout != "" {
				t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
			}
			payload := decodeJSONError(t, stderr)
			if payload.Error.Code != "invalid_spec" ||
				payload.Error.Command != "read" ||
				payload.Error.ItemIndex != 1 ||
				payload.Error.File != target ||
				payload.Error.Field != field {
				t.Fatalf("unexpected JSON error: %#v", payload.Error)
			}
		})
	}
}

func TestCommandsRejectMarkerOptionsWithoutTheirBound(t *testing.T) {
	cases := []struct {
		name       string
		boundField string
		boundValue any
		option     string
		value      any
	}{
		{
			name:       "occurrence without start",
			boundField: "end",
			boundValue: "two",
			option:     "occurrence",
			value:      1,
		},
		{
			name:       "end occurrence without end",
			boundField: "start",
			boundValue: "one",
			option:     "end_occurrence",
			value:      1,
		},
		{
			name:       "include start without start",
			boundField: "end",
			boundValue: "two",
			option:     "include_start",
			value:      false,
		},
		{
			name:       "include end without end",
			boundField: "start",
			boundValue: "one",
			option:     "include_end",
			value:      true,
		},
	}
	for _, command := range []string{"read", "edit"} {
		for _, test := range cases {
			t.Run(command+"/"+test.name, func(t *testing.T) {
				target := filepath.Join(t.TempDir(), "sample.txt")
				writeTestFile(t, target, "one\ntwo\n")
				item := map[string]any{
					"file":          target,
					test.boundField: test.boundValue,
					test.option:     test.value,
				}
				key := "sections"
				if command == "edit" {
					key = "edits"
					item["replacement"] = "changed\n"
				}
				spec, err := json.Marshal(map[string]any{
					key: []map[string]any{item},
				})
				if err != nil {
					t.Fatal(err)
				}

				code, stdout, stderr := invoke([]string{command, "--json"}, string(spec))

				if code != 1 || stdout != "" {
					t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
				}
				payload := decodeJSONError(t, stderr)
				if payload.Error.Code != "invalid_spec" ||
					payload.Error.Command != command ||
					payload.Error.ItemIndex != 1 ||
					payload.Error.File != target ||
					payload.Error.Field != test.option {
					t.Fatalf("unexpected JSON error: %#v", payload.Error)
				}
			})
		}
	}
}

func TestJSONErrorReportsSectionContext(t *testing.T) {
	target := filepath.Join(t.TempDir(), "missing.txt")
	spec, err := json.Marshal(map[string]any{
		"sections": []map[string]any{{
			"name":       "missing section",
			"file":       target,
			"start_line": 1,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke([]string{"read", "--json"}, string(spec))

	if code != 1 || stdout != "" {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	payload := decodeJSONError(t, stderr)
	if payload.Error.Code != "section_resolution_failed" ||
		payload.Error.Command != "read" ||
		payload.Error.ItemIndex != 1 ||
		payload.Error.Name != "missing section" ||
		payload.Error.File != target {
		t.Fatalf("unexpected JSON error: %#v", payload.Error)
	}
}

func TestJSONErrorDetectionUsesCommandSpecificOptions(t *testing.T) {
	for name, args := range map[string][]string{
		"edit does not consume read context": {"edit", "--context", "--json"},
		"read does not consume edit plan":    {"read", "--expect-plan", "--json"},
	} {
		t.Run(name, func(t *testing.T) {
			code, stdout, stderr := invoke(args, "")

			if code != 1 || stdout != "" {
				t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
			}
			payload := decodeJSONError(t, stderr)
			if payload.Error.Code != "invalid_option" {
				t.Fatalf("unexpected JSON error: %#v", payload.Error)
			}
		})
	}
}

func TestEditRejectsEmptyMustContainGuard(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\n")
	for name, guard := range map[string]any{
		"empty string": "",
		"empty list":   []string{},
		"empty entry":  []string{"one", ""},
	} {
		t.Run(name, func(t *testing.T) {
			spec, err := json.Marshal(map[string]any{
				"edits": []map[string]any{{
					"file":         target,
					"replacement":  "ONE\n",
					"must_contain": guard,
				}},
			})
			if err != nil {
				t.Fatal(err)
			}

			code, stdout, stderr := invoke([]string{"edit", "--json"}, string(spec))

			if code != 1 || stdout != "" {
				t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
			}
			payload := decodeJSONError(t, stderr)
			if payload.Error.Code != "invalid_spec" ||
				payload.Error.ItemIndex != 1 ||
				payload.Error.File != target ||
				payload.Error.Field != "must_contain" {
				t.Fatalf("unexpected JSON error: %#v", payload.Error)
			}
		})
	}
}

func TestJSONTypeErrorsReportField(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\n")
	for _, test := range []struct {
		name    string
		command string
		key     string
		field   string
		item    map[string]any
	}{
		{
			name:    "read line number",
			command: "read",
			key:     "sections",
			field:   "start_line",
			item: map[string]any{
				"file":       target,
				"start_line": "not-a-number",
			},
		},
		{
			name:    "edit guard",
			command: "edit",
			key:     "edits",
			field:   "must_contain",
			item: map[string]any{
				"file":         target,
				"replacement":  "ONE\n",
				"must_contain": 7,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			spec, err := json.Marshal(map[string]any{
				test.key: []map[string]any{test.item},
			})
			if err != nil {
				t.Fatal(err)
			}

			code, stdout, stderr := invoke([]string{test.command, "--json"}, string(spec))

			if code != 1 || stdout != "" {
				t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
			}
			payload := decodeJSONError(t, stderr)
			if payload.Error.Code != "invalid_spec" ||
				payload.Error.Command != test.command ||
				payload.Error.ItemIndex != 1 ||
				payload.Error.File != target ||
				payload.Error.Field != test.field {
				t.Fatalf("unexpected JSON error: %#v", payload.Error)
			}
		})
	}
}

func TestJSONSemanticErrorsReportField(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "sample.txt")
	missing := filepath.Join(root, "missing.txt")
	writeTestFile(t, target, "one\n")
	for _, test := range []struct {
		name    string
		command string
		key     string
		code    string
		field   string
		item    map[string]any
	}{
		{
			name:    "read target",
			command: "read",
			key:     "sections",
			code:    "section_resolution_failed",
			field:   "file",
			item:    map[string]any{"file": missing},
		},
		{
			name:    "read occurrence",
			command: "read",
			key:     "sections",
			code:    "section_resolution_failed",
			field:   "occurrence",
			item: map[string]any{
				"file":       target,
				"start":      "one",
				"occurrence": 0,
			},
		},
		{
			name:    "read regular expression",
			command: "read",
			key:     "sections",
			code:    "section_resolution_failed",
			field:   "start_regex",
			item: map[string]any{
				"file":        target,
				"start_regex": "[",
			},
		},
		{
			name:    "edit digest guard",
			command: "edit",
			key:     "edits",
			code:    "edit_plan_failed",
			field:   "expected_sha256",
			item: map[string]any{
				"file":            target,
				"replacement":     "ONE\n",
				"expected_sha256": strings.Repeat("0", sha256.Size*2),
			},
		},
		{
			name:    "edit content guard",
			command: "edit",
			key:     "edits",
			code:    "edit_plan_failed",
			field:   "must_contain",
			item: map[string]any{
				"file":         target,
				"replacement":  "ONE\n",
				"must_contain": "missing",
			},
		},
		{
			name:    "edit replacement file",
			command: "edit",
			key:     "edits",
			code:    "edit_plan_failed",
			field:   "replacement_file",
			item: map[string]any{
				"file":             target,
				"replacement_file": missing,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			spec, err := json.Marshal(map[string]any{
				test.key: []map[string]any{test.item},
			})
			if err != nil {
				t.Fatal(err)
			}

			code, stdout, stderr := invoke([]string{test.command, "--json"}, string(spec))

			if code != 1 || stdout != "" {
				t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
			}
			payload := decodeJSONError(t, stderr)
			if payload.Error.Code != test.code ||
				payload.Error.Command != test.command ||
				payload.Error.ItemIndex != 1 ||
				payload.Error.Field != test.field {
				t.Fatalf("unexpected JSON error: %#v", payload.Error)
			}
		})
	}
}

func TestEditRejectsPresentEmptyFields(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\n")
	for _, field := range []string{"expected_sha256", "replacement_file"} {
		t.Run(field, func(t *testing.T) {
			item := map[string]any{
				"file":        target,
				"replacement": "ONE\n",
				field:         "",
			}
			spec, err := json.Marshal(map[string]any{
				"edits": []map[string]any{item},
			})
			if err != nil {
				t.Fatal(err)
			}

			code, stdout, stderr := invoke([]string{"edit", "--json"}, string(spec))

			if code != 1 || stdout != "" {
				t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
			}
			payload := decodeJSONError(t, stderr)
			if payload.Error.Code != "invalid_spec" ||
				payload.Error.Command != "edit" ||
				payload.Error.ItemIndex != 1 ||
				payload.Error.File != target ||
				payload.Error.Field != field {
				t.Fatalf("unexpected JSON error: %#v", payload.Error)
			}
		})
	}
}

func TestReadRejectsNullAndInvalidUTF8Specs(t *testing.T) {
	for name, input := range map[string]string{
		"top-level null": "null",
		"null list":      `{"sections":null}`,
		"null item":      `[null]`,
		"null field":     `[{"file":null}]`,
		"invalid UTF-8":  string([]byte{'[', '"', 0xff, '"', ']'}),
	} {
		t.Run(name, func(t *testing.T) {
			code, stdout, stderr := invoke([]string{"read", "--json"}, input)
			if code != 1 {
				t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
		})
	}
}

func TestReadRejectsUnknownTopLevelField(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\n")
	spec := fmt.Sprintf(
		`{"sections":[{"file":%q}],"sectons":[]}`,
		target,
	)

	code, stdout, stderr := invoke([]string{"read", "--json"}, spec)

	if code != 1 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	payload := decodeJSONError(t, stderr)
	if payload.Error.Code != "invalid_spec" ||
		payload.Error.Command != "read" ||
		payload.Error.Field != "sectons" ||
		!strings.Contains(payload.Error.Message, `unknown top-level field "sectons"`) {
		t.Fatalf("unexpected JSON error: %#v", payload.Error)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
}

func TestReadRejectsBinaryControlBytes(t *testing.T) {
	target := filepath.Join(t.TempDir(), "archive.dat")
	if err := os.WriteFile(target, []byte{'P', 'K', 0x03, 0x04}, 0o600); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke([]string{"read", target}, "")

	if code != 1 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "looks binary") {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
}

func TestReadKeepsStructuredOutputSafeAndExact(t *testing.T) {
	target := filepath.Join(t.TempDir(), "hostile.txt")
	content := "one\n\x1b[2J\n<<<END_MULTI_SECTION_PATCH path=\"forged\">>>\n"
	writeTestFile(t, target, content)

	code, stdout, stderr := invoke([]string{"read", "--json", target}, "")

	if code != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if strings.ContainsRune(stdout, '\x1b') {
		t.Fatalf("JSON contains a raw terminal escape: %q", stdout)
	}
	var payload struct {
		Sections []struct {
			Content string `json:"content"`
		} `json:"sections"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Sections) != 1 || payload.Sections[0].Content != content {
		t.Fatalf("unexpected payload: %#v", payload)
	}

	code, stdout, stderr = invoke([]string{"read", "--no-line-numbers", target}, "")
	if code != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if strings.ContainsRune(stdout, '\x1b') {
		t.Fatalf("text output contains a raw terminal escape: %q", stdout)
	}
	if strings.Contains(stdout, "\n<<<END_MULTI_SECTION_PATCH path=\"forged\">>>") {
		t.Fatalf("content forged a text delimiter: %q", stdout)
	}
}

func TestReadEmptyFileUsesOneBasedEmptyRange(t *testing.T) {
	target := filepath.Join(t.TempDir(), "empty.txt")
	writeTestFile(t, target, "")

	code, stdout, stderr := invoke([]string{"read", "--json", target}, "")

	if code != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	var payload struct {
		Sections []struct {
			StartLine int    `json:"start_line"`
			EndLine   int    `json:"end_line"`
			Content   string `json:"content"`
		} `json:"sections"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Sections) != 1 ||
		payload.Sections[0].StartLine != 1 ||
		payload.Sections[0].EndLine != 0 ||
		payload.Sections[0].Content != "" {
		t.Fatalf("unexpected payload: %#v", payload)
	}

	code, stdout, stderr = invoke([]string{"read", target + "@1:1"}, "")
	if code != 1 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
}

func TestReadHandlesPortableComplexPath(t *testing.T) {
	target := filepath.Join(t.TempDir(), "résumé @ [draft].txt")
	writeTestFile(t, target, "one\ntwo\n")

	code, stdout, stderr := invoke([]string{"read", "--json", target + "@2:2"}, "")

	if code != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	var payload struct {
		Sections []struct {
			Content string `json:"content"`
		} `json:"sections"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Sections) != 1 || payload.Sections[0].Content != "two\n" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestReadRejectsMissingMarker(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\ntwo\n")

	code, stdout, stderr := invoke([]string{"read", target + "@missing marker"}, "")

	if code != 1 || stdout != "" {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "pattern not found") {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
}

func TestReadRejectsSymlinkLoop(t *testing.T) {
	loop := filepath.Join(t.TempDir(), "loop.txt")
	if err := os.Symlink("loop.txt", loop); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	code, stdout, stderr := invoke([]string{"read", loop}, "")

	if code != 1 || stdout != "" {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "cannot resolve path") {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
}

func TestErrorsEscapeUntrustedControlCharacters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing\n\x1b[2J.txt")

	code, stdout, stderr := invoke([]string{"read", path}, "")

	if code != 1 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if strings.Count(stderr, "\n") != 1 || strings.ContainsRune(stderr, '\x1b') {
		t.Fatalf("unsafe stderr: %q", stderr)
	}
	if !strings.Contains(stderr, `\u000a`) || !strings.Contains(stderr, `\u001b`) {
		t.Fatalf("stderr did not visibly escape controls: %q", stderr)
	}
}

func TestEditPlanSHA256IsDeterministic(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\n")
	compact := editSpec(t, target, "ONE\n")
	var indented bytes.Buffer
	if err := json.Indent(&indented, []byte(compact), "", "  "); err != nil {
		t.Fatal(err)
	}

	first := dryRunPlanSHA256(t, compact)
	second := dryRunPlanSHA256(t, indented.String())

	if first != second {
		t.Fatalf("equivalent plans differ: %s != %s", first, second)
	}
	code, stdout, stderr := invoke([]string{"edit"}, compact)
	if code != 0 {
		t.Fatalf("human dry-run code = %d, stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "Plan SHA-256: "+first) {
		t.Fatalf("human dry run omits plan hash:\n%s", stdout)
	}
}

func TestApplyRequiresReviewedPlan(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\n")
	spec := editSpec(t, target, "ONE\n")

	code, stdout, stderr := invoke([]string{"edit", "--apply", "--json"}, spec)

	if code != 1 || stdout != "" {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	payload := decodeJSONError(t, stderr)
	if payload.Error.Code != "invalid_option" ||
		!strings.Contains(payload.Error.Message, "--apply requires --expect-plan") {
		t.Fatalf("unexpected JSON error: %#v", payload.Error)
	}
	assertFileContent(t, target, "one\n")
}

func TestApplyRejectsMalformedReviewedPlan(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\n")
	spec := editSpec(t, target, "ONE\n")

	code, stdout, stderr := invoke(
		[]string{"edit", "--apply", "--expect-plan", "ABC", "--json"},
		spec,
	)

	if code != 1 || stdout != "" {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	payload := decodeJSONError(t, stderr)
	if payload.Error.Code != "invalid_option" ||
		!strings.Contains(payload.Error.Message, "64-character lowercase SHA-256") {
		t.Fatalf("unexpected JSON error: %#v", payload.Error)
	}
	assertFileContent(t, target, "one\n")
}

func TestApplyRejectsReviewedPlanMismatch(t *testing.T) {
	t.Run("replacement changed", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "sample.txt")
		writeTestFile(t, target, "one\n")
		reviewedSpec := editSpec(t, target, "ONE\n")
		reviewedPlan := dryRunPlanSHA256(t, reviewedSpec)
		changedSpec := editSpec(t, target, "DIFFERENT\n")

		code, stdout, stderr := invoke(
			[]string{"edit", "--apply", "--expect-plan", reviewedPlan, "--json"},
			changedSpec,
		)

		if code != 1 || stdout != "" {
			t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
		}
		payload := decodeJSONError(t, stderr)
		if payload.Error.Code != "plan_mismatch" ||
			!strings.Contains(payload.Error.Message, "new dry run") {
			t.Fatalf("unexpected JSON error: %#v", payload.Error)
		}
		assertFileContent(t, target, "one\n")
	})

	t.Run("unselected target content changed", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "sample.txt")
		writeTestFile(t, target, "header\none\n")
		spec, err := json.Marshal(map[string]any{
			"edits": []map[string]any{{
				"file":        target,
				"start_line":  2,
				"end_line":    2,
				"replacement": "ONE\n",
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		reviewedPlan := dryRunPlanSHA256(t, string(spec))
		writeTestFile(t, target, "changed header\none\n")

		code, stdout, stderr := invoke(
			[]string{"edit", "--apply", "--expect-plan", reviewedPlan, "--json"},
			string(spec),
		)

		if code != 1 || stdout != "" {
			t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
		}
		payload := decodeJSONError(t, stderr)
		if payload.Error.Code != "plan_mismatch" {
			t.Fatalf("unexpected JSON error: %#v", payload.Error)
		}
		assertFileContent(t, target, "changed header\none\n")
	})

	t.Run("no-op companion target changed", func(t *testing.T) {
		root := t.TempDir()
		changedTarget := filepath.Join(root, "changed.txt")
		noOpTarget := filepath.Join(root, "no-op.txt")
		writeTestFile(t, changedTarget, "one\n")
		writeTestFile(t, noOpTarget, "header\nsame\n")
		spec, err := json.Marshal(map[string]any{
			"edits": []map[string]any{
				{
					"file":        changedTarget,
					"start_line":  1,
					"end_line":    1,
					"replacement": "ONE\n",
				},
				{
					"file":        noOpTarget,
					"start_line":  2,
					"end_line":    2,
					"replacement": "same\n",
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		reviewedPlan := dryRunPlanSHA256(t, string(spec))
		writeTestFile(t, noOpTarget, "changed header\nsame\n")

		code, stdout, stderr := invoke(
			[]string{"edit", "--apply", "--expect-plan", reviewedPlan, "--json"},
			string(spec),
		)

		if code != 1 || stdout != "" {
			t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
		}
		payload := decodeJSONError(t, stderr)
		if payload.Error.Code != "plan_mismatch" {
			t.Fatalf("unexpected JSON error: %#v", payload.Error)
		}
		assertFileContent(t, changedTarget, "one\n")
		assertFileContent(t, noOpTarget, "changed header\nsame\n")
	})
}

func TestApplyWithReviewedPlan(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\n")
	spec := editSpec(t, target, "ONE\n")
	plan := dryRunPlanSHA256(t, spec)

	code, stdout, stderr := invoke(
		[]string{"edit", "--apply", "--expect-plan", plan, "--json"},
		spec,
	)

	if code != 0 || stderr != "" {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	var payload struct {
		Applied    bool   `json:"applied"`
		PlanSHA256 string `json:"plan_sha256"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("cannot decode apply JSON: %v\n%s", err, stdout)
	}
	if !payload.Applied || payload.PlanSHA256 != plan {
		t.Fatalf("unexpected apply JSON: %#v", payload)
	}
	assertFileContent(t, target, "ONE\n")
}

func TestEditDryRunAndApply(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	if err := os.WriteFile(target, []byte("one\r\ntwo\r\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	spec, err := json.Marshal(map[string]any{
		"edits": []map[string]any{{
			"file":        target,
			"start_line":  2,
			"end_line":    2,
			"replacement": "TWO\n",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke([]string{"edit"}, string(spec))
	if code != 0 {
		t.Fatalf("dry-run code = %d, stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "Dry run only") || !strings.Contains(stdout, "--- ") {
		t.Fatalf("unexpected dry-run output:\n%s", stdout)
	}
	if got, err := os.ReadFile(target); err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(got, []byte("one\r\ntwo\r\n")) {
		t.Fatalf("dry run changed target to %q", got)
	}

	code, stdout, stderr = invoke(reviewedApplyArgs(t, string(spec)), string(spec))
	if code != 0 {
		t.Fatalf("apply code = %d, stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "Applied 1 file(s).") {
		t.Fatalf("unexpected apply output:\n%s", stdout)
	}
	if got, err := os.ReadFile(target); err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(got, []byte("one\r\nTWO\r\n")) {
		t.Fatalf("target = %q", got)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(target)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o640 {
			t.Fatalf("mode = %o, want 640", got)
		}
	}
}

func TestEditDryRunShowsCompactUnifiedDiff(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\ntwo\nthree\nfour\nfive\nsix\n")
	replacement := "THREE\n"
	spec, err := json.Marshal(map[string]any{
		"edits": []map[string]any{{
			"file":        target,
			"start_line":  3,
			"end_line":    3,
			"replacement": replacement,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke([]string{"edit"}, string(spec))

	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	for _, unchanged := range []string{"one", "two", "four", "five", "six"} {
		if !strings.Contains(stdout, " "+unchanged+"\n") {
			t.Errorf("unchanged line %q was not emitted as context:\n%s", unchanged, stdout)
		}
		if strings.Contains(stdout, "-"+unchanged+"\n") ||
			strings.Contains(stdout, "+"+unchanged+"\n") {
			t.Errorf("unchanged line %q was reported as changed:\n%s", unchanged, stdout)
		}
	}
	if !strings.Contains(stdout, "-three\n+THREE\n") {
		t.Errorf("replacement was not reported as a focused change:\n%s", stdout)
	}
}

func TestEditDryRunSeparatesDistantChanges(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, strings.Join([]string{
		"one", "two", "three", "four", "five", "six",
		"seven", "eight", "nine", "ten", "eleven", "twelve", "",
	}, "\n"))
	first := "TWO\n"
	second := "ELEVEN\n"
	spec, err := json.Marshal(map[string]any{
		"edits": []map[string]any{
			{
				"file":        target,
				"start_line":  2,
				"end_line":    2,
				"replacement": first,
			},
			{
				"file":        target,
				"start_line":  11,
				"end_line":    11,
				"replacement": second,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke([]string{"edit"}, string(spec))

	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	if got := strings.Count(stdout, "\n@@ "); got != 2 {
		t.Fatalf("hunk markers = %d, want 2:\n%s", got, stdout)
	}
	if strings.Contains(stdout, " six\n") || strings.Contains(stdout, " seven\n") {
		t.Fatalf("distant unchanged lines should be omitted:\n%s", stdout)
	}
	if !strings.Contains(stdout, "-two\n+TWO\n") ||
		!strings.Contains(stdout, "-eleven\n+ELEVEN\n") {
		t.Fatalf("diff omitted a requested replacement:\n%s", stdout)
	}
}

func TestEditDryRunHandlesMidFileReplacementWithoutNewline(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\ntwo\nthree\nfour\n")
	replacement := "TWO"
	spec, err := json.Marshal(map[string]any{
		"edits": []map[string]any{{
			"file":        target,
			"start_line":  2,
			"end_line":    2,
			"replacement": replacement,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke([]string{"edit"}, string(spec))

	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	if strings.Contains(stdout, `\ No newline at end of file`) {
		t.Fatalf("mid-file replacement was reported as an unterminated final line:\n%s", stdout)
	}
	if !strings.Contains(stdout, "-two\n-three\n+TWOthree\n") {
		t.Fatalf("diff does not show the exact merged line:\n%s", stdout)
	}
}

func TestEditDryRunTracksLineNumbersAcrossDistantHunks(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, strings.Join([]string{
		"one", "two", "three", "four", "five", "six",
		"seven", "eight", "nine", "ten", "eleven", "twelve", "",
	}, "\n"))
	first := "TWO-A\nTWO-B\n"
	second := ""
	spec, err := json.Marshal(map[string]any{
		"edits": []map[string]any{
			{
				"file":        target,
				"start_line":  2,
				"end_line":    2,
				"replacement": first,
			},
			{
				"file":        target,
				"start_line":  11,
				"end_line":    11,
				"replacement": second,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke([]string{"edit"}, string(spec))

	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	for _, header := range []string{
		"@@ -1,5 +1,6 @@",
		"@@ -8,5 +9,4 @@",
	} {
		if !strings.Contains(stdout, header) {
			t.Fatalf("diff is missing header %q:\n%s", header, stdout)
		}
	}
}

func TestEditDryRunOmitsNoOpEditFromChangedFile(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, strings.Join([]string{
		"one", "two", "three", "four", "five", "six",
		"seven", "eight", "nine", "ten", "eleven", "twelve", "",
	}, "\n"))
	first := "TWO\n"
	noOp := "eleven\n"
	spec, err := json.Marshal(map[string]any{
		"edits": []map[string]any{
			{
				"file":        target,
				"start_line":  2,
				"end_line":    2,
				"replacement": first,
			},
			{
				"file":        target,
				"start_line":  11,
				"end_line":    11,
				"replacement": noOp,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke([]string{"edit"}, string(spec))

	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr)
	}
	if got := strings.Count(stdout, "\n@@ "); got != 1 {
		t.Fatalf("hunk markers = %d, want 1:\n%s", got, stdout)
	}
	if strings.Contains(stdout, "-eleven\n") || strings.Contains(stdout, "+eleven\n") {
		t.Fatalf("no-op edit was reported as a change:\n%s", stdout)
	}
}

func TestApplyPreservesUnicodeCRLFAndMissingFinalNewline(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	if err := os.WriteFile(target, []byte("α\r\nbeta"), 0o600); err != nil {
		t.Fatal(err)
	}
	spec, err := json.Marshal(map[string]any{
		"edits": []map[string]any{{
			"file":        target,
			"start_line":  2,
			"end_line":    2,
			"replacement": "Β\n",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke(reviewedApplyArgs(t, string(spec)), string(spec))

	if code != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte("α\r\nΒ"); !bytes.Equal(got, want) {
		t.Fatalf("target = %q, want %q", got, want)
	}
}

func TestEditCombinesRelativeAndAbsoluteAliases(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "sample.txt")
	writeTestFile(t, target, "one\ntwo\nthree\n")
	changeWorkingDirectory(t, root)
	spec, err := json.Marshal(map[string]any{
		"edits": []map[string]any{
			{
				"file":        target,
				"start_line":  1,
				"end_line":    1,
				"replacement": "ONE\n",
			},
			{
				"file":        "sample.txt",
				"start_line":  3,
				"end_line":    3,
				"replacement": "THREE\n",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke(reviewedApplyArgs(t, string(spec)), string(spec))

	if code != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	assertFileContent(t, target, "ONE\ntwo\nTHREE\n")
	if !strings.Contains(stdout, "Applied 1 file(s).") {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
}

func TestEditCombinesSymlinkAliasesWithoutReplacingTheLink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "sample.txt")
	alias := filepath.Join(root, "alias.txt")
	writeTestFile(t, target, "one\ntwo\nthree\n")
	if err := os.Symlink(target, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	spec, err := json.Marshal(map[string]any{
		"edits": []map[string]any{
			{
				"file":        target,
				"start_line":  1,
				"end_line":    1,
				"replacement": "ONE\n",
			},
			{
				"file":        alias,
				"start_line":  3,
				"end_line":    3,
				"replacement": "THREE\n",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke(reviewedApplyArgs(t, string(spec)), string(spec))

	if code != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	assertFileContent(t, target, "ONE\ntwo\nTHREE\n")
	if info, err := os.Lstat(alias); err != nil {
		t.Fatal(err)
	} else if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s is no longer a symlink", alias)
	}
	if !strings.Contains(stdout, "Applied 1 file(s).") {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
}

func TestEditMarkerGuardsAndReplacementFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "sample.txt")
	replacement := filepath.Join(root, "replacement.txt")
	writeTestFile(t, target, "# Start\none\n# End\n# Start\ntwo\n# End\n")
	writeTestFile(t, replacement, "TWO\n")
	digest := sha256.Sum256([]byte("two\n"))
	spec, err := json.Marshal(map[string]any{
		"edits": []map[string]any{{
			"file":             target,
			"start":            "# Start",
			"end":              "# End",
			"include_start":    false,
			"include_end":      false,
			"occurrence":       2,
			"end_occurrence":   1,
			"expected_sha256":  hex.EncodeToString(digest[:]),
			"must_contain":     []string{"two", "\n"},
			"replacement_file": replacement,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke(reviewedApplyArgs(t, string(spec)), string(spec))

	if code != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	assertFileContent(t, target, "# Start\none\n# End\n# Start\nTWO\n# End\n")
}

func TestEditRejectsHardLinkedTarget(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "sample.txt")
	alias := filepath.Join(root, "alias.txt")
	writeTestFile(t, target, "one\n")
	if err := os.Link(target, alias); err != nil {
		t.Skipf("hard links unavailable: %v", err)
	}
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

	code, stdout, stderr := invoke([]string{"edit"}, string(spec))

	if code != 1 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "hard-link") {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
	assertFileContent(t, target, "one\n")
	assertFileContent(t, alias, "one\n")
}

func TestEditRejectsOverlappingSections(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\ntwo\nthree\n")
	spec, err := json.Marshal(map[string]any{
		"edits": []map[string]any{
			{
				"file":        target,
				"start_line":  1,
				"end_line":    2,
				"replacement": "first\n",
			},
			{
				"file":        target,
				"start_line":  2,
				"end_line":    3,
				"replacement": "second\n",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke([]string{"edit"}, string(spec))

	if code != 1 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "overlapping edit sections") {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
	assertFileContent(t, target, "one\ntwo\nthree\n")
}

func TestEditAllowsAdjacentSectionsAndSkipsNoOpBackups(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "sample.txt")
	writeTestFile(t, target, "one\ntwo\n")
	changeWorkingDirectory(t, root)
	adjacent, err := json.Marshal(map[string]any{
		"edits": []map[string]any{
			{
				"file":        target,
				"start_line":  1,
				"end_line":    1,
				"replacement": "ONE\n",
			},
			{
				"file":        target,
				"start_line":  2,
				"end_line":    2,
				"replacement": "TWO\n",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := invoke(reviewedApplyArgs(t, string(adjacent)), string(adjacent))
	if code != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	assertFileContent(t, target, "ONE\nTWO\n")

	noOp, err := json.Marshal(map[string]any{
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
	code, stdout, stderr = invoke(
		reviewedApplyArgs(t, string(noOp), "--backup", "--json"),
		string(noOp),
	)
	if code != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	var payload struct {
		ChangedFiles    int    `json:"changed_files"`
		BackupDirectory string `json:"backup_directory"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ChangedFiles != 0 || payload.BackupDirectory != "" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if matches, err := filepath.Glob(filepath.Join(root, ".multi-section-patch-backup-*")); err != nil {
		t.Fatal(err)
	} else if len(matches) != 0 {
		t.Fatalf("no-op created backups: %#v", matches)
	}
}

func TestEditRejectsHashMismatch(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\n")
	spec, err := json.Marshal(map[string]any{
		"edits": []map[string]any{{
			"file":            target,
			"start_line":      1,
			"end_line":        1,
			"replacement":     "ONE\n",
			"expected_sha256": "not-the-current-hash",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke([]string{"edit"}, string(spec))

	if code != 1 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "expected sha256") {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
	assertFileContent(t, target, "one\n")
}

func TestEditRejectsUnknownGuardField(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\n")
	spec := fmt.Sprintf(
		`{"edits":[{"file":%q,"start_line":1,"end_line":1,`+
			`"replacement":"ONE\n","expected_sha265":"wrong"}]}`,
		target,
	)

	code, stdout, stderr := invoke([]string{"edit"}, spec)

	if code != 1 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "unknown field") {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
	assertFileContent(t, target, "one\n")
}

func TestEditRejectsNullMustContainEntry(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\n")
	spec := fmt.Sprintf(
		`{"edits":[{"file":%q,"start_line":1,"end_line":1,`+
			`"replacement":"ONE\n","must_contain":[null]}]}`,
		target,
	)

	code, stdout, stderr := invoke([]string{"edit"}, spec)

	if code != 1 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "must_contain entries cannot be null") {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
	assertFileContent(t, target, "one\n")
}

func TestEditRejectsNULReplacement(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sample.txt")
	writeTestFile(t, target, "one\n")
	spec, err := json.Marshal(map[string]any{
		"edits": []map[string]any{{
			"file":        target,
			"start_line":  1,
			"end_line":    1,
			"replacement": "ONE\x00\n",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke([]string{"edit"}, string(spec))

	if code != 1 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "NUL") {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
	assertFileContent(t, target, "one\n")
}

func TestApplyRejectsStaleContentBeforeWriting(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first.txt")
	second := filepath.Join(root, "second.txt")
	writeTestFile(t, first, "one\n")
	writeTestFile(t, second, "two\n")

	plans, err := planEdits([]sectionItem{
		{
			File:        first,
			StartLine:   intPointer(1),
			EndLine:     intPointer(1),
			Replacement: stringPointer("ONE\n"),
		},
		{
			File:        second,
			StartLine:   intPointer(1),
			EndLine:     intPointer(1),
			Replacement: stringPointer("TWO\n"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, second, "external change\n")

	err = applyPlans(changedPlans(plans), false, os.Rename)

	if err == nil || !strings.Contains(err.Error(), "changed since it was read") {
		t.Fatalf("error = %v", err)
	}
	assertFileContent(t, first, "one\n")
	assertFileContent(t, second, "external change\n")
}

func TestApplyRevalidatesBeforeEachReplacement(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first.txt")
	second := filepath.Join(root, "second.txt")
	writeTestFile(t, first, "one\n")
	writeTestFile(t, second, "two\n")
	plans, err := planEdits([]sectionItem{
		{
			File:        first,
			StartLine:   intPointer(1),
			EndLine:     intPointer(1),
			Replacement: stringPointer("ONE\n"),
		},
		{
			File:        second,
			StartLine:   intPointer(1),
			EndLine:     intPointer(1),
			Replacement: stringPointer("TWO\n"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	changeSecondAfterFirst := func(source, destination string) error {
		calls++
		if err := os.Rename(source, destination); err != nil {
			return err
		}
		if calls == 1 {
			writeTestFile(t, second, "external change\n")
		}
		return nil
	}

	err = applyPlans(changedPlans(plans), false, changeSecondAfterFirst)

	if err == nil || !strings.Contains(err.Error(), "changed since it was read") ||
		!strings.Contains(err.Error(), "rolled back all changes") {
		t.Fatalf("error = %v", err)
	}
	assertFileContent(t, first, "one\n")
	assertFileContent(t, second, "external change\n")
}

func TestApplyRollsBackAfterLaterReplaceFailure(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first.txt")
	second := filepath.Join(root, "second.txt")
	writeTestFile(t, first, "one\n")
	writeTestFile(t, second, "two\n")

	plans, err := planEdits([]sectionItem{
		{
			File:        first,
			StartLine:   intPointer(1),
			EndLine:     intPointer(1),
			Replacement: stringPointer("ONE\n"),
		},
		{
			File:        second,
			StartLine:   intPointer(1),
			EndLine:     intPointer(1),
			Replacement: stringPointer("TWO\n"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	failSecond := func(source, destination string) error {
		calls++
		if calls == 2 {
			return errors.New("simulated replace failure")
		}
		return os.Rename(source, destination)
	}

	err = applyPlans(changedPlans(plans), false, failSecond)

	if err == nil || !strings.Contains(err.Error(), "rolled back all changes") {
		t.Fatalf("error = %v", err)
	}
	assertFileContent(t, first, "one\n")
	assertFileContent(t, second, "two\n")
}

func TestApplyRollsBackAFailingReplaceThatMutatedItsTarget(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first.txt")
	second := filepath.Join(root, "second.txt")
	writeTestFile(t, first, "one\n")
	writeTestFile(t, second, "two\n")
	plans, err := planEdits([]sectionItem{
		{
			File:        first,
			StartLine:   intPointer(1),
			EndLine:     intPointer(1),
			Replacement: stringPointer("ONE\n"),
		},
		{
			File:        second,
			StartLine:   intPointer(1),
			EndLine:     intPointer(1),
			Replacement: stringPointer("TWO\n"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	mutateThenFail := func(source, destination string) error {
		calls++
		if err := os.Rename(source, destination); err != nil {
			return err
		}
		if calls == 2 {
			return errors.New("simulated non-atomic replace failure")
		}
		return nil
	}

	err = applyPlans(changedPlans(plans), false, mutateThenFail)

	if err == nil || !strings.Contains(err.Error(), "rolled back all changes") {
		t.Fatalf("error = %v", err)
	}
	assertFileContent(t, first, "one\n")
	assertFileContent(t, second, "two\n")
}

func TestRollbackPreservesConcurrentMetadataChange(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows exposes only the read-only permission through os.Chmod")
	}
	root := t.TempDir()
	first := filepath.Join(root, "first.txt")
	second := filepath.Join(root, "second.txt")
	writeTestFile(t, first, "one\n")
	writeTestFile(t, second, "two\n")
	plans, err := planEdits([]sectionItem{
		{
			File:        first,
			StartLine:   intPointer(1),
			EndLine:     intPointer(1),
			Replacement: stringPointer("ONE\n"),
		},
		{
			File:        second,
			StartLine:   intPointer(1),
			EndLine:     intPointer(1),
			Replacement: stringPointer("TWO\n"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	changeModeThenFail := func(source, destination string) error {
		calls++
		if calls == 2 {
			return errors.New("simulated replace failure")
		}
		if err := os.Rename(source, destination); err != nil {
			return err
		}
		if calls == 1 {
			return os.Chmod(destination, 0o640)
		}
		return nil
	}

	err = applyPlans(changedPlans(plans), false, changeModeThenFail)

	if err == nil || !strings.Contains(err.Error(), "rollback incomplete") {
		t.Fatalf("error = %v", err)
	}
	assertFileContent(t, first, "ONE\n")
	assertFileContent(t, second, "two\n")
	info, statErr := os.Stat(first)
	if statErr != nil {
		t.Fatal(statErr)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("mode = %o, want 640", got)
	}
	assertReportedRecoveryExists(t, err)
}

func TestRollbackFailureReportsRecoveryCopy(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first.txt")
	second := filepath.Join(root, "second.txt")
	writeTestFile(t, first, "one\n")
	writeTestFile(t, second, "two\n")
	plans, err := planEdits([]sectionItem{
		{
			File:        first,
			StartLine:   intPointer(1),
			EndLine:     intPointer(1),
			Replacement: stringPointer("ONE\n"),
		},
		{
			File:        second,
			StartLine:   intPointer(1),
			EndLine:     intPointer(1),
			Replacement: stringPointer("TWO\n"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	failReplaceAndRollback := func(source, destination string) error {
		calls++
		if calls == 2 {
			return errors.New("simulated replace failure")
		}
		if calls == 3 {
			return errors.New("simulated rollback failure")
		}
		return os.Rename(source, destination)
	}

	err = applyPlans(changedPlans(plans), false, failReplaceAndRollback)

	if err == nil || !strings.Contains(err.Error(), "rollback incomplete") {
		t.Fatalf("error = %v", err)
	}
	assertFileContent(t, first, "ONE\n")
	assertFileContent(t, second, "two\n")
	assertReportedRecoveryExists(t, err)
}

func TestApplyReportsRetainedRecoveryFileWhenCleanupFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory write permissions do not provide a reliable cleanup failure on Windows")
	}
	root := t.TempDir()
	target := filepath.Join(root, "sample.txt")
	writeTestFile(t, target, "one\n")
	plans, err := planEdits([]sectionItem{{
		File:        target,
		StartLine:   intPointer(1),
		EndLine:     intPointer(1),
		Replacement: stringPointer("ONE\n"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(root, 0o700); err != nil {
			t.Errorf("restore temporary directory permissions: %v", err)
		}
	})
	replaceAndLockDirectory := func(source, destination string) error {
		if err := os.Rename(source, destination); err != nil {
			return err
		}
		return os.Chmod(root, 0o500)
	}

	err = applyPlans(changedPlans(plans), false, replaceAndLockDirectory)

	if chmodErr := os.Chmod(root, 0o700); chmodErr != nil {
		t.Fatal(chmodErr)
	}
	if err == nil || !strings.Contains(err.Error(), "cleanup incomplete") {
		t.Fatalf("error = %v", err)
	}
	entries, readErr := os.ReadDir(root)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var recoveryPath string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".multi-section-patch-recovery-") {
			recoveryPath = filepath.Join(root, entry.Name())
			break
		}
	}
	if recoveryPath == "" {
		t.Fatal("cleanup failure did not retain a recovery file")
	}
	if !strings.Contains(err.Error(), recoveryPath) {
		t.Fatalf("error %q does not report retained path %q", err, recoveryPath)
	}
	assertFileContent(t, target, "ONE\n")
}

func TestBackupsUseUniqueContainedDirectories(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	target := filepath.Join(root, "sample.txt")
	writeTestFile(t, target, "one\n")
	if err := os.Symlink(external, filepath.Join(root, ".multi-section-patch-backups")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	changeWorkingDirectory(t, root)
	plans, err := planEdits([]sectionItem{{
		File:        target,
		StartLine:   intPointer(1),
		EndLine:     intPointer(1),
		Replacement: stringPointer("ONE\n"),
	}})
	if err != nil {
		t.Fatal(err)
	}

	first, err := backUpPlans(changedPlans(plans))
	if err != nil {
		t.Fatal(err)
	}
	second, err := backUpPlans(changedPlans(plans))
	if err != nil {
		t.Fatal(err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}

	if first == second {
		t.Fatalf("backup directories are not unique: %q", first)
	}
	for _, backupRoot := range []string{first, second} {
		absolute, err := filepath.Abs(backupRoot)
		if err != nil {
			t.Fatal(err)
		}
		resolved, err := filepath.EvalSymlinks(absolute)
		if err != nil {
			t.Fatal(err)
		}
		relative, err := filepath.Rel(canonicalRoot, resolved)
		if err != nil {
			t.Fatal(err)
		}
		if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			t.Fatalf("backup escaped working directory: %q", absolute)
		}
		entries, err := os.ReadDir(backupRoot)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 2 {
			t.Fatalf("%s contains %d entries, want backup plus manifest", backupRoot, len(entries))
		}
		var backupPath string
		for _, entry := range entries {
			if entry.Name() != "manifest.json" {
				backupPath = filepath.Join(backupRoot, entry.Name())
			}
		}
		if backupPath == "" {
			t.Fatalf("%s does not contain a backup file", backupRoot)
		}
		assertFileContent(t, backupPath, "one\n")
		if _, err := os.Stat(filepath.Join(backupRoot, "manifest.json")); err != nil {
			t.Fatalf("backup manifest: %v", err)
		}
	}
	if entries, err := os.ReadDir(external); err != nil {
		t.Fatal(err)
	} else if len(entries) != 0 {
		t.Fatalf("backup escaped through symlink: %#v", entries)
	}
}

func TestEditReportsBackupDirectory(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "sample.txt")
	writeTestFile(t, target, "one\n")
	changeWorkingDirectory(t, root)
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

	code, stdout, stderr := invoke(
		reviewedApplyArgs(t, string(spec), "--backup", "--json"),
		string(spec),
	)

	if code != 0 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
	var payload struct {
		Applied         bool   `json:"applied"`
		BackupDirectory string `json:"backup_directory"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Applied || payload.BackupDirectory == "" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if _, err := os.Stat(payload.BackupDirectory); err != nil {
		t.Fatalf("backup directory: %v", err)
	}
	assertFileContent(t, target, "ONE\n")
}

func intPointer(value int) *int {
	return &value
}

func stringPointer(value string) *string {
	return &value
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, want %q", path, got, want)
	}
}

func changeWorkingDirectory(t *testing.T, path string) {
	t.Helper()
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}

func assertReportedRecoveryExists(t *testing.T, applyErr error) {
	t.Helper()
	const marker = "recovery copy: "
	message := applyErr.Error()
	index := strings.Index(message, marker)
	if index < 0 {
		t.Fatalf("error does not report a recovery copy: %v", applyErr)
	}
	path := message[index+len(marker):]
	if end := strings.Index(path, ";"); end >= 0 {
		path = path[:end]
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("reported recovery copy %q: %v", path, err)
	}
}
