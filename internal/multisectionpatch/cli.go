package multisectionpatch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

type stringList []string

func (values *stringList) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("must_contain cannot be null")
	}
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*values = []string{single}
		return nil
	}
	var rawValues []json.RawMessage
	if err := json.Unmarshal(data, &rawValues); err != nil {
		return errors.New("must_contain must be a string or list of strings")
	}
	multiple := make([]string, 0, len(rawValues))
	for _, raw := range rawValues {
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return errors.New("must_contain entries cannot be null")
		}
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return errors.New("must_contain must be a string or list of strings")
		}
		multiple = append(multiple, value)
	}
	*values = multiple
	return nil
}

type sectionItem struct {
	File            string     `json:"file"`
	Name            string     `json:"name"`
	StartLine       *int       `json:"start_line"`
	EndLine         *int       `json:"end_line"`
	Start           *string    `json:"start"`
	End             *string    `json:"end"`
	StartRegex      *string    `json:"start_regex"`
	EndRegex        *string    `json:"end_regex"`
	IncludeStart    *bool      `json:"include_start"`
	IncludeEnd      *bool      `json:"include_end"`
	Occurrence      *int       `json:"occurrence"`
	EndOccurrence   *int       `json:"end_occurrence"`
	Replacement     *string    `json:"replacement"`
	ReplacementFile string     `json:"replacement_file"`
	ExpectedSHA256  string     `json:"expected_sha256"`
	MustContain     stringList `json:"must_contain"`
}

// Run executes the Multi Section Patch CLI and returns a process-style exit
// code.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: multi-section-patch read ... | multi-section-patch edit ...")
		return 2
	}

	var err error
	switch args[0] {
	case "read":
		err = runRead(args[1:], stdin, stdout)
	case "edit":
		err = runEdit(args[1:], stdin, stdout)
	default:
		fmt.Fprintln(stderr, "Usage: multi-section-patch read ... | multi-section-patch edit ...")
		return 2
	}
	if err != nil {
		fmt.Fprintf(stderr, "multi-section-patch: error: %s\n", escapeErrorText(err.Error()))
		return 1
	}
	return 0
}

func loadSpecData(path string, stdin io.Reader) ([]byte, error) {
	if path != "" && path != "-" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s: cannot read: %w", path, err)
		}
		return data, nil
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return nil, fmt.Errorf("cannot read JSON from stdin: %w", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, errors.New("provide --spec FILE or JSON on stdin")
	}
	return data, nil
}

func decodeSectionItems(data []byte, key string) ([]sectionItem, error) {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil, errors.New("spec cannot be null")
	}
	var rawItems []json.RawMessage
	if err := json.Unmarshal(data, &rawItems); err != nil {
		var object map[string]json.RawMessage
		if objectErr := json.Unmarshal(data, &object); objectErr != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
		for field := range object {
			if field != key {
				return nil, fmt.Errorf("unknown top-level field %q", field)
			}
		}
		raw, ok := object[key]
		if !ok {
			return nil, fmt.Errorf("spec must contain a %s list", key)
		}
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return nil, fmt.Errorf("%s must be a list, not null", key)
		}
		if err := json.Unmarshal(raw, &rawItems); err != nil {
			return nil, fmt.Errorf("%s must be a list", key)
		}
	}

	items := make([]sectionItem, 0, len(rawItems))
	for _, raw := range rawItems {
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return nil, errors.New("section item cannot be null")
		}
		var selector string
		if err := json.Unmarshal(raw, &selector); err == nil {
			item, err := parseSelector(selector)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
			continue
		}
		var item sectionItem
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
			return nil, errors.New("section item must be an object or selector string")
		}
		for name, value := range fields {
			if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return nil, fmt.Errorf("section field %q cannot be null", name)
			}
		}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&item); err != nil {
			return nil, fmt.Errorf("invalid section item: %w", err)
		}
		if item.File == "" {
			return nil, errors.New("section is missing file")
		}
		if err := validateSelectorFields(item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
