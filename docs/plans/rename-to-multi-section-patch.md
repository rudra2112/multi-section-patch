# Rename to Multi Section Patch

Status: Completed

Related specification: `docs/specs/portable-multi-section-patch.md`

This is a living plan. Keep Progress, Discoveries, Decisions, and Outcome
current whenever work pauses or reality changes.

## Purpose

Adopt Multi Section Patch as the single identity of the unpublished project and
installable skill while keeping it portable across coding agents and operating
systems. The installed skill must not require Python, Go, Node.js, or another
separately installed language runtime.

## Context

The project has no commits, remote, tags, or published releases, so the rename
can be complete without a compatibility layer. The repository already contains
a Go CLI, an Agent Skills-compatible package, six native executables, durable
specifications and plans, and a seven-job native/reproducibility workflow.

Constraints:

- Make all changes locally. Do not commit, push, tag, publish, or create a
  GitHub release.
- Use `rudra2112/multi-section-patch` as the future repository identifier.
- Keep real execution coverage for Linux, macOS, and Windows on AMD64 and ARM64.
- Keep documentation and metadata agent-neutral.
- Do not retain unpublished legacy names or command aliases.

Canonical identifiers:

- Product name: `Multi Section Patch`
- Repository and skill name: `multi-section-patch`
- Command package: `cmd/multi-section-patch`
- Implementation package: `internal/multisectionpatch`
- Binary prefix: `multi-section-patch-`
- Installable skill: `skills/multi-section-patch`
- Portable design documents: `portable-multi-section-patch.md`

Structured output markers, temporary-file names, environment variables, Go
package identifiers, test fixtures, and documentation examples use the same
canonical identity.

## Approach

The repository will contain one Agent Skills-compatible directory. An installer
copies or links that complete directory into an agent's skill discovery path.
The skill metadata tells the agent when to use it; `SKILL.md` tells the agent how
to choose and invoke the bundled binary for the current operating system and
architecture.

At execution time, the agent runs one of six precompiled binaries:

- Linux AMD64 or ARM64
- macOS AMD64 or ARM64
- Windows AMD64 or ARM64

No Python, Go, Node.js, package manager, network request, or background service
is required after installation. Node.js is needed only when the user chooses the
optional `npx skills` installer; GitHub CLI and manual-copy installation paths
remain available.

Keep the six native GitHub-hosted runner jobs because each supported binary is
executed on its real platform. These are standard runners and therefore have
free, unlimited Actions minutes for public repositories under GitHub's current
policy. Remove the temporary workflow artifact upload because it is not part of
installation or release delivery and consumes the account's artifact-storage
quota. Private repositories and larger runners have different billing rules.

## Milestones

### M1 — One canonical identity

Rename directories, documentation files, module paths, package names, commands,
binaries, markers, metadata, examples, tests, and CI identifiers. Remove the
unpublished executable aliases.

Proof: the Go suite passes and a repository-wide scan finds no former branding.

### M2 — Public adoption is explicit

Rewrite the README, skill instructions, CLI reference, security guidance, and
release checklist. Explain installation, progressive skill loading, native
binary selection, dry-run/apply use, fork module paths, unsigned-binary limits,
and CI billing.

Proof: Agent Skills validation and local installer discovery each find exactly
one skill named `multi-section-patch`.

### M3 — Generated outputs and repository path match

Rebuild the six binaries, regenerate `SHA256SUMS`, verify deterministic output,
run the complete local validation suite, remove generated prototype leftovers,
and rename the local repository directory.

Proof: checksums, executable metadata, smoke tests, stale-name scans, and the
post-rename acceptance check all pass.

## Validation and acceptance

Run from the repository root:

```text
go test ./...
go vet ./...
./scripts/build-artifacts.sh
gh skill publish --dry-run .
npx --yes skills@1.5.19 add ./skills/multi-section-patch --list
```

Then verify `SHA256SUMS`, native executable metadata, size budgets, a read and
dry-run/apply smoke test, a clean repository-wide branding scan, and the local
directory name.

Acceptance criteria:

- The canonical command and skill name are `multi-section-patch`.
- No project-owned path or text retains the former product name or acronym.
- A supported agent can discover the skill and execute the bundled native
  binary without a separately installed language runtime.
- Read operations remain non-mutating.
- Edit operations still default to a reviewed dry run and require an explicit
  apply action.
- All six committed binaries match their recorded checksums.
- The full local validation suite passes.
- Git state remains uncommitted and unpushed.

## Safety and recovery

Path renames preserve existing files, and generated binaries are replaced only
by the canonical build script. If validation fails, keep the repository
unpublished, correct the named source or documentation, regenerate all
artifacts, and rerun the affected check followed by the complete suite. No step
requires changing GitHub or Git history.

## Progress

- [x] Record a failing pre-rename path check.
- [x] Rename source, skill, documentation, and public identifiers.
- [x] Update installation, execution, portability, and CI-cost documentation.
- [x] Rebuild and verify all six native executables.
- [x] Complete final validation and stale-name cleanup.
- [x] Rename the local repository directory.

## Discoveries

- Standard GitHub-hosted runners are free and unlimited for public repositories;
  the removed workflow upload was the only recurring artifact-storage use.
- Agent installers preserve the whole skill directory, discover metadata first,
  load `SKILL.md` on a matching task, and run one platform binary.
- GitHub CLI skill commands are a preview and differ by version, so public
  instructions defer to the installed command's help.
- The native binaries need no separately installed language runtime, but the
  initial macOS and Windows artifacts are not platform-signed.

## Decisions

- Decision: use the full name in public commands, paths, markers, and internal
  package identifiers.
  Rationale: the project is unpublished, so complete consistency is preferable
  to a legacy alias layer.
  Date: 2026-07-19.
- Decision: retain all six native CI jobs and remove only the artifact upload.
  Rationale: native execution is release evidence; the upload adds storage use
  but no installation or release capability.
  Date: 2026-07-19.
- Decision: make no commit, push, repository, tag, or release.
  Rationale: the user explicitly prohibited commits and pushes and has not
  authorized any publication mutation.
  Date: 2026-07-19.

## Outcome

The project, repository directory, skill, command, Go module, package,
documentation, CI identifiers, structured output, temporary paths, and all six
generated binaries now use the Multi Section Patch identity. Source tests, race
tests, vet, benchmarks, checksum verification, byte-for-byte rebuilds, workflow
parsing, Agent Skills validation, local installer discovery/copy, and a
no-runtime native smoke test passed.

The native matrix retains all six standard public-repository runners while
disabling caches and artifact uploads. No repository, remote, commit, push, tag,
release, authentication change, or other GitHub mutation was created.
