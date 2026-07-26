# Changelog

Notable user-visible changes are recorded here. Versions follow
[Semantic Versioning](https://semver.org/).

`v0.2.0` remains listed even though it was withdrawn so the published tag and
release history are represented accurately.

## [Unreleased]

### Changed

- Updated installation guidance to require GitHub CLI 2.94.0 or newer.

## [0.2.2] - 2026-07-26

### Added

- Added tag-only GitHub artifact attestations for all six native executables
  and `SHA256SUMS`.
- Added source and maintainer metadata, a two-file demonstration, and a
  reproducible byte-reduction measurement.

### Changed

- Made installation, discovery, and removal guidance agent-neutral.
- Kept the CLI, file formats, supported platforms, native executables, and
  checksums unchanged from `v0.2.1`.

## [0.2.1] - 2026-07-26

### Fixed

- Corrected the Windows exclusive-open target test to exercise reviewed apply
  instead of a dry run.
- Established the first supported release of the complete `v0.2` feature set
  after all required CI passed.

## [0.2.0] - 2026-07-26 — Withdrawn

### Added

- Added `plan_sha256` to dry-run output and required the matching
  `--expect-plan` value before applying edits.
- Added structured JSON error context for commands, items, files, and fields.
- Expanded regression coverage for reviewed-plan application and transaction
  safety.

### Changed

- Added request-local file snapshot caching so selectors and edits use a
  consistent view of each file.
- Improved read, edit, and transaction handling along with the public CLI
  documentation.

### Status

- Withdrawn because required Windows CI failed: the exclusive-open regression
  test exercised a dry run instead of reviewed apply.
- Do not install this version; use `v0.2.1` or later.

## [0.1.1] - 2026-07-19

### Fixed

- Fixed regular-expression selectors ending in `$` for newline-terminated LF
  and CRLF files in both `read` and `edit`.

### Changed

- Added concise Go documentation comments for production functions.

## [0.1.0] - 2026-07-19

### Added

- Published the first vendor-neutral Multi Section Patch Agent Skill.
- Added bounded multi-file reads using whole-file, line-range, marker, heading,
  and RE2 regular-expression selectors.
- Added guarded multi-file edit previews with explicit application,
  content-digest checks, staging, and rollback.
- Added self-contained native executables for Windows, macOS, and Linux on
  x86-64 and ARM64.
- Added local-only operation without telemetry or a separately installed
  language runtime.

[Unreleased]: https://github.com/rudra2112/multi-section-patch/compare/v0.2.2...HEAD
[0.2.2]: https://github.com/rudra2112/multi-section-patch/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/rudra2112/multi-section-patch/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/rudra2112/multi-section-patch/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/rudra2112/multi-section-patch/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/rudra2112/multi-section-patch/releases/tag/v0.1.0
