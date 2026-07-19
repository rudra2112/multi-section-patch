package multisectionpatch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func BenchmarkReadSizeDoubling(b *testing.B) {
	for _, lineCount := range []int{10_000, 20_000, 40_000} {
		content := []byte(strings.Repeat("line\n", lineCount))
		path := filepath.Join(b.TempDir(), fmt.Sprintf("%d.txt", lineCount))
		if err := os.WriteFile(path, content, 0o600); err != nil {
			b.Fatal(err)
		}
		b.Run(fmt.Sprintf("lines-%d", lineCount), func(b *testing.B) {
			b.SetBytes(int64(len(content)))
			for iteration := 0; iteration < b.N; iteration++ {
				if _, err := resolveSection(sectionItem{File: path}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkEditSizeDoubling(b *testing.B) {
	for _, lineCount := range []int{10_000, 20_000, 40_000} {
		original := []byte(strings.Repeat("line\n", lineCount))
		lines := splitLines(string(original))
		edits := evenlySpacedEdits(lines, 32)
		b.Run(fmt.Sprintf("lines-%d", lineCount), func(b *testing.B) {
			b.SetBytes(int64(len(original)))
			for iteration := 0; iteration < b.N; iteration++ {
				plan := &filePlan{
					original: original,
					lines:    lines,
					edits:    append([]plannedEdit(nil), edits...),
				}
				if err := finishPlan(plan); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkManyEdits(b *testing.B) {
	original := []byte(strings.Repeat("line\n", 100_000))
	lines := splitLines(string(original))
	for _, editCount := range []int{10, 100, 1_000} {
		edits := evenlySpacedEdits(lines, editCount)
		b.Run(fmt.Sprintf("edits-%d", editCount), func(b *testing.B) {
			for iteration := 0; iteration < b.N; iteration++ {
				plan := &filePlan{
					original: original,
					lines:    lines,
					edits:    append([]plannedEdit(nil), edits...),
				}
				if err := finishPlan(plan); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkDiffManyHunks(b *testing.B) {
	original := []byte(strings.Repeat("line\n", 100_000))
	lines := splitLines(string(original))
	for _, editCount := range []int{10, 100, 1_000} {
		edits := evenlySpacedEdits(lines, editCount)
		plan := &filePlan{
			path:     "benchmark.txt",
			original: original,
			lines:    lines,
			edits:    append([]plannedEdit(nil), edits...),
		}
		if err := finishPlan(plan); err != nil {
			b.Fatal(err)
		}
		b.Run(fmt.Sprintf("hunks-%d", editCount), func(b *testing.B) {
			for iteration := 0; iteration < b.N; iteration++ {
				unifiedDiff(plan)
			}
		})
	}
}

func evenlySpacedEdits(lines []string, count int) []plannedEdit {
	edits := make([]plannedEdit, 0, count)
	step := len(lines) / count
	for index := 0; index < count; index++ {
		line := index * step
		edits = append(edits, plannedEdit{
			section: section{
				start: line,
				end:   line + 1,
				lines: lines,
			},
			replacement: "changed\n",
		})
	}
	return edits
}
