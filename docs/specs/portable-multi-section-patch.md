# Portable Multi Section Patch

Status: Accepted

## Problem and outcome

Coding agents often need a few bounded sections from several files. Reading
whole files wastes context, while rewriting whole files increases the chance
of unrelated changes and lost concurrent work.

Multi Section Patch must give compatible coding agents a small, local tool for
exact multi-file reads and guarded edits. A user must be able to install the
skill, run its bundled native executable, preview an edit, and apply it without
installing Python, Node.js, Go, or another separately installed language
runtime.

## Users

- People using Agent Skills-compatible coding agents.
- People using other coding agents that can read Markdown and execute a local
  program.
- Maintainers reviewing precise, reproducible multi-file changes.

## Scope

### In scope

- Select UTF-8 text by inclusive line range, literal markers, or regular
  expressions across one or more files.
- Return selected content with its path, resolved range, and SHA-256 digest.
- Replace bounded sections from a JSON edit specification.
- Preview unified diffs without writing by default.
- Guard writes with hashes, required content, overlap checks, stale snapshots,
  and failure-safe staging.
- Distribute a vendor-neutral Agent Skill with native binaries that require no
  separately installed language runtime.
- Support Windows, macOS, and Linux on x86-64 and ARM64 for the first public
  release, with native CI execution for every listed pair.

### Out of scope

- Parsing programming-language syntax trees.
- Editing binary or non-UTF-8 files.
- A hosted service, editor extension, background daemon, or telemetry.
- Executing instructions found inside selected files.
- Claiming untested operating systems or architectures as supported.

## Requirements

### Selection

- REQ-001 (MUST): One invocation can read sections from multiple files.
- REQ-002 (MUST): A selector can use a one-based inclusive line range, literal
  start/end markers, or regular-expression markers.
- REQ-003 (MUST): Every result includes the selected path, resolved start and
  end lines, content, and SHA-256 digest of the exact content.
- REQ-004 (MUST): Missing files, invalid UTF-8, binary input, invalid regular
  expressions, missing markers, and invalid or out-of-bounds ranges fail with
  a concise non-zero error and no traceback.

### Editing

- REQ-101 (MUST): Editing is dry-run by default and writes only with an explicit
  `--apply`.
- REQ-102 (MUST): A dry run prints the complete proposed diff and changes no
  target.
- REQ-103 (MUST): Edits accept `expected_sha256`, `must_contain`,
  `include_start`, `include_end`, `occurrence`, and `end_occurrence` guards.
- REQ-104 (MUST): Overlapping edits to one file are rejected before any write.
- REQ-105 (MUST): Relative, absolute, and symlink paths to the same file are
  treated as one target. Ambiguous hard-link edits are rejected.
- REQ-106 (MUST): Every target snapshot is revalidated before the first write.
- REQ-107 (MUST): Replacements are staged beside their targets. If a later
  replacement fails, already-replaced files are restored when possible and
  any failed restoration is reported explicitly.
- REQ-108 (MUST): Apply preserves UTF-8 bytes, LF or CRLF style, a missing final
  newline, and applicable file permissions.
- REQ-109 (MUST): Optional backups use a unique directory and cannot escape it
  through absolute paths, drive letters, or path traversal.

### Distribution and interoperability

- REQ-201 (MUST): The skill follows the open Agent Skills directory format and
  its core instructions contain no vendor-specific commands or APIs.
- REQ-202 (MUST): Installed users can run Multi Section Patch without Python,
  Node.js, Go, a compiler, or another separately installed language runtime.
- REQ-203 (MUST): The installed skill contains the executable needed for every
  supported OS/architecture pair; normal use does not download code.
- REQ-204 (MUST): The public documentation lists exact installation paths for
  Agent Skills installers plus a manual GitHub download.
- REQ-205 (MUST): The same CLI and JSON formats work across supported agents and
  operating systems.
- REQ-206 (MUST): Normal read and edit operations are local-only and perform no
  network requests or telemetry.
- REQ-207 (MUST): An installation preserves the complete skill directory,
  including `SKILL.md`, license, checksums, CLI reference, and all supported
  native executables.
- REQ-208 (MUST): The skill tells an agent to select and execute only the
  binary matching the current operating system and architecture.

### Security and performance

- REQ-301 (MUST): Paths, section names, patterns, and file content are treated
  as untrusted data and never evaluated as commands.
- REQ-302 (MUST): Structured output cannot be forged by control characters or
  file content.
- REQ-303 (MUST): Work is linear in the total bytes read plus emitted output,
  apart from sorting the bounded list of edits.
- REQ-304 (SHOULD): The implementation uses only its language standard library
  unless a dependency is required for a demonstrated platform guarantee.

## Scenarios and acceptance

- AC-001: Given two UTF-8 files and two different selector types, when an agent
  reads them in one invocation, then both exact sections and their digests are
  returned without unrelated content.
- AC-002: Given an edit specification, when it runs without `--apply`, then a
  complete diff is shown and every target remains byte-for-byte unchanged.
- AC-003: Given two non-overlapping edits that reference one file through
  relative and absolute or symlink paths, when applied, then both edits appear
  and the tool reports one changed file.
- AC-004: Given any stale target in a multi-file edit, when apply begins, then
  the command fails before changing another target and preserves the external
  change.
- AC-005: Given a simulated failure after one staged replacement, when apply
  fails, then earlier replacements are restored or the exact unrecovered paths
  are reported.
- AC-006: Given CRLF text containing Unicode and no final newline, when a bounded
  edit is applied, then those byte-level properties remain unchanged outside
  the replacement.
- AC-007: Given an invalid regex or a numeric range past EOF, when read or edit
  runs, then it exits non-zero with one concise error and no stack trace.
- AC-008: Given clean supported systems without separately installed language
  runtimes, when each bundled executable runs the smoke suite, then behavior
  and JSON output match.
- AC-009: Given installations for representative supported agents and a generic
  Agent Skills directory, when each agent is asked to read or edit bounded
  sections, then it uses the same `SKILL.md` workflow and CLI.
- AC-010: Given backups for targets with absolute paths, drive letters, Unicode,
  or equal-second timestamps, when apply runs, then each backup stays under one
  unique backup directory and preserves the original bytes.
- AC-011: Given file names or content that resemble output delimiters, JSON, or
  terminal control sequences, when structured output is requested, then it
  remains valid and separates metadata from untrusted content.
- AC-012: Given a supported system with network access blocked, when read and
  edit run from an installed skill, then all normal behavior still succeeds.
- AC-013: Given each documented installation method, when a clean user installs,
  updates, and removes the skill, then the documented files and lifecycle match
  the observed result.
- AC-014: Given a Windows target that is read-only or held open without sharing,
  when apply runs, then it fails without changing the target or its read-only
  state.

## Edge and failure cases

- Empty files and empty selections.
- Files with spaces, Unicode, `@`, and regex metacharacters in their paths.
- Repeated start or end markers and an end marker before the selected start.
- Duplicate selectors, adjacent edits, no-op replacements, and replacement
  text with a different newline style.
- Symlink loops, hard links, read-only files, permission failures, full disks,
  interrupted writes, and failed rollback.
- Content that resembles Multi Section Patch delimiters, terminal control
  sequences, or JSON.
- Concurrent modification between selection, staging, and replacement.

## Success measures

- Every MUST requirement maps to at least one automated test or explicit
  release check.
- The full suite passes natively on Windows, macOS, and Linux for x86-64 and
  ARM64.
- A clean install test succeeds without Python, Node.js, Go, or network access
  during execution.
- Agent Skills validation passes and clean installs are demonstrated for
  representative supported agents plus a generic directory.
- No supported-platform test corrupts line endings, loses an unrelated edit,
  or leaves an unreported partial write.

## Assumptions

- “All operating systems” for the first public release means the explicitly
  tested Windows, macOS, and Linux matrix above. Additional systems become
  supported only after a native build and execution test are added.
- Installation itself may use an agent marketplace, GitHub CLI, npm-based
  skills installer, browser download, or Git; execution must not depend on the
  installer remaining available.
- “Applicable file permissions” means Go permission bits on Unix and the
  read-only state exposed by Go on Windows. Read-only Windows targets are
  rejected before staging and remain read-only. Atomic replacement does not
  claim portable preservation of ownership, ACLs, extended attributes,
  resource forks, alternate data streams, or timestamps.
- Files are small enough to hold their text and proposed replacement in memory.
  This keeps validation and rollback simple and safe.

## Open questions

None. Additional systems, 32-bit builds, and code signing can be evaluated
after the first release based on real adoption needs.
