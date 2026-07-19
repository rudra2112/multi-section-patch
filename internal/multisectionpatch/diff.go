package multisectionpatch

import (
	"fmt"
	"strconv"
	"strings"
)

const diffContextLines = 3

type diffLine struct {
	prefix byte
	text   string
}

type diffHunk struct {
	start int
	end   int
}

// unifiedDiff renders only changed regions plus nearby context. Planned edit
// boundaries let this remain linear in file size without a general-purpose
// sequence-diff algorithm.
func unifiedDiff(plan *filePlan) string {
	lines := plannedDiffLines(plan)
	if !diffLinesMatchPlan(lines, plan) {
		// File-level newline preservation or a replacement that joins adjacent
		// source lines can change boundaries outside the selected range. This
		// fallback is still exact and linear, though less granular.
		lines = fallbackDiffLines(string(plan.original), string(plan.updated))
	}

	var output strings.Builder
	fmt.Fprintf(&output, "--- %s (current)\n", strconv.Quote(plan.path))
	fmt.Fprintf(&output, "+++ %s (multi-section-patch)\n", strconv.Quote(plan.path))
	cursor := 0
	oldLine, newLine := 1, 1
	for _, hunk := range diffHunks(lines, diffContextLines) {
		for cursor < hunk.start {
			line := lines[cursor]
			if line.prefix != '+' {
				oldLine++
			}
			if line.prefix != '-' {
				newLine++
			}
			cursor++
		}
		oldLine, newLine = writeDiffHunk(&output, lines, hunk, oldLine, newLine)
		cursor = hunk.end
	}
	return output.String()
}

func plannedDiffLines(plan *filePlan) []diffLine {
	lines := make([]diffLine, 0, len(plan.lines))
	cursor := 0
	for _, edit := range plan.edits {
		for cursor < edit.section.start {
			lines = append(lines, diffLine{prefix: ' ', text: plan.lines[cursor]})
			cursor++
		}
		if edit.replacement == edit.section.content() {
			for cursor < edit.section.end {
				lines = append(lines, diffLine{prefix: ' ', text: plan.lines[cursor]})
				cursor++
			}
			continue
		}
		for cursor < edit.section.end {
			lines = append(lines, diffLine{prefix: '-', text: plan.lines[cursor]})
			cursor++
		}
		for _, replacement := range splitLines(edit.replacement) {
			lines = append(lines, diffLine{prefix: '+', text: replacement})
		}
	}
	for cursor < len(plan.lines) {
		lines = append(lines, diffLine{prefix: ' ', text: plan.lines[cursor]})
		cursor++
	}
	return lines
}

func diffLinesMatchPlan(lines []diffLine, plan *filePlan) bool {
	if !diffStreamHasValidLineBoundaries(lines, '+') ||
		!diffStreamHasValidLineBoundaries(lines, '-') {
		return false
	}
	var original strings.Builder
	var updated strings.Builder
	for _, line := range lines {
		if line.prefix != '+' {
			original.WriteString(line.text)
		}
		if line.prefix != '-' {
			updated.WriteString(line.text)
		}
	}
	return original.String() == string(plan.original) &&
		updated.String() == string(plan.updated)
}

// diffStreamHasValidLineBoundaries ensures an unterminated line appears only
// at the end of its old or new stream, where a unified-diff EOF marker is valid.
func diffStreamHasValidLineBoundaries(lines []diffLine, excludedPrefix byte) bool {
	unterminated := false
	for _, line := range lines {
		if line.prefix == excludedPrefix {
			continue
		}
		if unterminated {
			return false
		}
		unterminated = !strings.HasSuffix(line.text, "\n")
	}
	return true
}

func fallbackDiffLines(original, updated string) []diffLine {
	oldLines := splitLines(original)
	newLines := splitLines(updated)
	prefix := 0
	for prefix < len(oldLines) &&
		prefix < len(newLines) &&
		oldLines[prefix] == newLines[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(oldLines)-prefix &&
		suffix < len(newLines)-prefix &&
		oldLines[len(oldLines)-1-suffix] == newLines[len(newLines)-1-suffix] {
		suffix++
	}

	lines := make([]diffLine, 0, len(oldLines)+len(newLines)-prefix-suffix)
	for _, line := range oldLines[:prefix] {
		lines = append(lines, diffLine{prefix: ' ', text: line})
	}
	for _, line := range oldLines[prefix : len(oldLines)-suffix] {
		lines = append(lines, diffLine{prefix: '-', text: line})
	}
	for _, line := range newLines[prefix : len(newLines)-suffix] {
		lines = append(lines, diffLine{prefix: '+', text: line})
	}
	for _, line := range oldLines[len(oldLines)-suffix:] {
		lines = append(lines, diffLine{prefix: ' ', text: line})
	}
	return lines
}

func diffHunks(lines []diffLine, context int) []diffHunk {
	type changeGroup struct {
		first int
		last  int
	}
	groups := make([]changeGroup, 0)
	current := changeGroup{first: -1}
	unchanged := 0
	for index, line := range lines {
		if line.prefix == ' ' {
			if current.first >= 0 {
				unchanged++
			}
			continue
		}
		if current.first < 0 {
			current = changeGroup{first: index, last: index}
		} else if unchanged > 2*context {
			groups = append(groups, current)
			current = changeGroup{first: index, last: index}
		} else {
			current.last = index
		}
		unchanged = 0
	}
	if current.first >= 0 {
		groups = append(groups, current)
	}

	hunks := make([]diffHunk, 0, len(groups))
	for _, group := range groups {
		start := group.first
		for remaining := context; start > 0 && remaining > 0; remaining-- {
			start--
		}
		end := group.last + 1
		for remaining := context; end < len(lines) && remaining > 0; remaining-- {
			end++
		}
		hunks = append(hunks, diffHunk{start: start, end: end})
	}
	return hunks
}

func writeDiffHunk(
	output *strings.Builder,
	lines []diffLine,
	hunk diffHunk,
	oldLine, newLine int,
) (int, int) {
	oldCount, newCount := 0, 0
	for _, line := range lines[hunk.start:hunk.end] {
		if line.prefix != '+' {
			oldCount++
		}
		if line.prefix != '-' {
			newCount++
		}
	}
	fmt.Fprintf(
		output,
		"@@ -%s +%s @@\n",
		formatDiffRange(oldLine, oldCount),
		formatDiffRange(newLine, newCount),
	)
	for _, line := range lines[hunk.start:hunk.end] {
		writeDiffLine(output, string(line.prefix), line.text)
	}
	return oldLine + oldCount, newLine + newCount
}

func formatDiffRange(start, count int) string {
	switch count {
	case 0:
		return fmt.Sprintf("%d,0", start-1)
	case 1:
		return strconv.Itoa(start)
	default:
		return fmt.Sprintf("%d,%d", start, count)
	}
}

func writeDiffLine(output *strings.Builder, prefix, line string) {
	output.WriteString(prefix)
	output.WriteString(escapeControlText(line))
	if !strings.HasSuffix(line, "\n") {
		output.WriteString("\n\\ No newline at end of file\n")
	}
}
