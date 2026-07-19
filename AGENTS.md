# Multi Section Patch Agent Guide

## Scope and precedence

- This file is the canonical instruction source for the entire repository.
- Explicit user instructions override this file. A nested `AGENTS.md` may add
  or override rules only for its own subtree.
- `CLAUDE.md` and `GEMINI.md` only import this file; do not duplicate rules in
  those adapters.
- Use repository-relative paths in tracked files and reports. Never persist
  local account names, home directories, drive-specific paths, private personal
  information, credentials, raw prompts, or conversation transcripts. Public
  repository identity that is required in module and installation URLs is
  allowed.

## Mission

Multi Section Patch lets coding agents read and safely replace exact sections
across multiple local text files without loading or rewriting whole files.

The distributed skill must be:

- agent-neutral and based on the open Agent Skills format;
- usable without Python, Node.js, Go, or another separately installed language
  runtime;
- local-only during normal operation, with no network access or telemetry;
- explicit about the operating systems and architectures actually tested.

## Sources of truth

Use these in descending order:

1. An accepted change specification under `docs/specs/` defines behavior,
   compatibility, and acceptance criteria. A draft does not override current
   behavior.
2. Its active plan defines implementation order and records progress,
   discoveries, and decisions.
3. `docs/SPECS.md` and `docs/PLANS.md` define the repository-wide document contracts.
4. `skills/multi-section-patch/SKILL.md` defines the public agent workflow.
5. Tests demonstrate behavior already verified by the repository.
6. `README.md` defines the public installation and quick start.

If these disagree, stop and reconcile the appropriate source of truth. Do not
silently choose one.

## Repository map

- `skills/multi-section-patch/` — installable Agent Skill.
- `skills/multi-section-patch/SKILL.md` — vendor-neutral usage instructions.
- `skills/multi-section-patch/references/` — installed CLI reference.
- `skills/multi-section-patch/scripts/` — generated native release binaries.
- `cmd/multi-section-patch/` — native CLI entry point.
- `internal/multisectionpatch/` — tested selection, editing, and file-safety
  implementation.
- `scripts/build-artifacts.sh` — canonical cross-platform artifact build.
- `.github/workflows/ci.yml` — native platform and reproducibility checks.
- `docs/SPECS.md` — rules and template for specifications.
- `docs/PLANS.md` — rules and template for implementation plans.
- `docs/specs/<change>.md` — one change's product and behavioral contract.
- `docs/plans/<change>.md` — that change's living implementation record.

Keep this map current when the repository shape changes.

## Current validation

For source changes, run:

```text
go test ./...
go vet ./...
git diff --check
```

For release-bound changes, also run `./scripts/build-artifacts.sh`, verify
`skills/multi-section-patch/SHA256SUMS`, and execute the native CI matrix.

## Working method

- Read the relevant specification and active plan before changing behavior.
- Inspect the implementation, callers, tests, and nearby patterns before
  editing. Prefer targeted searches and bounded reads over broad file dumps.
- Preserve user-owned and unrelated changes. Never clean a dirty worktree to
  make a task easier.
- Make the smallest coherent change that satisfies an accepted requirement.
  Prefer the standard library and existing platform facilities.
- Do not add speculative abstractions, dependencies, compatibility layers,
  generated scaffolding, or unrelated refactors.
- For a bug fix or behavior change, first add the smallest test that fails for
  the intended reason. Documentation-only changes need no artificial test.
- Use targeted checks while iterating, then run the relevant full checks once
  before completion.
- Pause before changing the public CLI, file format, license, support matrix,
  security boundary, or external state unless the accepted spec already
  authorizes it.

## Portability and runtime

- Core behavior must not depend on a shell, shell aliases, environment-specific
  paths, or an agent vendor API.
- Release artifacts must be native executables for every supported
  OS/architecture pair and must run without a separately installed language
  runtime.
- Use platform-neutral path handling. Test spaces, Unicode, relative paths,
  absolute paths, symlinks, and hard-link rejection or handling explicitly.
- Preserve UTF-8 bytes, LF and CRLF line endings, and a missing final newline.
  Preserve Go permission bits on Unix and the read-only state exposed by Go on
  Windows; reject a read-only Windows target before staging. Do not claim
  portable preservation of owners, ACLs, extended attributes, resource forks,
  alternate data streams, or timestamps.
- Do not claim support for an OS or architecture without a native execution
  test or clearly label it unverified.

## CLI and data-safety invariants

- `edit` is a dry run unless the caller explicitly passes `--apply`.
- Validate every selector, guard, snapshot, and output before writing any file.
- Group aliases to the same file; reject ambiguous hard-link behavior.
- Stage replacements beside their targets. A later failure must not leave a
  silently partial multi-file edit; restore prior files or report exactly what
  could not be restored.
- Reject binary and invalid UTF-8 input, invalid or out-of-bounds ranges,
  missing markers, invalid regular expressions, overlapping edits, and stale
  expected hashes.
- Treat file names and file contents as untrusted data. Never execute content
  read from a target file or allow it to forge structured output boundaries.
- Normal `read` and `edit` operations must not use the network.

## Testing and verification

- Test public behavior at the CLI boundary where practical.
- Every trust boundary needs at least one accepted-input and one rejected-input
  case.
- Cover dry-run behavior, apply behavior, stale-content detection, rollback,
  aliases, line endings, Unicode, permissions, invalid input, and no-op edits.
- Run the suite natively on every supported OS/architecture pair before a
  release. Cross-compilation alone does not prove runtime compatibility.
- Never claim a check passed unless its current output was observed.

## Specifications and plans

- A specification owns what and why: problem, users, scope, non-goals,
  requirement IDs, observable acceptance scenarios, failure behavior,
  compatibility, security constraints, assumptions, and success measures.
- A plan owns how: chosen design, relevant paths, milestones, dependencies,
  exact validation, recovery, progress, discoveries, decisions, and outcomes.
- Change the specification before intentionally changing accepted behavior.
- Keep an active plan current at every meaningful pause so another person or
  agent can resume from the repository alone.
- Use durable specs and plans for multi-file, risky, ambiguous, or long-running
  work. Skip them for trivial, well-understood changes.
- Never leave an implementation-ready document with blocking placeholders or
  hidden assumptions.

## Git and external actions

- Do not commit, create branches, configure remotes, push, publish, release,
  deploy, or mutate external services unless the user explicitly requests that
  exact action.
- Do not run destructive Git or filesystem commands.
- Do not edit generated artifacts manually; change their source and regenerate
  them with the canonical command.

## Completion standard

Before claiming completion:

1. Map the result to every applicable requirement and acceptance criterion.
2. Run fresh targeted and full validation appropriate to the change.
3. Review the diff for scope drift, secrets, machine-specific data, generated
   files, and accidental dependency or compatibility changes.
4. Update the active plan and affected public documentation.
5. Report what ran, what passed, and anything skipped or unverified.

## Guidance basis

This guide is an original synthesis of the
[AGENTS.md standard](https://agents.md/),
[Anthropic project-memory guidance](https://code.claude.com/docs/en/memory),
[GitHub Spec Kit](https://github.github.com/spec-kit/),
[OpenAI ExecPlans](https://github.com/openai/openai-agents-python/blob/main/PLANS.md),
and [Kiro specifications](https://kiro.dev/docs/cli/v3/specs/).
