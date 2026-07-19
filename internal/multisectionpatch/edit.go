package multisectionpatch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

type editOptions struct {
	specPath string
	apply    bool
	backup   bool
	json     bool
}

type plannedEdit struct {
	section     section
	replacement string
}

type filePlan struct {
	path     string
	info     os.FileInfo
	identity string
	original []byte
	lines    []string
	edits    []plannedEdit
	updated  []byte
}

func runEdit(args []string, stdin io.Reader, stdout io.Writer) error {
	options, err := parseEditOptions(args)
	if err != nil {
		return err
	}
	data, err := loadSpecData(options.specPath, stdin)
	if err != nil {
		return err
	}
	items, err := decodeSectionItems(data, "edits")
	if err != nil {
		return err
	}
	plans, err := planEdits(items)
	if err != nil {
		return err
	}

	changed := changedPlans(plans)
	diffs := make([]string, 0, len(changed))
	for _, plan := range changed {
		diffs = append(diffs, unifiedDiff(plan))
	}
	if !options.apply {
		if options.json {
			if err := writeEditJSON(stdout, diffs, len(changed), false, ""); err != nil {
				return err
			}
		} else if len(diffs) == 0 {
			if err := writeOutputString(stdout, "No changes.\n"); err != nil {
				return err
			}
		} else {
			for _, diff := range diffs {
				if err := writeOutputString(stdout, diff); err != nil {
					return err
				}
			}
		}
		if !options.json {
			if err := writeOutputString(
				stdout,
				"Dry run only. Re-run with --apply to write changes.\n",
			); err != nil {
				return err
			}
		}
		return nil
	}

	if !options.json {
		if len(diffs) == 0 {
			if err := writeOutputString(stdout, "No changes.\n"); err != nil {
				return err
			}
		} else {
			for _, diff := range diffs {
				if err := writeOutputString(stdout, diff); err != nil {
					return err
				}
			}
		}
	}
	backupDirectory := ""
	if err := applyPlansWithBackupReport(
		changed,
		options.backup,
		os.Rename,
		func(path string) { backupDirectory = path },
	); err != nil {
		if backupDirectory != "" {
			return fmt.Errorf("%w; backups retained at %s", err, strconv.Quote(backupDirectory))
		}
		return err
	}
	if options.json {
		if err := writeEditJSON(stdout, diffs, len(changed), true, backupDirectory); err != nil {
			return err
		}
	} else {
		if err := writeOutputf(stdout, "Applied %d file(s).\n", len(changed)); err != nil {
			return err
		}
		if backupDirectory != "" {
			if err := writeOutputf(
				stdout,
				"Backups: %s\n",
				strconv.Quote(backupDirectory),
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseEditOptions(args []string) (editOptions, error) {
	var options editOptions
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--spec":
			index++
			if index == len(args) {
				return options, errors.New("--spec requires a file")
			}
			options.specPath = args[index]
		case "--apply":
			options.apply = true
		case "--backup":
			options.backup = true
		case "--json":
			options.json = true
		default:
			return options, fmt.Errorf("unknown edit option %q", args[index])
		}
	}
	return options, nil
}

func planEdits(items []sectionItem) ([]*filePlan, error) {
	plans := make([]*filePlan, 0)
	plansByIdentity := make(map[string]*filePlan)
	for _, item := range items {
		if item.Replacement == nil && item.ReplacementFile == "" {
			return nil, fmt.Errorf("%s: missing replacement or replacement_file", itemName(item))
		}
		if item.Replacement != nil && item.ReplacementFile != "" {
			return nil, fmt.Errorf("%s: use replacement or replacement_file, not both", itemName(item))
		}

		snapshot, err := readFileSnapshot(item.File)
		if err != nil {
			return nil, err
		}
		if err := validateTargetForEdit(snapshot.path, snapshot.info); err != nil {
			return nil, err
		}
		if snapshot.links > 1 {
			return nil, fmt.Errorf(
				"%s: hard-link target has %d links; refusing ambiguous edit",
				snapshot.path,
				snapshot.links,
			)
		}
		plan := plansByIdentity[snapshot.identity]
		if plan == nil {
			plan = &filePlan{
				path:     snapshot.path,
				info:     snapshot.info,
				identity: snapshot.identity,
				original: snapshot.data,
				lines:    splitLines(string(snapshot.data)),
			}
			plans = append(plans, plan)
			plansByIdentity[snapshot.identity] = plan
		} else if !bytes.Equal(plan.original, snapshot.data) ||
			plan.info.Mode().Perm() != snapshot.info.Mode().Perm() {
			return nil, fmt.Errorf("%s: changed while planning edits", snapshot.path)
		}

		start, end, err := sectionRange(item, plan.lines)
		if err != nil {
			return nil, err
		}
		selected := section{
			path:  plan.path,
			name:  itemName(item),
			start: start,
			end:   end,
			lines: plan.lines,
		}
		if item.ExpectedSHA256 != "" && item.ExpectedSHA256 != selected.digest() {
			return nil, fmt.Errorf(
				"%s: expected sha256 %s, found %s",
				plan.path,
				item.ExpectedSHA256,
				selected.digest(),
			)
		}
		for _, required := range item.MustContain {
			if !strings.Contains(selected.content(), required) {
				return nil, fmt.Errorf("%s: selected section does not contain %q", plan.path, required)
			}
		}
		replacement, err := replacementText(item)
		if err != nil {
			return nil, err
		}
		if err := validateTextData(itemName(item)+" replacement", []byte(replacement)); err != nil {
			return nil, err
		}
		plan.edits = append(plan.edits, plannedEdit{
			section:     selected,
			replacement: normalizeNewlines(replacement, string(plan.original)),
		})
	}

	for _, plan := range plans {
		if err := finishPlan(plan); err != nil {
			return nil, err
		}
	}
	return plans, nil
}

func replacementText(item sectionItem) (string, error) {
	if item.Replacement != nil {
		return *item.Replacement, nil
	}
	_, data, err := readTextFile(item.ReplacementFile)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func normalizeNewlines(replacement, original string) string {
	replacement = strings.ReplaceAll(replacement, "\r\n", "\n")
	if strings.Contains(original, "\r\n") {
		return strings.ReplaceAll(replacement, "\n", "\r\n")
	}
	return replacement
}

func finishPlan(plan *filePlan) error {
	sort.Slice(plan.edits, func(left, right int) bool {
		return plan.edits[left].section.start < plan.edits[right].section.start
	})
	for index := 1; index < len(plan.edits); index++ {
		current := plan.edits[index].section
		previous := plan.edits[index-1].section
		if current.start < previous.end ||
			(current.start == previous.start && current.end == previous.end) {
			return fmt.Errorf("%s: overlapping edit sections are not allowed", plan.path)
		}
	}

	var updated strings.Builder
	updated.Grow(len(plan.original))
	cursor := 0
	for _, edit := range plan.edits {
		for cursor < edit.section.start {
			updated.WriteString(plan.lines[cursor])
			cursor++
		}
		updated.WriteString(edit.replacement)
		cursor = edit.section.end
	}
	for cursor < len(plan.lines) {
		updated.WriteString(plan.lines[cursor])
		cursor++
	}
	plan.updated = preserveFinalNewline(plan.original, []byte(updated.String()))
	return nil
}

// preserveFinalNewline keeps the file-level newline invariant when an edit reaches EOF.
func preserveFinalNewline(original, updated []byte) []byte {
	originalHasFinalNewline := bytes.HasSuffix(original, []byte("\n"))
	updatedHasFinalNewline := bytes.HasSuffix(updated, []byte("\n"))
	if originalHasFinalNewline && len(updated) != 0 && !updatedHasFinalNewline {
		if bytes.HasSuffix(original, []byte("\r\n")) {
			return append(updated, '\r', '\n')
		}
		return append(updated, '\n')
	}
	if !originalHasFinalNewline {
		for bytes.HasSuffix(updated, []byte("\n")) {
			updated = updated[:len(updated)-1]
			if bytes.HasSuffix(updated, []byte("\r")) {
				updated = updated[:len(updated)-1]
			}
		}
	}
	return updated
}

func changedPlans(plans []*filePlan) []*filePlan {
	changed := make([]*filePlan, 0, len(plans))
	for _, plan := range plans {
		if !bytes.Equal(plan.original, plan.updated) {
			changed = append(changed, plan)
		}
	}
	return changed
}

func writeEditJSON(
	writer io.Writer,
	diffs []string,
	changed int,
	applied bool,
	backupDirectory string,
) error {
	payload := struct {
		Diffs           []string `json:"diffs"`
		ChangedFiles    int      `json:"changed_files"`
		Applied         bool     `json:"applied"`
		BackupDirectory string   `json:"backup_directory,omitempty"`
	}{
		Diffs:           diffs,
		ChangedFiles:    changed,
		Applied:         applied,
		BackupDirectory: backupDirectory,
	}
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}
