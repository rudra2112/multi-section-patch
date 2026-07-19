package multisectionpatch

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type fileSnapshot struct {
	path     string
	info     os.FileInfo
	identity string
	links    uint64
	data     []byte
}

// readFileSnapshot resolves a regular text file and captures its identity,
// hard-link count, metadata, and validated bytes from the same open handle.
func readFileSnapshot(name string) (fileSnapshot, error) {
	absolute, err := filepath.Abs(name)
	if err != nil {
		return fileSnapshot{}, fmt.Errorf("%s: cannot resolve path: %w", name, err)
	}
	path, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return fileSnapshot{}, fmt.Errorf("%s: cannot resolve path: %w", name, err)
	}
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		return fileSnapshot{}, fmt.Errorf("%s: cannot stat: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return fileSnapshot{}, fmt.Errorf("%s: not a regular file", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return fileSnapshot{}, fmt.Errorf("%s: cannot open: %w", path, err)
	}
	info, err = file.Stat()
	if err != nil {
		_ = file.Close()
		return fileSnapshot{}, fmt.Errorf("%s: cannot stat: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return fileSnapshot{}, fmt.Errorf("%s: not a regular file", path)
	}

	// Identity and bytes come from one open handle so path aliases cannot race
	// separate stat and read operations during planning.
	identity, links, err := fileIdentityAndLinks(file, info)
	if err != nil {
		_ = file.Close()
		return fileSnapshot{}, fmt.Errorf("%s: cannot identify file: %w", path, err)
	}
	data, err := io.ReadAll(file)
	if err != nil {
		_ = file.Close()
		return fileSnapshot{}, fmt.Errorf("%s: cannot read: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fileSnapshot{}, fmt.Errorf("%s: cannot close after reading: %w", path, err)
	}
	if err := validateTextData(path, data); err != nil {
		return fileSnapshot{}, err
	}
	return fileSnapshot{
		path:     path,
		info:     info,
		identity: identity,
		links:    links,
		data:     data,
	}, nil
}

type stagedPlan struct {
	plan                *filePlan
	replacementPath     string
	replacementIdentity string
	recoveryPath        string
	preserveRecovery    bool
}

// applyPlans applies plans with optional backups through the guarded staging
// and rollback path when the caller does not need the backup location reported.
func applyPlans(plans []*filePlan, backup bool, replace func(string, string) error) error {
	return applyPlansWithBackupReport(plans, backup, replace, nil)
}

// applyPlansWithBackupReport stages every replacement and recovery file before
// replacing any target, revalidates targets, then replaces sequentially with
// guarded rollback.
func applyPlansWithBackupReport(
	plans []*filePlan,
	backup bool,
	replace func(string, string) error,
	reportBackup func(string),
) (resultErr error) {
	if len(plans) == 0 {
		return nil
	}

	staged := make([]*stagedPlan, 0, len(plans))
	defer func() {
		for _, item := range staged {
			if item.replacementPath != "" {
				resultErr = removeTemporary(resultErr, item.replacementPath)
			}
			if item.recoveryPath != "" && !item.preserveRecovery {
				resultErr = removeTemporary(resultErr, item.recoveryPath)
			}
		}
	}()

	// Stage every replacement and recovery file before touching any target.
	for _, plan := range plans {
		replacementPath, replacementIdentity, err := stageBytes(
			plan.path,
			"replacement",
			plan.updated,
			plan.info.Mode(),
		)
		if err != nil {
			return err
		}
		item := &stagedPlan{
			plan:                plan,
			replacementPath:     replacementPath,
			replacementIdentity: replacementIdentity,
		}
		staged = append(staged, item)

		recoveryPath, _, err := stageBytes(plan.path, "recovery", plan.original, plan.info.Mode())
		if err != nil {
			return err
		}
		item.recoveryPath = recoveryPath
	}

	// The batch preflight prevents a stale target from causing partial writes.
	if err := verifyPlansUnchanged(plans); err != nil {
		return err
	}
	if backup {
		root, err := backUpPlans(plans)
		if err != nil {
			return err
		}
		if reportBackup != nil {
			reportBackup(root)
		}
	}
	if err := verifyPlansUnchanged(plans); err != nil {
		return err
	}

	applied := 0
	for _, item := range staged {
		// Recheck immediately before each rename to narrow the concurrent-write
		// window after the batch preflight.
		if err := verifyPlanUnchanged(item.plan); err != nil {
			return rollbackResult(err, staged[:applied], replace)
		}
		if err := replace(item.replacementPath, item.plan.path); err != nil {
			cause := fmt.Errorf("%s: cannot replace: %w", item.plan.path, err)
			return rollbackResult(cause, staged[:applied+1], replace)
		}
		item.replacementPath = ""
		applied++
	}
	return nil
}

// stageBytes writes, sets permissions on, syncs, and identifies a single-link
// temporary file beside its target so replacement stays on one filesystem.
func stageBytes(
	target string,
	purpose string,
	data []byte,
	mode os.FileMode,
) (path string, identity string, err error) {
	file, err := os.CreateTemp(filepath.Dir(target), ".multi-section-patch-"+purpose+"-*")
	if err != nil {
		return "", "", fmt.Errorf("%s: cannot stage %s: %w", target, purpose, err)
	}
	path = file.Name()
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
		if err != nil {
			err = removeTemporary(err, path)
		}
	}()

	if err = file.Chmod(mode.Perm()); err != nil {
		return "", "", fmt.Errorf("%s: cannot set staged permissions: %w", target, err)
	}
	if _, err = file.Write(data); err != nil {
		return "", "", fmt.Errorf("%s: cannot stage %s: %w", target, purpose, err)
	}
	if err = file.Sync(); err != nil {
		return "", "", fmt.Errorf("%s: cannot sync staged %s: %w", target, purpose, err)
	}
	info, err := file.Stat()
	if err != nil {
		return "", "", fmt.Errorf("%s: cannot stat staged %s: %w", target, purpose, err)
	}
	identity, links, err := fileIdentityAndLinks(file, info)
	if err != nil {
		return "", "", fmt.Errorf("%s: cannot identify staged %s: %w", target, purpose, err)
	}
	if links != 1 {
		return "", "", fmt.Errorf("%s: staged %s unexpectedly has %d links", target, purpose, links)
	}
	closeErr := file.Close()
	closed = true
	if closeErr != nil {
		return "", "", fmt.Errorf("%s: cannot close staged %s: %w", target, purpose, closeErr)
	}
	return path, identity, nil
}

// removeTemporary preserves the primary failure while making every retained
// staging path visible to the caller.
func removeTemporary(result error, path string) error {
	removeErr := os.Remove(path)
	if removeErr == nil || os.IsNotExist(removeErr) {
		return result
	}
	if result == nil {
		return fmt.Errorf(
			"cleanup incomplete: temporary file retained at %s: %v",
			path,
			removeErr,
		)
	}
	return fmt.Errorf(
		"%w; cleanup incomplete: temporary file retained at %s: %v",
		result,
		path,
		removeErr,
	)
}

// verifyPlansUnchanged re-snapshots every planned target and stops at the first
// identity, link-count, permission, or content mismatch.
func verifyPlansUnchanged(plans []*filePlan) error {
	for _, plan := range plans {
		if err := verifyPlanUnchanged(plan); err != nil {
			return err
		}
	}
	return nil
}

// verifyPlanUnchanged rejects a target whose identity, hard-link state,
// permissions, or bytes no longer match its planning snapshot.
func verifyPlanUnchanged(plan *filePlan) error {
	snapshot, err := readFileSnapshot(plan.path)
	if err != nil {
		return fmt.Errorf("%s: changed since it was read: %w", plan.path, err)
	}
	if plan.identity != snapshot.identity ||
		snapshot.links > 1 ||
		plan.info.Mode().Perm() != snapshot.info.Mode().Perm() ||
		!bytes.Equal(plan.original, snapshot.data) {
		return fmt.Errorf("%s: changed since it was read", plan.path)
	}
	return nil
}

// rollbackResult visits applied plans in reverse order and restores originals
// only while the expected replacement remains untouched, retaining recovery
// files rather than overwriting concurrent changes.
func rollbackResult(cause error, applied []*stagedPlan, replace func(string, string) error) error {
	if len(applied) == 0 {
		return cause
	}

	var failures []string
	for index := len(applied) - 1; index >= 0; index-- {
		item := applied[index]
		current, err := readFileSnapshot(item.plan.path)
		if err != nil {
			item.preserveRecovery = true
			failures = append(failures, fmt.Sprintf(
				"%s cannot be checked after replacement: %v; recovery copy: %s",
				item.plan.path,
				err,
				item.recoveryPath,
			))
			continue
		}
		originalIsStillPresent := current.identity == item.plan.identity &&
			current.links == 1 &&
			current.info.Mode().Perm() == item.plan.info.Mode().Perm() &&
			bytes.Equal(current.data, item.plan.original)
		if originalIsStillPresent {
			continue
		}
		replacementIsStillPresent := current.identity == item.replacementIdentity &&
			current.links == 1 &&
			current.info.Mode().Perm() == item.plan.info.Mode().Perm() &&
			bytes.Equal(current.data, item.plan.updated)
		if !replacementIsStillPresent {
			// Never overwrite a concurrent change; retain the recovery copy so
			// the caller can restore it deliberately.
			item.preserveRecovery = true
			failures = append(failures, fmt.Sprintf(
				"%s changed after replacement; recovery copy: %s",
				item.plan.path,
				item.recoveryPath,
			))
			continue
		}
		if err := replace(item.recoveryPath, item.plan.path); err != nil {
			item.preserveRecovery = true
			failures = append(failures, fmt.Sprintf(
				"%s could not be restored: %v; recovery copy: %s",
				item.plan.path,
				err,
				item.recoveryPath,
			))
			continue
		}
		item.recoveryPath = ""
		restored, err := readFileSnapshot(item.plan.path)
		if err != nil ||
			restored.info.Mode().Perm() != item.plan.info.Mode().Perm() ||
			!bytes.Equal(restored.data, item.plan.original) {
			failures = append(failures, fmt.Sprintf(
				"%s restoration could not be verified",
				item.plan.path,
			))
		}
	}
	if len(failures) == 0 {
		return fmt.Errorf("%w; rolled back all changes", cause)
	}
	return fmt.Errorf(
		"%w; rollback incomplete: %s",
		cause,
		strings.Join(failures, "; "),
	)
}

// backUpPlans writes each original plus a checksum-and-mode manifest into a new
// timestamped directory and returns its absolute path. A failure may retain the
// partially populated directory for diagnosis and manual recovery.
func backUpPlans(plans []*filePlan) (string, error) {
	prefix := ".multi-section-patch-backup-" + time.Now().UTC().Format("20060102-150405-")
	root, err := os.MkdirTemp(".", prefix)
	if err != nil {
		return "", fmt.Errorf("cannot create backup directory: %w", err)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("cannot resolve backup directory: %w", err)
	}
	type manifestEntry struct {
		Source string `json:"source"`
		Backup string `json:"backup"`
		SHA256 string `json:"sha256"`
		Mode   string `json:"mode"`
	}
	manifest := struct {
		Version int             `json:"version"`
		Files   []manifestEntry `json:"files"`
	}{
		Version: 1,
		Files:   make([]manifestEntry, 0, len(plans)),
	}
	for _, plan := range plans {
		sum := sha256.Sum256([]byte(plan.path))
		name := hex.EncodeToString(sum[:6]) + "-" + filepath.Base(plan.path)
		target := filepath.Join(root, name)
		if err := os.WriteFile(target, plan.original, plan.info.Mode().Perm()); err != nil {
			return "", fmt.Errorf("%s: cannot create backup: %w", plan.path, err)
		}
		contentSum := sha256.Sum256(plan.original)
		manifest.Files = append(manifest.Files, manifestEntry{
			Source: plan.path,
			Backup: name,
			SHA256: hex.EncodeToString(contentSum[:]),
			Mode:   fmt.Sprintf("%04o", plan.info.Mode().Perm()),
		})
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", fmt.Errorf("cannot encode backup manifest for %s: %w", root, err)
	}
	manifestData = append(manifestData, '\n')
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), manifestData, 0o600); err != nil {
		return "", fmt.Errorf("cannot create backup manifest in %s: %w", root, err)
	}
	return root, nil
}
