package multisectionpatch

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type section struct {
	path  string
	name  string
	start int
	end   int
	lines []string
}

// content joins the section's zero-based half-open line range without changing
// the original line terminators.
func (s section) content() string {
	return strings.Join(s.lines[s.start:s.end], "")
}

// digest returns the lowercase SHA-256 digest of the exact selected content.
func (s section) digest() string {
	sum := sha256.Sum256([]byte(s.content()))
	return hex.EncodeToString(sum[:])
}

type readOptions struct {
	specPath     string
	context      int
	json         bool
	lineNumbers  bool
	selectors    []string
	readFromSpec bool
}

// runRead parses and resolves every requested section before emitting human or
// JSON output, preventing a later invalid selector from producing partial data.
func runRead(args []string, stdin io.Reader, stdout io.Writer) error {
	options, err := parseReadOptions(args)
	if err != nil {
		return err
	}

	var items []sectionItem
	if options.readFromSpec {
		data, err := loadSpecData(options.specPath, stdin)
		if err != nil {
			return err
		}
		items, err = decodeSectionItems(data, "sections")
		if err != nil {
			return err
		}
	} else {
		items = make([]sectionItem, 0, len(options.selectors))
		for _, selector := range options.selectors {
			item, err := parseSelector(selector)
			if err != nil {
				return err
			}
			items = append(items, item)
		}
	}

	sections := make([]section, 0, len(items))
	for _, item := range items {
		resolved, err := resolveSection(item)
		if err != nil {
			return err
		}
		sections = append(sections, resolved)
	}

	if options.json {
		return writeSectionsJSON(stdout, sections)
	}
	for _, selected := range sections {
		startLine, endLine := displayRange(selected)
		if err := writeOutputf(
			stdout,
			"<<<MULTI_SECTION_PATCH path=%s name=%s lines=%d-%d sha256=%s>>>\n",
			strconv.Quote(selected.path),
			strconv.Quote(selected.name),
			startLine,
			endLine,
			selected.digest(),
		); err != nil {
			return err
		}
		outStart := max(0, selected.start-options.context)
		outEnd := min(len(selected.lines), selected.end+options.context)
		for index := outStart; index < outEnd; index++ {
			if options.lineNumbers {
				if err := writeOutputf(stdout, "%6d| ", index+1); err != nil {
					return err
				}
			}
			line := selected.lines[index]
			if err := writeDisplayLine(stdout, line); err != nil {
				return err
			}
			if line != "" && !strings.HasSuffix(line, "\n") && !strings.HasSuffix(line, "\r") {
				if err := writeOutputString(stdout, "\n"); err != nil {
					return err
				}
			}
		}
		if err := writeOutputf(
			stdout,
			"<<<END_MULTI_SECTION_PATCH path=%s>>>\n",
			strconv.Quote(selected.path),
		); err != nil {
			return err
		}
	}
	return nil
}

// parseReadOptions separates read flags from inline selectors, defaults to
// numbered human output, and falls back to specification input when needed.
func parseReadOptions(args []string) (readOptions, error) {
	options := readOptions{lineNumbers: true}
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--spec":
			index++
			if index == len(args) {
				return options, errors.New("--spec requires a file")
			}
			options.specPath = args[index]
			options.readFromSpec = true
		case "--context":
			index++
			if index == len(args) {
				return options, errors.New("--context requires a non-negative integer")
			}
			value, err := strconv.Atoi(args[index])
			if err != nil || value < 0 {
				return options, errors.New("--context requires a non-negative integer")
			}
			options.context = value
		case "--json":
			options.json = true
		case "--no-line-numbers":
			options.lineNumbers = false
		case "--":
			options.selectors = append(options.selectors, args[index+1:]...)
			index = len(args)
		default:
			if strings.HasPrefix(args[index], "-") {
				return options, fmt.Errorf("unknown read option %q", args[index])
			}
			options.selectors = append(options.selectors, args[index])
		}
	}
	if len(options.selectors) == 0 {
		options.readFromSpec = true
	}
	if options.specPath != "" && len(options.selectors) != 0 {
		return options, errors.New("use selectors or --spec, not both")
	}
	return options, nil
}

// parseSelector converts an inline file selector into numeric, literal-marker,
// or regular-expression fields using the final @ as the file separator.
func parseSelector(selector string) (sectionItem, error) {
	if _, err := os.Stat(selector); err == nil {
		return sectionItem{File: selector}, nil
	}
	at := strings.LastIndex(selector, "@")
	if at < 0 {
		return sectionItem{File: selector}, nil
	}

	item := sectionItem{File: selector[:at]}
	raw := selector[at+1:]
	numeric, err := regexp.MatchString(`^\d*:\d*$`, raw)
	if err != nil {
		return item, err
	}
	if numeric {
		parts := strings.SplitN(raw, ":", 2)
		if parts[0] != "" {
			value, err := strconv.Atoi(parts[0])
			if err != nil {
				return item, fmt.Errorf("invalid line number %q", parts[0])
			}
			item.StartLine = &value
		}
		if parts[1] != "" {
			value, err := strconv.Atoi(parts[1])
			if err != nil {
				return item, fmt.Errorf("invalid line number %q", parts[1])
			}
			item.EndLine = &value
		}
		return item, nil
	}
	if marker := strings.Index(raw, ".."); marker >= 0 {
		if start := raw[:marker]; start != "" {
			item.Start, item.StartRegex = inlinePattern(start)
		}
		if end := raw[marker+2:]; end != "" {
			item.End, item.EndRegex = inlinePattern(end)
		}
		return item, nil
	}
	if raw != "" {
		item.Start, item.StartRegex = inlinePattern(raw)
	}
	return item, nil
}

// inlinePattern treats a slash-delimited value as a regular expression and
// every other value as a literal line marker.
func inlinePattern(value string) (literal, regex *string) {
	if len(value) >= 2 && strings.HasPrefix(value, "/") && strings.HasSuffix(value, "/") {
		unwrapped := value[1 : len(value)-1]
		return nil, &unwrapped
	}
	return &value, nil
}

// resolveSection reads and validates a file, resolves the requested line
// bounds, and records the display name alongside the original line data.
func resolveSection(item sectionItem) (section, error) {
	path, data, err := readTextFile(item.File)
	if err != nil {
		return section{}, err
	}
	lines := splitLines(string(data))
	start, end, err := sectionRange(item, lines)
	if err != nil {
		return section{}, err
	}
	name := item.Name
	if name == "" {
		name = item.File
	}
	return section{path: path, name: name, start: start, end: end, lines: lines}, nil
}

// readTextFile resolves symlinks to a canonical path, verifies the opened
// target is a regular file, and returns validated UTF-8 text bytes.
func readTextFile(name string) (string, []byte, error) {
	absolute, err := filepath.Abs(name)
	if err != nil {
		return "", nil, fmt.Errorf("%s: cannot resolve path: %w", name, err)
	}
	path, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", nil, fmt.Errorf("%s: cannot resolve path: %w", name, err)
	}
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		return "", nil, fmt.Errorf("%s: cannot stat: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return "", nil, fmt.Errorf("%s: not a regular file", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return "", nil, fmt.Errorf("%s: cannot open: %w", path, err)
	}
	info, err = file.Stat()
	if err != nil {
		_ = file.Close()
		return "", nil, fmt.Errorf("%s: cannot stat after opening: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return "", nil, fmt.Errorf("%s: not a regular file", path)
	}
	data, err := io.ReadAll(file)
	if err != nil {
		_ = file.Close()
		return "", nil, fmt.Errorf("%s: cannot read: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return "", nil, fmt.Errorf("%s: cannot close after reading: %w", path, err)
	}
	if err := validateTextData(path, data); err != nil {
		return "", nil, err
	}
	return path, data, nil
}

// sectionRange validates a selector and resolves it to a zero-based half-open
// range, applying marker occurrence and inclusion options when configured.
func sectionRange(item sectionItem, lines []string) (int, int, error) {
	if err := validateSelectorFields(item); err != nil {
		return 0, 0, err
	}
	if item.StartLine != nil || item.EndLine != nil {
		startLine := 1
		endLine := len(lines)
		if item.StartLine != nil {
			startLine = *item.StartLine
		}
		if item.EndLine != nil {
			endLine = *item.EndLine
		}
		if startLine < 1 || endLine < startLine || startLine > len(lines) || endLine > len(lines) {
			return 0, 0, fmt.Errorf("%s: invalid line range %d:%d", itemName(item), startLine, endLine)
		}
		return startLine - 1, endLine, nil
	}

	includeStart := true
	if item.IncludeStart != nil {
		includeStart = *item.IncludeStart
	}
	includeEnd := false
	if item.IncludeEnd != nil {
		includeEnd = *item.IncludeEnd
	}
	occurrence, err := occurrenceValue("occurrence", item.Occurrence)
	if err != nil {
		return 0, 0, err
	}
	endOccurrence, err := occurrenceValue("end_occurrence", item.EndOccurrence)
	if err != nil {
		return 0, 0, err
	}

	start := 0
	endSearch := 0
	if pattern, ok := itemStartPattern(item); ok {
		match, err := findLine(lines, pattern, 0, occurrence)
		if err != nil {
			return 0, 0, err
		}
		start = match
		if !includeStart {
			start++
		}
		endSearch = match + 1
	}

	end := len(lines)
	if pattern, ok := itemEndPattern(item); ok {
		match, err := findLine(lines, pattern, endSearch, endOccurrence)
		if err != nil {
			return 0, 0, err
		}
		end = match
		if includeEnd {
			end++
		}
	}
	if end < start {
		return 0, 0, fmt.Errorf("%s: end marker resolved before start marker", itemName(item))
	}
	return start, end, nil
}

type linePattern struct {
	text  string
	regex bool
}

// itemStartPattern returns the configured regular-expression or literal start
// marker and reports whether a start bound exists.
func itemStartPattern(item sectionItem) (linePattern, bool) {
	if item.StartRegex != nil {
		return linePattern{text: *item.StartRegex, regex: true}, true
	}
	if item.Start != nil {
		return linePattern{text: *item.Start}, true
	}
	return linePattern{}, false
}

// itemEndPattern returns the configured regular-expression or literal end
// marker and reports whether an end bound exists.
func itemEndPattern(item sectionItem) (linePattern, bool) {
	if item.EndRegex != nil {
		return linePattern{text: *item.EndRegex, regex: true}, true
	}
	if item.End != nil {
		return linePattern{text: *item.End}, true
	}
	return linePattern{}, false
}

// validateSelectorFields rejects mixed selector families and marker-only
// options that have no literal or regular-expression marker to modify.
func validateSelectorFields(item sectionItem) error {
	numeric := item.StartLine != nil || item.EndLine != nil
	literal := item.Start != nil || item.End != nil
	regex := item.StartRegex != nil || item.EndRegex != nil
	families := 0
	for _, used := range []bool{numeric, literal, regex} {
		if used {
			families++
		}
	}
	if families > 1 {
		return fmt.Errorf("%s: use one selector family: lines, literal markers, or regex markers", itemName(item))
	}
	hasMarkerOptions := item.IncludeStart != nil ||
		item.IncludeEnd != nil ||
		item.Occurrence != nil ||
		item.EndOccurrence != nil
	if hasMarkerOptions && !literal && !regex {
		return fmt.Errorf("%s: marker options require a literal or regex selector", itemName(item))
	}
	return nil
}

// itemName returns the user-facing section name, falling back to its file path
// and then a generic label for diagnostics.
func itemName(item sectionItem) string {
	if item.Name != "" {
		return item.Name
	}
	if item.File != "" {
		return item.File
	}
	return "section"
}

// occurrenceValue defaults an omitted marker occurrence to one and rejects
// non-positive counts.
func occurrenceValue(name string, value *int) (int, error) {
	if value == nil {
		return 1, nil
	}
	if *value < 1 {
		return 0, fmt.Errorf("%s must be at least 1", name)
	}
	return *value, nil
}

// findLine scans from start for the requested occurrence, using substring
// matching for literals or one compiled RE2 expression over logical line text.
func findLine(lines []string, pattern linePattern, start, occurrence int) (int, error) {
	text := pattern.text
	var compiled *regexp.Regexp
	if pattern.regex {
		var err error
		compiled, err = regexp.Compile(text)
		if err != nil {
			return 0, fmt.Errorf("invalid regex %q: %w", text, err)
		}
	}
	count := 0
	for index := start; index < len(lines); index++ {
		matched := strings.Contains(lines[index], text)
		if compiled != nil {
			line := strings.TrimSuffix(strings.TrimSuffix(lines[index], "\n"), "\r")
			matched = compiled.MatchString(line)
		}
		if matched {
			count++
			if count == occurrence {
				return index, nil
			}
		}
	}
	kind := "text"
	if compiled != nil {
		kind = "regex"
	}
	return 0, fmt.Errorf("%s pattern not found after line %d: %s", kind, start+1, text)
}

// writeSectionsJSON serializes resolved ranges, exact content, and digests as
// stable indented JSON without HTML escaping.
func writeSectionsJSON(writer io.Writer, sections []section) error {
	type result struct {
		File      string `json:"file"`
		Name      string `json:"name"`
		StartLine int    `json:"start_line"`
		EndLine   int    `json:"end_line"`
		SHA256    string `json:"sha256"`
		Content   string `json:"content"`
	}
	payload := struct {
		Sections []result `json:"sections"`
	}{Sections: make([]result, 0, len(sections))}
	for _, selected := range sections {
		startLine, endLine := displayRange(selected)
		payload.Sections = append(payload.Sections, result{
			File:      selected.path,
			Name:      selected.name,
			StartLine: startLine,
			EndLine:   endLine,
			SHA256:    selected.digest(),
			Content:   selected.content(),
		})
	}
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

// displayRange converts a zero-based half-open range into one-based inclusive
// line numbers, representing an empty file as 1:0.
func displayRange(selected section) (int, int) {
	if len(selected.lines) == 0 {
		return 1, 0
	}
	return selected.start + 1, selected.end
}
