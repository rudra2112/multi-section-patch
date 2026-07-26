package multisectionpatch

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
	specPath   string
	expectPlan string
	apply      bool
	backup     bool
	json       bool
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

// runEdit validates and plans the complete request, emits its diff by default,
// and invokes the guarded apply pipeline only when --apply is present.
func runEdit(args []string, stdin io.Reader, stdout io.Writer) error {
	options, err := parseEditOptions(args)
	if err != nil {
		return annotateError(err, errorContext{code: errorInvalidOption})
	}
	data, err := loadSpecData(options.specPath, stdin)
	if err != nil {
		return annotateError(err, errorContext{code: errorSpecReadFailed})
	}
	items, err := decodeSectionItems(data, "edits")
	if err != nil {
		return annotateError(err, errorContext{code: errorInvalidSpec})
	}
	plans, err := planEdits(items)
	if err != nil {
		return annotateError(err, errorContext{code: errorEditPlanFailed})
	}

	changed := changedPlans(plans)
	planDigest, err := planSHA256(plans)
	if err != nil {
		return annotateError(err, errorContext{code: errorEditPlanFailed})
	}
	if options.apply && options.expectPlan != planDigest {
		return annotateError(
			errors.New(
				"reviewed plan no longer matches; run a new dry run and review it before applying",
			),
			errorContext{code: errorPlanMismatch},
		)
	}
	diffs := make([]string, 0, len(changed))
	for _, plan := range changed {
		diffs = append(diffs, unifiedDiff(plan))
	}
	if !options.apply {
		if options.json {
			if err := writeEditJSON(
				stdout,
				diffs,
				len(changed),
				planDigest,
				false,
				"",
			); err != nil {
				return annotateError(err, errorContext{code: errorOutputFailed})
			}
		} else if len(diffs) == 0 {
			if err := writeOutputString(stdout, "No changes.\n"); err != nil {
				return annotateError(err, errorContext{code: errorOutputFailed})
			}
		} else {
			for _, diff := range diffs {
				if err := writeOutputString(stdout, diff); err != nil {
					return annotateError(err, errorContext{code: errorOutputFailed})
				}
			}
		}
		if !options.json {
			if err := writeOutputf(stdout, "Plan SHA-256: %s\n", planDigest); err != nil {
				return annotateError(err, errorContext{code: errorOutputFailed})
			}
			if err := writeOutputString(
				stdout,
				"Dry run only. Re-run with --apply --expect-plan "+
					planDigest+" to write changes.\n",
			); err != nil {
				return annotateError(err, errorContext{code: errorOutputFailed})
			}
		}
		return nil
	}

	if !options.json {
		if len(diffs) == 0 {
			if err := writeOutputString(stdout, "No changes.\n"); err != nil {
				return annotateError(err, errorContext{code: errorOutputFailed})
			}
		} else {
			for _, diff := range diffs {
				if err := writeOutputString(stdout, diff); err != nil {
					return annotateError(err, errorContext{code: errorOutputFailed})
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
			return annotateError(
				fmt.Errorf("%w; backups retained at %s", err, strconv.Quote(backupDirectory)),
				errorContext{code: errorApplyFailed},
			)
		}
		return annotateError(err, errorContext{code: errorApplyFailed})
	}
	if options.json {
		if err := writeEditJSON(
			stdout,
			diffs,
			len(changed),
			planDigest,
			true,
			backupDirectory,
		); err != nil {
			return annotateError(err, errorContext{code: errorOutputFailed})
		}
	} else {
		if err := writeOutputf(stdout, "Applied %d file(s).\n", len(changed)); err != nil {
			return annotateError(err, errorContext{code: errorOutputFailed})
		}
		if backupDirectory != "" {
			if err := writeOutputf(
				stdout,
				"Backups: %s\n",
				strconv.Quote(backupDirectory),
			); err != nil {
				return annotateError(err, errorContext{code: errorOutputFailed})
			}
		}
	}
	return nil
}

// parseEditOptions recognizes the edit subcommand's specification, reviewed
// plan, apply, backup, and JSON flags and rejects invalid combinations.
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
		case "--expect-plan":
			index++
			if index == len(args) {
				return options, errors.New("--expect-plan requires a SHA-256 value")
			}
			options.expectPlan = args[index]
		case "--backup":
			options.backup = true
		case "--json":
			options.json = true
		default:
			return options, fmt.Errorf("unknown edit option %q", args[index])
		}
	}
	if options.apply && options.expectPlan == "" {
		return options, errors.New("--apply requires --expect-plan from a reviewed dry run")
	}
	if !options.apply && options.expectPlan != "" {
		return options, errors.New("--expect-plan requires --apply")
	}
	if options.expectPlan != "" && !validPlanSHA256(options.expectPlan) {
		return options, errors.New("--expect-plan must be a 64-character lowercase SHA-256")
	}
	return options, nil
}

// validPlanSHA256 reports whether a reviewed-plan token is canonical lowercase
// SHA-256 text.
func validPlanSHA256(value string) bool {
	if len(value) != sha256.Size*2 || value != strings.ToLower(value) {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}

// planEdits snapshots each target by filesystem identity, validates selectors
// and guards, groups edits per file, and computes the final bytes before writes.
func planEdits(items []sectionItem) ([]*filePlan, error) {
	plans := make([]*filePlan, 0)
	plansByIdentity := make(map[string]*filePlan)
	snapshots := newFileSnapshotCache()
	for index, item := range items {
		if item.Replacement == nil && item.ReplacementFile == "" {
			return nil, annotateItemError(
				fmt.Errorf("%s: missing replacement or replacement_file", itemName(item)),
				errorEditPlanFailed,
				index,
				item,
			)
		}
		if item.Replacement != nil && item.ReplacementFile != "" {
			return nil, annotateItemError(
				fmt.Errorf("%s: use replacement or replacement_file, not both", itemName(item)),
				errorEditPlanFailed,
				index,
				item,
			)
		}

		snapshot, err := snapshots.read(item.File)
		if err != nil {
			return nil, annotateItemError(
				annotateFieldError(err, "file"),
				errorEditPlanFailed,
				index,
				item,
			)
		}
		if err := validateTargetForEdit(snapshot.path, snapshot.info); err != nil {
			return nil, annotateItemError(
				annotateFieldError(err, "file"),
				errorEditPlanFailed,
				index,
				item,
			)
		}
		if snapshot.links > 1 {
			return nil, annotateItemError(
				annotateFieldError(
					fmt.Errorf(
						"%s: hard-link target has %d links; refusing ambiguous edit",
						snapshot.path,
						snapshot.links,
					),
					"file",
				),
				errorEditPlanFailed,
				index,
				item,
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
			return nil, annotateItemError(
				annotateFieldError(
					fmt.Errorf("%s: changed while planning edits", snapshot.path),
					"file",
				),
				errorEditPlanFailed,
				index,
				item,
			)
		}

		start, end, err := sectionRange(item, plan.lines)
		if err != nil {
			return nil, annotateItemError(err, errorEditPlanFailed, index, item)
		}
		selected := section{
			path:  plan.path,
			name:  itemName(item),
			start: start,
			end:   end,
			lines: plan.lines,
		}
		if item.ExpectedSHA256 != "" && item.ExpectedSHA256 != selected.digest() {
			return nil, annotateItemError(
				annotateFieldError(
					fmt.Errorf(
						"%s: expected sha256 %s, found %s",
						plan.path,
						item.ExpectedSHA256,
						selected.digest(),
					),
					"expected_sha256",
				),
				errorEditPlanFailed,
				index,
				item,
			)
		}
		for _, required := range item.MustContain {
			if !strings.Contains(selected.content(), required) {
				return nil, annotateItemError(
					annotateFieldError(
						fmt.Errorf(
							"%s: selected section does not contain %q",
							plan.path,
							required,
						),
						"must_contain",
					),
					errorEditPlanFailed,
					index,
					item,
				)
			}
		}
		replacement, err := replacementText(item, snapshots)
		if err != nil {
			return nil, annotateItemError(err, errorEditPlanFailed, index, item)
		}
		if err := validateTextData(itemName(item)+" replacement", []byte(replacement)); err != nil {
			field := "replacement_file"
			if item.Replacement != nil {
				field = "replacement"
			}
			return nil, annotateItemError(
				annotateFieldError(err, field),
				errorEditPlanFailed,
				index,
				item,
			)
		}
		plan.edits = append(plan.edits, plannedEdit{
			section:     selected,
			replacement: normalizeNewlines(replacement, string(plan.original)),
		})
	}

	for _, plan := range plans {
		if err := finishPlan(plan); err != nil {
			return nil, annotateError(
				err,
				errorContext{code: errorEditPlanFailed, file: plan.path},
			)
		}
	}
	return plans, nil
}

// replacementText returns an inline replacement or reads a replacement file
// through the same request-local snapshot cache used for edit targets.
func replacementText(item sectionItem, snapshots *fileSnapshotCache) (string, error) {
	if item.Replacement != nil {
		return *item.Replacement, nil
	}
	snapshot, err := snapshots.read(item.ReplacementFile)
	if err != nil {
		return "", annotateFieldError(err, "replacement_file")
	}
	return string(snapshot.data), nil
}

// normalizeNewlines converts CRLF sequences in replacement text to LF, then
// converts LF to CRLF when the target uses CRLF line endings.
func normalizeNewlines(replacement, original string) string {
	replacement = strings.ReplaceAll(replacement, "\r\n", "\n")
	if strings.Contains(original, "\r\n") {
		return strings.ReplaceAll(replacement, "\n", "\r\n")
	}
	return replacement
}

// finishPlan sorts and rejects overlapping edits, streams replacements between
// untouched lines, and preserves the target's final-newline state.
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

// preserveFinalNewline adds or removes trailing LF or CRLF bytes so non-empty
// output retains the original final-newline state; fully deleted files stay empty.
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

// changedPlans filters out byte-identical plans so dry runs and the apply
// pipeline operate only on files whose content would actually change.
func changedPlans(plans []*filePlan) []*filePlan {
	changed := make([]*filePlan, 0, len(plans))
	for _, plan := range plans {
		if !bytes.Equal(plan.original, plan.updated) {
			changed = append(changed, plan)
		}
	}
	return changed
}

// planSHA256 returns a deterministic digest of the exact ordered file plans
// that the diff and apply pipeline consume.
func planSHA256(plans []*filePlan) (string, error) {
	type digestEdit struct {
		Start             int    `json:"start"`
		End               int    `json:"end"`
		ReplacementSHA256 string `json:"replacement_sha256"`
	}
	type digestFile struct {
		Path           string       `json:"path"`
		Identity       string       `json:"identity"`
		Mode           uint32       `json:"mode"`
		OriginalSHA256 string       `json:"original_sha256"`
		UpdatedSHA256  string       `json:"updated_sha256"`
		Edits          []digestEdit `json:"edits"`
	}
	payload := struct {
		Format string       `json:"format"`
		Files  []digestFile `json:"files"`
	}{
		Format: "multi-section-patch-plan-v1",
		Files:  make([]digestFile, 0, len(plans)),
	}
	for _, plan := range plans {
		originalDigest := sha256.Sum256(plan.original)
		updatedDigest := sha256.Sum256(plan.updated)
		file := digestFile{
			Path:           plan.path,
			Identity:       plan.identity,
			Mode:           uint32(plan.info.Mode().Perm()),
			OriginalSHA256: hex.EncodeToString(originalDigest[:]),
			UpdatedSHA256:  hex.EncodeToString(updatedDigest[:]),
			Edits:          make([]digestEdit, 0, len(plan.edits)),
		}
		for _, edit := range plan.edits {
			replacementDigest := sha256.Sum256([]byte(edit.replacement))
			file.Edits = append(file.Edits, digestEdit{
				Start:             edit.section.start,
				End:               edit.section.end,
				ReplacementSHA256: hex.EncodeToString(replacementDigest[:]),
			})
		}
		payload.Files = append(payload.Files, file)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("cannot encode edit plan: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

// writeEditJSON serializes the generated diffs, changed-file count, apply
// status, plan digest, and optional backup directory as indented JSON.
func writeEditJSON(
	writer io.Writer,
	diffs []string,
	changed int,
	planDigest string,
	applied bool,
	backupDirectory string,
) error {
	payload := struct {
		Diffs           []string `json:"diffs"`
		ChangedFiles    int      `json:"changed_files"`
		PlanSHA256      string   `json:"plan_sha256"`
		Applied         bool     `json:"applied"`
		BackupDirectory string   `json:"backup_directory,omitempty"`
	}{
		Diffs:           diffs,
		ChangedFiles:    changed,
		PlanSHA256:      planDigest,
		Applied:         applied,
		BackupDirectory: backupDirectory,
	}
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}
