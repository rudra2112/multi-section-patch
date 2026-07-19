// Package multisectionpatch implements the local, vendor-neutral Multi Section
// Patch CLI.
//
// Reads resolve every requested section before emitting output. Edits are
// dry-run by default; applied edits validate all snapshots, stage replacement
// and recovery files beside each target, and roll back completed replacements
// after a later failure when doing so cannot overwrite a concurrent change.
package multisectionpatch
