package multisectionpatch

import (
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

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

func containsNUL(data []byte) bool {
	for _, value := range data {
		if value == 0 {
			return true
		}
	}
	return false
}

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

func writeDisplayLine(writer io.Writer, line string) error {
	if strings.HasPrefix(line, "<<<MULTI_SECTION_PATCH") || strings.HasPrefix(line, "<<<END_MULTI_SECTION_PATCH") {
		if err := writeOutputString(writer, `\`); err != nil {
			return err
		}
	}
	return writeOutputString(writer, escapeControlText(line))
}

func writeOutputString(writer io.Writer, value string) error {
	if _, err := io.WriteString(writer, value); err != nil {
		return fmt.Errorf("cannot write output: %w", err)
	}
	return nil
}

func writeOutputf(writer io.Writer, format string, values ...any) error {
	if _, err := fmt.Fprintf(writer, format, values...); err != nil {
		return fmt.Errorf("cannot write output: %w", err)
	}
	return nil
}

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
