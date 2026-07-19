package multisectionpatch

import (
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// splitLines separates text at LF boundaries while retaining each original
// terminator so later reads and edits can preserve bytes exactly.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	lines := make([]string, 0, strings.Count(text, "\n")+1)
	for len(text) > 0 {
		index := strings.IndexByte(text, '\n')
		if index < 0 {
			lines = append(lines, text)
			break
		}
		lines = append(lines, text[:index+1])
		text = text[index+1:]
	}
	return lines
}

// containsNUL reports whether data contains a NUL byte so obvious binary input
// can be rejected before UTF-8 and control-character validation.
func containsNUL(data []byte) bool {
	for _, value := range data {
		if value == 0 {
			return true
		}
	}
	return false
}

// validateTextData accepts valid UTF-8 text and rejects NUL bytes or unsupported
// control characters before content reaches selectors or replacements.
func validateTextData(path string, data []byte) error {
	if containsNUL(data) {
		return fmt.Errorf("%s: looks binary; contains NUL", path)
	}
	if !utf8.Valid(data) {
		return fmt.Errorf("%s: not valid UTF-8", path)
	}
	for _, value := range string(data) {
		if unicode.IsControl(value) &&
			value != '\t' &&
			value != '\n' &&
			value != '\r' &&
			value != '\x1b' {
			return fmt.Errorf("%s: looks binary; contains control character U+%04X", path, value)
		}
	}
	return nil
}

// writeDisplayLine prevents selected content from imitating output boundaries
// and escapes unsafe control characters before writing the line.
func writeDisplayLine(writer io.Writer, line string) error {
	if strings.HasPrefix(line, "<<<MULTI_SECTION_PATCH") || strings.HasPrefix(line, "<<<END_MULTI_SECTION_PATCH") {
		if err := writeOutputString(writer, `\`); err != nil {
			return err
		}
	}
	return writeOutputString(writer, escapeControlText(line))
}

// writeOutputString writes exact CLI output and adds operation context to any
// writer failure.
func writeOutputString(writer io.Writer, value string) error {
	if _, err := io.WriteString(writer, value); err != nil {
		return fmt.Errorf("cannot write output: %w", err)
	}
	return nil
}

// writeOutputf formats CLI output directly to the writer and adds operation
// context to any write failure.
func writeOutputf(writer io.Writer, format string, values ...any) error {
	if _, err := fmt.Fprintf(writer, format, values...); err != nil {
		return fmt.Errorf("cannot write output: %w", err)
	}
	return nil
}

// escapeErrorText renders every Unicode control character as a hexadecimal
// escape so untrusted paths and patterns cannot forge terminal output.
func escapeErrorText(text string) string {
	var output strings.Builder
	for _, value := range text {
		if unicode.IsControl(value) {
			fmt.Fprintf(&output, "\\u%04x", value)
			continue
		}
		output.WriteRune(value)
	}
	return output.String()
}

// escapeControlText preserves tabs and valid line endings while rendering all
// other control characters as hexadecimal escapes for safe display.
func escapeControlText(text string) string {
	var output strings.Builder
	for index, value := range text {
		allowedCRLF := value == '\r' &&
			index == len(text)-2 &&
			strings.HasSuffix(text, "\r\n")
		if unicode.IsControl(value) &&
			value != '\t' &&
			value != '\n' &&
			!allowedCRLF {
			fmt.Fprintf(&output, "\\u%04x", value)
			continue
		}
		output.WriteRune(value)
	}
	return output.String()
}
