# Ship a self-contained, cross-agent Multi Section Patch skill

Status: Active

Related specification: `docs/specs/portable-multi-section-patch.md`

This is a living plan. Keep Progress, Discoveries, Decisions, and Outcome
current whenever work pauses or reality changes.

## Purpose

Replace the transitional Python-only package with a safe native CLI and a
vendor-neutral Agent Skill. A person should be able to install Multi Section
Patch into a supported coding agent, run it on Windows, macOS, or Linux without
a separately installed language runtime, preview an exact multi-file edit, and
apply it without losing unrelated or concurrent work.

## Context

The audited starting point bundled a Python program and referenced private PATH
aliases that clean skill installers did not create. The transitional
implementation and Python tests have since been removed after their behavior
and failure cases were captured in the native suite.

The original regression suite demonstrated four working behaviors and exposed
five launch-blocking failures:

- line ranges past EOF are silently accepted;
- invalid regular expressions emit a traceback;
- relative and absolute aliases can lose one edit;
- stale content can be overwritten;
- a later multi-file write failure is not rolled back.

Text-mode writes can also corrupt CRLF on Windows, backup paths are not
portable, and output delimiters can be forged by untrusted content. The current
Go implementation addresses those cases; the remaining release gates are
generated artifacts, native CI on all six platform pairs, and clean public
installation tests.

## Approach

Implement the CLI in Go using the standard library. Go provides one small
source implementation, native cross-compilation, and standalone executables
without requiring users to install Go.

Preserve the useful public concepts—`read`, `edit`, selectors, JSON specs,
dry-run diffs, hashes, and guards—while fixing validation and file replacement
at shared boundaries. Keep the CLI independent from every agent vendor.

Bundle stripped native executables inside the installed skill for the supported
OS/architecture matrix. Release archives may contain the same binaries, but
normal skill execution must not download them.

Rejected alternatives:

- Keeping Python requires a runtime the user explicitly rejected.
- Freezing Python embeds a larger interpreter and complicates deterministic
  cross-platform builds without improving the small CLI.
- A first-run downloader keeps the repository smaller but adds network and
  supply-chain failure to normal use.

## Milestones

### M1 — Establish the portable contract

Changes:

- Add canonical cross-agent guidance in `AGENTS.md`.
- Add import-only `CLAUDE.md` and `GEMINI.md` adapters.
- Add the repository-wide `docs/SPECS.md` and `docs/PLANS.md` contracts.
- Add the behavior specification and this living plan.

Proof:

- All paths are repository-relative.
- `AGENTS.md` remains concise and contains no vendor-only workflow.
- A search finds no local account names, home-directory paths, credentials, or
  machine configuration in the new documents. The intended public repository
  owner may appear where a canonical module or installation URL requires it.

### M2 — Implement the native CLI

Changes:

- Add a minimal Go module and one testable CLI implementation.
- Port read selectors, marker and regex resolution, JSON specs, output hashes,
  edit guards, dry-run diffs, and exit behavior.
- Replace Python regression tests with CLI-boundary Go tests that preserve the
  established contract.

Proof:

- `go test ./...` passes.
- A native build can read multiple sections and apply one guarded edit.
- Public users need no Python or Go after the executable is built.

### M3 — Make writes failure-safe

Changes:

- Resolve and identify target files once.
- Reject out-of-range selectors, invalid regexes, overlaps, ambiguous hard
  links, binary data, invalid UTF-8, and stale snapshots before writing.
- Stage byte-exact replacements beside targets, preserve applicable permission
  bits, and restore earlier targets after a later failure.
- Make backups unique and path-safe on every supported OS.

Proof:

- Targeted tests cover AC-002 through AC-007, AC-010, and AC-014.
- Failure injection demonstrates no silent partial multi-file result.
- CRLF, Unicode, permissions, and missing-final-newline checks pass.

### M4 — Package every supported platform

Changes:

- Build native executables for Windows, macOS, and Linux on x86-64 and ARM64.
- Place the executables under deterministic platform-specific filenames inside
  the skill.
- Update `SKILL.md` to select the bundled executable instead of assuming PATH
  aliases or an installed interpreter.
- Add checksums and native CI jobs for each supported pair.

Proof:

- Each native runner executes the same smoke and regression suite.
- Cross-compilation and checksum generation are reproducible from documented
  commands.
- A clean installed skill runs with Python, Node.js, and Go removed from PATH.

### M5 — Prepare public adoption

Changes:

- Add a concise README, permissive license, security notes, contribution
  commands, and installation/update/removal examples.
- Validate the skill with current Agent Skills tooling.
- Test clean installations for representative supported agents and a generic
  Agent Skills target.
- Prepare the exact GitHub repository, release, topic, and announcement
  commands for the user to run.

Proof:

- All MUST requirements have evidence.
- The repository is ready for public publication without machine-specific
  state.
- No commit, push, repository creation, tag, release, or external mutation is
  performed unless the user separately authorizes it.

## Validation and acceptance

Run commands from the repository root.

```text
go test ./...
go vet ./...
./scripts/build-artifacts.sh
```

Before a public release:

- Run the full suite on every supported native CI runner.
- Run the built executable against AC-001 through AC-013.
- Validate `skills/multi-section-patch/SKILL.md` with both the Agent Skills
  validator and each documented installer.
- Compare the implementation and README against every MUST requirement.
- Review the complete diff for generated binaries, secrets, machine paths,
  dependency drift, and accidental external metadata.

Acceptance coverage:

- REQ-001 through REQ-004: AC-001 and AC-007.
- REQ-101 through REQ-109: AC-002 through AC-007 and AC-010.
- Windows permission and sharing behavior: AC-014.
- REQ-201 through REQ-208: AC-008, AC-009, AC-012, and AC-013.
- REQ-301 and REQ-302: AC-011 plus adversarial path/content tests.
- REQ-303: code review plus size-doubling, edit-count, and diff-hunk benchmarks.
- REQ-304: dependency manifest review.

## Safety and recovery

- Keep dry-run as the default throughout the migration.
- Preserve the transitional implementation until the native CLI reaches
  behavioral parity; remove it only after the native tests pass.
- Generate binaries from source rather than editing them.
- Stage file replacements in the target directory and retain rollback material
  until every replacement succeeds.
- If a rollback cannot complete, preserve recovery files and report their
  repository-independent paths clearly.
- Re-running build, validation, or packaging steps must not change source files
  unexpectedly.

## Progress

- [x] (2026-07-19) Audited the current skill layout and publishing tools.
- [x] (2026-07-19) Reproduced five safety and correctness failures.
- [x] (2026-07-19) Researched AGENTS.md, Agent Skills, cross-agent installers,
  OpenAI ExecPlans, GitHub Spec Kit, Anthropic guidance, Gemini guidance, and
  Kiro specifications.
- [x] (2026-07-19) Chose one canonical `AGENTS.md` with import-only compatibility
  adapters.
- [x] (2026-07-19) Completed and self-reviewed the M1 documentation.
- [x] (2026-07-19) Accepted Windows, macOS, and Linux on x86-64 and ARM64 as
  the first release's native support matrix.
- [x] (2026-07-19) Implemented M2 native read/edit behavior and verified it
  with the full suite, race detector, vet, native smoke tests, and
  size-doubling benchmarks.
- [x] (2026-07-19) Implemented M3 staging, stale-snapshot detection, alias and
  hard-link handling, rollback, retained-recovery reporting, permission and
  newline preservation, Windows read-only preflight, contained backups, and
  cleanup-failure reporting.
- [ ] (2026-07-19) M4 is partially complete: all six deterministic binaries and
  checksums are bundled, reproduce byte-for-byte, fit the release budget, have
  the expected executable formats, and their test suites cross-compile. The
  macOS ARM64 binary passed local runtime- and network-isolated smoke tests;
  the six hosted native CI jobs remain release gates.
- [ ] (2026-07-19) M5 is partially complete: public documentation, license,
  security policy, contribution guide, release checklist, Agent Skills
  validation, GitHub CLI local installation, and the six-target local npm
  install/update/removal lifecycle pass. Public remote installs and a tagged
  archive remain gated on repository publication and native CI.
- [x] (2026-07-19) Re-audited the candidate: organized durable documents under
  `docs/`, pinned artifact builds to Go 1.26.1, corrected GitHub CLI lifecycle
  guidance, made output failures nonzero, rejected non-regular text inputs, and
  changed dry-run output to focused unified-diff hunks.

## Discoveries

- Agent Skills installers copy bundled files but do not synthesize private PATH
  aliases, so the skill must invoke its selected executable directly.
- Python text-mode writes can duplicate carriage returns on Windows.
- Canonical path grouping fixes symlink and relative-path aliases; hard links
  need explicit rejection or file-identity handling.
- Cross-file atomic replacement is not portable. The honest guarantee is full
  preflight, per-file staging, and explicit rollback with recovery reporting.
- Go can portably preserve Unix permission bits and Windows read-only state,
  but not ownership, ACLs, extended attributes, resource forks, alternate data
  streams, or timestamps during staged replacement.
- GitHub CLI skill installation currently drops Unix executable bits, so public
  instructions include a one-time `chmod` fallback; the npm-based installer
  preserves them.
- Claude Code reads `CLAUDE.md`, not `AGENTS.md`; Gemini CLI defaults to
  `GEMINI.md`. Both support importing the canonical guide without duplication.
- A successful file replacement can still leave a temporary recovery file when
  cleanup is denied. Cleanup errors now retain the primary result and report
  every exact leftover path.
- The stripped six-binary set is about 16.2 MB and stays below the v0.1 limits
  of 3 MiB per executable and 18 MiB total.
- The size-doubling benchmarks scale proportionally for reading and edit
  construction; edit sorting and distant diff-hunk rendering remain linear in
  the bounded plan and emitted output.
- A `go` directive sets language compatibility, not the release compiler; the
  build script and CI must select the same exact Go toolchain.
- GitHub CLI's skill commands are a preview and differ between releases, so the
  public guide points users to the installed command's help and retains
  exact-directory deletion as the removal fallback.

## Decisions

- Decision: Use `AGENTS.md` as the single canonical project guide.
  Rationale: It is the Linux Foundation-backed cross-agent standard; thin
  imports cover tools that use another default filename.
  Date: 2026-07-19.
- Decision: Keep specifications about what/why and plans about how/progress.
  Rationale: This combines the strongest parts of GitHub Spec Kit and OpenAI
  ExecPlans without their vendor-specific scaffolding.
  Date: 2026-07-19.
- Decision: Use Go and bundled native executables.
  Rationale: Users explicitly require execution without a separately installed
  language runtime, and a small standard-library CLI does not justify a
  heavier stack.
  Date: 2026-07-19.
- Decision: Test Windows, macOS, and Linux on x86-64 and ARM64 for v0.1.
  Rationale: These are the release-blocking coding-agent environments; support
  claims for additional systems require native execution evidence.
  Date: 2026-07-19.
- Decision: Do not commit or push.
  Rationale: The user explicitly prohibited both actions.
  Date: 2026-07-19.
- Decision: Use `github.com/rudra2112/multi-section-patch` as the Go module.
  Rationale: It matches the user's intended personal GitHub account and the
  eventual canonical public repository URL.
  Date: 2026-07-19.
- Decision: Bundle the full MIT notice and a focused CLI reference inside the
  installable skill.
  Rationale: Installers copy the skill directory, so users need the license and
  exact contract without relying on repository-root files.
  Date: 2026-07-19.
- Decision: Pin release artifacts to Go 1.26.1 while retaining Go 1.23 as the
  source language floor.
  Rationale: Exact compiler selection makes local and CI binaries reproducible
  without raising the language features required by the source.
  Date: 2026-07-19.

## Outcome

The local repository is a complete publication candidate: it contains the
agent-neutral documentation contract, structured native source, adversarial
tests, build and native-CI automation, bundled license and CLI reference, and
six reproducible, self-contained binaries with verified checksums.

The plan remains Active. No repository, commit, push, tag, or release has been
created. Before v0.1 can be described as supported, the owner must publish the
repository, observe all six hosted native jobs and the reproducibility check,
and run the documented remote installation lifecycle against the tagged
release.
