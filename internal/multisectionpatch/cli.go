package multisectionpatch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

const (
	errorInvalidOption           = "invalid_option"
	errorSpecReadFailed          = "spec_read_failed"
	errorInvalidSpec             = "invalid_spec"
	errorSectionResolutionFailed = "section_resolution_failed"
	errorEditPlanFailed          = "edit_plan_failed"
	errorPlanMismatch            = "plan_mismatch"
	errorApplyFailed             = "apply_failed"
	errorOutputFailed            = "output_failed"
	errorOperationFailed         = "operation_failed"
)

type stringList []string

// UnmarshalJSON accepts either one string or a list of strings and rejects
// null or non-string values before they reach edit guards.
func (values *stringList) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return fieldError("must_contain", "must_contain cannot be null")
	}
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*values = []string{single}
		return nil
	}
	var rawValues []json.RawMessage
	if err := json.Unmarshal(data, &rawValues); err != nil {
		return fieldError(
			"must_contain",
			"must_contain must be a string or list of strings",
		)
	}
	multiple := make([]string, 0, len(rawValues))
	for _, raw := range rawValues {
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return fieldError("must_contain", "must_contain entries cannot be null")
		}
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return fieldError(
				"must_contain",
				"must_contain must be a string or list of strings",
			)
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

// cliError carries stable machine-readable context while preserving the
// original message and error chain for human diagnostics.
type cliError struct {
	code      string
	itemIndex int
	name      string
	file      string
	field     string
	err       error
}

// Error returns the original diagnostic without adding machine-only fields.
func (err *cliError) Error() string {
	return err.err.Error()
}

// Unwrap exposes the original error to errors.Is and errors.As callers.
func (err *cliError) Unwrap() error {
	return err.err
}

// errorContext describes fields to add without replacing context already
// attached closer to the failure.
type errorContext struct {
	code      string
	itemIndex int
	name      string
	file      string
	field     string
}

// annotateError adds stable error metadata, preferring more specific metadata
// already attached closer to the failure.
func annotateError(err error, context errorContext) error {
	if err == nil {
		return nil
	}
	var existing *cliError
	if errors.As(err, &existing) {
		if existing.code != "" {
			context.code = existing.code
		}
		if existing.itemIndex != 0 {
			context.itemIndex = existing.itemIndex
		}
		if existing.name != "" {
			context.name = existing.name
		}
		if existing.file != "" {
			context.file = existing.file
		}
		if existing.field != "" {
			context.field = existing.field
		}
	}
	return &cliError{
		code:      context.code,
		itemIndex: context.itemIndex,
		name:      context.name,
		file:      context.file,
		field:     context.field,
		err:       err,
	}
}

// annotateItemError adds one-based item and user-supplied section details to a
// failure raised while decoding, resolving, or planning that item.
func annotateItemError(err error, code string, index int, item sectionItem) error {
	return annotateError(err, errorContext{
		code:      code,
		itemIndex: index + 1,
		name:      item.Name,
		file:      item.File,
	})
}

// fieldError creates an invalid-spec error tied to one input field.
func fieldError(field, format string, arguments ...any) error {
	return annotateError(
		fmt.Errorf(format, arguments...),
		errorContext{code: errorInvalidSpec, field: field},
	)
}

// annotateFieldError records the JSON field responsible for an operational
// failure without overriding the stable error code assigned by its caller.
func annotateFieldError(err error, field string) error {
	return annotateError(err, errorContext{field: field})
}

// jsonTypeErrorField returns the final JSON field path component reported by
// encoding/json for a value whose type does not match sectionItem.
func jsonTypeErrorField(err error) string {
	var typeError *json.UnmarshalTypeError
	if !errors.As(err, &typeError) || typeError.Field == "" {
		return ""
	}
	if separator := strings.LastIndexByte(typeError.Field, '.'); separator >= 0 {
		return typeError.Field[separator+1:]
	}
	return typeError.Field
}

// Run dispatches the requested subcommand, writes sanitized diagnostics to
// stderr, and returns 0 for success, 1 for subcommand errors, or 2 when the
// subcommand is missing or unknown.
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
		if commandRequestsJSON(args[0], args[1:]) {
			_ = writeJSONError(stderr, args[0], err)
		} else {
			fmt.Fprintf(stderr, "multi-section-patch: error: %s\n", escapeErrorText(err.Error()))
		}
		return 1
	}
	return 0
}

// commandRequestsJSON reports whether --json is an option rather than a value
// or a read selector after --.
func commandRequestsJSON(command string, args []string) bool {
	for index := 0; index < len(args); index++ {
		if args[index] == "--json" {
			return true
		}
		switch command {
		case "read":
			switch args[index] {
			case "--":
				return false
			case "--spec", "--context":
				index++
			}
		case "edit":
			switch args[index] {
			case "--spec", "--expect-plan":
				index++
			}
		}
	}
	return false
}

// writeJSONError emits one JSON object to stderr while leaving exit status as
// the authoritative success signal.
func writeJSONError(writer io.Writer, command string, err error) error {
	detail := &cliError{code: errorOperationFailed, err: err}
	var annotated *cliError
	if errors.As(err, &annotated) {
		detail = annotated
	}
	payload := struct {
		Error struct {
			Code      string `json:"code"`
			Command   string `json:"command"`
			Message   string `json:"message"`
			ItemIndex int    `json:"item_index,omitempty"`
			Name      string `json:"name,omitempty"`
			File      string `json:"file,omitempty"`
			Field     string `json:"field,omitempty"`
		} `json:"error"`
	}{}
	payload.Error.Code = detail.code
	if payload.Error.Code == "" {
		payload.Error.Code = errorOperationFailed
	}
	payload.Error.Command = command
	payload.Error.Message = err.Error()
	payload.Error.ItemIndex = detail.itemIndex
	payload.Error.Name = detail.name
	payload.Error.File = detail.file
	payload.Error.Field = detail.field
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(payload)
}

// loadSpecData reads a named specification file or standard input, requiring
// non-blank input when no file path is supplied.
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

// decodeSectionItems accepts a bare list or a keyed specification object,
// validates its shape strictly, and converts selector strings or objects into
// section items.
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
		topLevelFields := make([]string, 0, len(object))
		for field := range object {
			topLevelFields = append(topLevelFields, field)
		}
		sort.Strings(topLevelFields)
		for _, field := range topLevelFields {
			if field != key {
				return nil, fieldError(field, "unknown top-level field %q", field)
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
	for index, raw := range rawItems {
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return nil, annotateItemError(
				errors.New("section item cannot be null"),
				errorInvalidSpec,
				index,
				sectionItem{},
			)
		}
		var selector string
		if err := json.Unmarshal(raw, &selector); err == nil {
			item, err := parseSelector(selector)
			if err != nil {
				return nil, annotateItemError(
					err,
					errorInvalidSpec,
					index,
					sectionItem{File: selector},
				)
			}
			items = append(items, item)
			continue
		}
		var item sectionItem
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
			return nil, annotateItemError(
				errors.New("section item must be an object or selector string"),
				errorInvalidSpec,
				index,
				sectionItem{},
			)
		}
		var itemContext struct {
			File string `json:"file"`
			Name string `json:"name"`
		}
		_ = json.Unmarshal(raw, &itemContext)
		item.File = itemContext.File
		item.Name = itemContext.Name
		fieldNames := make([]string, 0, len(fields))
		for name := range fields {
			fieldNames = append(fieldNames, name)
		}
		sort.Strings(fieldNames)
		for _, name := range fieldNames {
			value := fields[name]
			if !sectionFieldAllowed(key, name) {
				var fieldErr error
				if key == "sections" && editOnlyField(name) {
					fieldErr = fieldError(name, "field %q is not valid for read sections", name)
				} else {
					fieldErr = fieldError(name, "unknown field %q", name)
				}
				return nil, annotateItemError(fieldErr, errorInvalidSpec, index, item)
			}
			if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return nil, annotateItemError(
					fieldError(name, "section field %q cannot be null", name),
					errorInvalidSpec,
					index,
					item,
				)
			}
		}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&item); err != nil {
			decodeErr := fmt.Errorf("invalid section item: %w", err)
			if field := jsonTypeErrorField(err); field != "" {
				decodeErr = fieldError(field, "%s", decodeErr)
			}
			return nil, annotateItemError(
				decodeErr,
				errorInvalidSpec,
				index,
				item,
			)
		}
		if key == "edits" {
			if err := validatePresentEditFields(fields, item); err != nil {
				return nil, annotateItemError(err, errorInvalidSpec, index, item)
			}
		}
		if item.File == "" {
			return nil, annotateItemError(
				fieldError("file", "section is missing file"),
				errorInvalidSpec,
				index,
				item,
			)
		}
		if err := validateSelectorFields(item); err != nil {
			return nil, annotateItemError(err, errorInvalidSpec, index, item)
		}
		items = append(items, item)
	}
	return items, nil
}

// validatePresentEditFields rejects edit-only fields that were explicitly
// supplied but would otherwise have no effect on planning or guard checks.
func validatePresentEditFields(fields map[string]json.RawMessage, item sectionItem) error {
	if _, present := fields["expected_sha256"]; present && item.ExpectedSHA256 == "" {
		return fieldError("expected_sha256", "expected_sha256 cannot be empty")
	}
	if _, present := fields["must_contain"]; present {
		if len(item.MustContain) == 0 {
			return fieldError(
				"must_contain",
				"must_contain must contain at least one non-empty string",
			)
		}
		for _, required := range item.MustContain {
			if required == "" {
				return fieldError("must_contain", "must_contain entries cannot be empty")
			}
		}
	}
	if _, present := fields["replacement_file"]; present && item.ReplacementFile == "" {
		return fieldError("replacement_file", "replacement_file cannot be empty")
	}
	return nil
}

// sectionFieldAllowed reports whether a raw item field belongs to the shared
// selector schema or the requested command's edit extension.
func sectionFieldAllowed(key, field string) bool {
	switch field {
	case "file", "name", "start_line", "end_line", "start", "end",
		"start_regex", "end_regex", "include_start", "include_end",
		"occurrence", "end_occurrence":
		return true
	case "replacement", "replacement_file", "expected_sha256", "must_contain":
		return key == "edits"
	default:
		return false
	}
}

// editOnlyField reports whether a known item field has meaning only for edit.
func editOnlyField(field string) bool {
	switch field {
	case "replacement", "replacement_file", "expected_sha256", "must_contain":
		return true
	default:
		return false
	}
}
