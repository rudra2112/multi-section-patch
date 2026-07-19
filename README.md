# Multi Section Patch

[![CI](https://github.com/rudra2112/multi-section-patch/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/rudra2112/multi-section-patch/actions/workflows/ci.yml)

Multi Section Patch is a vendor-neutral
[Agent Skill](https://agentskills.io/) for reading and safely replacing exact
sections across multiple local text files.

Instead of loading or rewriting entire files, a coding agent can request only
the line ranges, headings, literal markers, or regular-expression bounds that
matter. Editing defaults to a complete diff, and writing requires an explicit
`--apply`.

The installed skill is self-contained. It bundles native executables for
Windows, macOS, and Linux, so normal use requires no Python, Node.js, Go,
compiler, package manager, network connection, or background service.

| At a glance | |
| --- | --- |
| Best for | Precise reads and coordinated, bounded edits across files |
| Input | Regular UTF-8 text files |
| Safe default | Dry run; writes require explicit `--apply` |
| Platforms | Windows, macOS, and Linux on x86-64 and ARM64 |
| Runtime dependencies | None beyond the bundled native executable |
| Network and telemetry | None during normal `read` or `edit` use |
| License | [MIT](LICENSE) |

## Quick start

You do not need a marketplace. Agent Skills-compatible coding agents discover
skills from local directories containing a `SKILL.md` file. An installer only
puts this repository's skill directory in the location expected by the chosen
agent.

### Install with GitHub CLI

This is the simplest user-wide installation when
[GitHub CLI](https://cli.github.com/) 2.91.0 or newer is available. The
`gh skill` commands are currently a public preview.

Preview the files before trusting them:

```text
gh skill preview rudra2112/multi-section-patch multi-section-patch
```

Install for Codex:

```text
gh skill install rudra2112/multi-section-patch multi-section-patch --agent codex --scope user
```

Replace `codex` with another supported agent ID:

| Agent | ID |
| --- | --- |
| Claude Code | `claude-code` |
| Codex | `codex` |
| Cursor | `cursor` |
| Gemini CLI | `gemini-cli` |
| GitHub Copilot | `github-copilot` |
| Generic Agent Skills host | `universal` |

Use `--scope project` instead of `--scope user` when the skill should apply
only to the current repository.

On macOS or Linux, a GitHub-based copy may not preserve executable mode bits.
If the first use reports `permission denied`, make only the binary matching the
current platform executable:

```text
chmod +x <installed-skill-directory>/scripts/multi-section-patch-darwin-arm64
```

Replace the filename with the one for the current OS and architecture.

### Install with `npx skills`

The open-source [`skills` CLI](https://github.com/vercel-labs/skills) provides
a convenient project-scoped installation. Run it from the target repository
root:

```text
npx --yes skills@1.5.19 add rudra2112/multi-section-patch --skill multi-section-patch --agent codex --yes
```

Replace `codex` with an agent ID from the table above. Repeat the command for
another agent, or deliberately use `--agent "*"` to install for every agent
recognized by the installer.

The pinned installer requires Node.js 22.20.0 or newer only while it runs.
Node.js is not needed by the installed skill. If the filesystem cannot create
symlinks, add `--copy`.

### Confirm that the agent sees it

Start or reload the coding agent from a repository where the skill is in
scope, then inspect its available skills:

- In Codex, type `$` and select `multi-section-patch`.
- In Claude Code, invoke `/multi-section-patch`.
- In GitHub Copilot CLI, run `/skills info multi-section-patch`.
- In another host, use its skill list or restart the session if it does not
  watch skill directories for changes.

Then try:

```text
Use the multi-section-patch skill to read the "Install with GitHub CLI"
section and the "Install with npx skills" section from README.md.
Return the resolved ranges and SHA-256 digests.
```

The repository is public and can be installed now. An unpinned remote install
resolves the latest tagged release when available, then falls back to the
default branch. Pin a reviewed tag or commit for repeatable installations.

## Why use Multi Section Patch?

Coding agents often need a few related sections from several files, not every
byte in those files. Ordinary file reads waste context, while ordinary search
results may omit the boundaries needed for a safe edit. Whole-file rewrites
also make it easy to disturb unrelated formatting or concurrent changes.

Multi Section Patch provides:

- **Focused context.** Read only the bounded sections needed for the task,
  which can reduce noise and token use.
- **One multi-file request.** Gather related sections from many files with one
  command or JSON specification.
- **Exact evidence.** Every result includes the canonical path, resolved
  one-based range, exact content, and SHA-256 digest.
- **Reviewable edits.** Editing is a dry run by default and prints the complete
  proposed diff.
- **Guarded writes.** SHA-256, required-content, overlap, stale-snapshot,
  file-identity, staging, rollback, and backup checks protect changes.
- **Minimal disturbance.** Unrelated bytes, line-ending style, and final-newline
  state remain unchanged.
- **Local operation.** The executable makes no network requests, executes no
  selected content, and emits no telemetry.
- **Portable delivery.** One skill contract works across compatible coding
  agents, and one native executable covers each supported OS/architecture pair.
- **No language runtime.** The installed command does not depend on Python,
  Node.js, Go, or a shell interpreter.

### Good uses

Use Multi Section Patch when:

- several files contain corresponding sections that must be reviewed together;
- a line range, heading, literal marker, or RE2 pattern precisely defines the
  relevant text;
- an agent should show a complete multi-file diff before writing;
- the current content must match a previously reviewed digest;
- preserving unrelated formatting and bytes matters.

### Poor uses

Use another tool when:

- the input is binary, non-UTF-8, or not a regular file;
- the task is a semantic AST refactor such as renaming a symbol across a
  language;
- the desired text cannot be bounded reliably;
- the task is an open-ended whole-repository search;
- the platform is not in the supported matrix;
- the workflow requires a filesystem-atomic transaction across multiple files.

## How installation works

Multi Section Patch is a skill directory, not an agent plugin API and not a
hosted service. A marketplace can improve discovery, but it is not required
for installation or execution.

The common project destinations are:

| Host | Project destination |
| --- | --- |
| Claude Code | `.claude/skills/multi-section-patch/` |
| Codex | `.agents/skills/multi-section-patch/` |
| Cursor | `.agents/skills/multi-section-patch/` |
| Gemini CLI | `.agents/skills/multi-section-patch/` |
| GitHub Copilot | `.agents/skills/multi-section-patch/` |
| Another compatible host | Its documented Agent Skills directory |

Personal or user-wide destinations differ by host. Let `gh skill` or
`npx skills` select the correct location instead of guessing it.

Compatibility has three requirements:

1. The host discovers the Agent Skills `SKILL.md` format.
2. The host can invoke a local executable.
3. The machine matches one of the supported OS/architecture pairs.

This makes the skill broadly portable, but it does not make every AI chat
product compatible. A hosted or locked-down agent that cannot discover local
skills or execute local files cannot use the bundled command.

### What gets installed

The complete directory is:

```text
multi-section-patch/
├── SKILL.md
├── LICENSE.txt
├── SHA256SUMS
├── references/
│   └── CLI.md
└── scripts/
    ├── multi-section-patch-darwin-amd64
    ├── multi-section-patch-darwin-arm64
    ├── multi-section-patch-linux-amd64
    ├── multi-section-patch-linux-arm64
    ├── multi-section-patch-windows-amd64.exe
    └── multi-section-patch-windows-arm64.exe
```

Preserve the whole directory. `SKILL.md` refers to the CLI reference and native
executables by relative path.

An Agent Skills-compatible host normally uses progressive disclosure:

1. It initially reads only the skill name and description.
2. It loads the full `SKILL.md` when the task matches or the user invokes it.
3. The instructions select exactly one executable for the current platform.
4. The detailed `references/CLI.md` is loaded only when needed.
5. The selected executable reads local input and returns bounded results or a
   proposed diff.

The other five executables remain inert files on disk.

### Manual installation

Manual installation needs no Node.js or Python:

1. Download or clone this repository.
2. Review `skills/multi-section-patch/SKILL.md`.
3. Verify the files in `skills/multi-section-patch/scripts/` against
   `skills/multi-section-patch/SHA256SUMS`.
4. Copy the complete `skills/multi-section-patch/` directory to the project
   destination for the chosen host.
5. On macOS or Linux, make only the matching native executable executable if
   the copy did not preserve its mode.
6. Reload or restart the agent and confirm the skill appears.

For example, on an Apple-silicon Mac:

```text
chmod +x <skill-directory>/scripts/multi-section-patch-darwin-arm64
```

For repeatable manual installations, prefer a tagged source archive from
[GitHub Releases](https://github.com/rudra2112/multi-section-patch/releases)
over a moving branch.

## Use it through a coding agent

The simplest interface is a natural-language request. Name the skill when you
want deterministic activation.

### Read related sections

```text
Use multi-section-patch to read "## Development setup" from CONTRIBUTING.md
and "## Build, test, and contribute" from README.md. Include three context
lines and return the exact ranges and SHA-256 digests.
```

### Prepare a guarded edit

```text
Use multi-section-patch to update the bounded installation sections in
README.md and CONTRIBUTING.md. Read them first, guard each edit with the
returned SHA-256, show the complete dry-run diff, and do not apply it until I
approve.
```

### Apply an approved edit

```text
The dry-run diff is approved. Revalidate the same multi-section-patch edit
specification and apply it. Report any nonzero exit or recovery file.
```

A well-behaved agent follows this sequence:

1. Read every target section.
2. Build a tightly bounded JSON edit specification.
3. Add `expected_sha256` when the reviewed content must remain unchanged.
4. Run `edit` without `--apply`.
5. Present the entire diff.
6. Wait for approval.
7. Rerun with `--apply`, which validates the current files again before
   writing.

## Direct CLI guide

Agents normally choose the executable automatically. Humans and other tools
can invoke it directly.

### Select the native executable

| Operating system | Architecture | Bundled executable |
| --- | --- | --- |
| macOS | Intel 64-bit (`amd64`) | `scripts/multi-section-patch-darwin-amd64` |
| macOS | Apple silicon (`arm64`) | `scripts/multi-section-patch-darwin-arm64` |
| Linux | x86-64 (`amd64`) | `scripts/multi-section-patch-linux-amd64` |
| Linux | ARM64 (`arm64`) | `scripts/multi-section-patch-linux-arm64` |
| Windows | x86-64 (`amd64`) | `scripts/multi-section-patch-windows-amd64.exe` |
| Windows | ARM64 (`arm64`) | `scripts/multi-section-patch-windows-arm64.exe` |

In the examples below, `"<multi-section-patch>"` means the quoted absolute
path to the matching executable.

macOS and Linux native executables conventionally have no filename extension.
Windows executables use `.exe`. These are compiled native programs, not Python
scripts, so a `.py` suffix would be incorrect.

### Read selectors

```text
"<multi-section-patch>" read [--spec FILE] [--context N] [--json] [--no-line-numbers] [SELECTOR...]
```

Compact selectors have these forms:

| Form | Selection |
| --- | --- |
| `FILE` | The complete file |
| `FILE@START:END` | Inclusive, one-based line range |
| `FILE@START..END` | Literal start marker through before the literal end marker |
| `FILE@/START/../END/` | Start and end using Go/RE2 regular expressions |
| `FILE@START` | Literal start marker through end of file |

Either numeric endpoint may be omitted. Literal markers match anywhere on a
line. Marker selections include the start line and exclude the end line by
default.

Examples:

```text
"<multi-section-patch>" read "README.md@10:40"
"<multi-section-patch>" read "README.md@## Quick start..## Why use Multi Section Patch?"
"<multi-section-patch>" read "src/main.go@/func main/../^}/" --context 3
"<multi-section-patch>" read "SECURITY.md@## Security boundary"
```

Use `--json` when another program will consume the result:

```text
"<multi-section-patch>" read --json "README.md@## Quick start..## Why use Multi Section Patch?"
```

Use a JSON specification when there are many sections or when a path or marker
would make compact `@` or `..` syntax ambiguous:

```json
{
  "sections": [
    {
      "name": "quick-start",
      "file": "README.md",
      "start": "## Quick start",
      "end": "## Why use Multi Section Patch?"
    },
    {
      "name": "development-setup",
      "file": "CONTRIBUTING.md",
      "start": "## Development setup",
      "end": "## Code and documentation"
    },
    {
      "name": "main-function",
      "file": "cmd/multi-section-patch/main.go",
      "start_regex": "^func main",
      "end_regex": "^}"
    }
  ]
}
```

Run it with:

```text
"<multi-section-patch>" read --spec sections.json
```

Use `--spec -`, or omit a named specification and all selectors, to read JSON
from standard input.

Every successful result contains:

- the canonical path;
- the optional display name;
- the resolved one-based line range;
- the SHA-256 of the exact selected bytes; and
- the selected content.

Human-readable output uses guarded `<<<MULTI_SECTION_PATCH ...>>>` and
`<<<END_MULTI_SECTION_PATCH ...>>>` boundaries. Selected lines that resemble a
boundary are escaped so file content cannot impersonate the output structure.

### Prepare an edit

Editing accepts a JSON specification:

```json
{
  "edits": [
    {
      "name": "readme-installation",
      "file": "README.md",
      "start": "## Quick start",
      "end": "## Why use Multi Section Patch?",
      "replacement_file": "replacements/readme-quick-start.md",
      "expected_sha256": "<digest-returned-by-read>",
      "must_contain": [
        "gh skill install",
        "npx --yes skills"
      ]
    },
    {
      "name": "contributing-command",
      "file": "CONTRIBUTING.md",
      "start_line": 12,
      "end_line": 18,
      "replacement": "Replacement UTF-8 text.\n"
    }
  ]
}
```

Each edit uses exactly one selector family:

- `start_line` and `end_line`;
- `start` and `end` literal markers; or
- `start_regex` and `end_regex` RE2 patterns.

Each edit also uses exactly one replacement source:

- `replacement` for inline UTF-8 text; or
- `replacement_file` for UTF-8 text read from another regular file.

Useful guards and controls are:

| Field | Purpose |
| --- | --- |
| `expected_sha256` | Require the selected bytes to match the previous read |
| `must_contain` | Require one or more strings in the selected bytes |
| `include_start` | Replace the start marker line; defaults to `true` |
| `include_end` | Replace the end marker line; defaults to `false` |
| `occurrence` | Choose a later one-based start-marker match |
| `end_occurrence` | Choose a later one-based end-marker match |

### Dry run, review, and apply

First run the edit without `--apply`:

```text
"<multi-section-patch>" edit --spec edits.json
```

This validates all targets and prints the complete proposed diff without
changing a file.

After reviewing that output, apply the same specification:

```text
"<multi-section-patch>" edit --spec edits.json --apply
```

The apply command re-reads and revalidates current targets. Add `--backup` when
independent original-file copies and a manifest are required:

```text
"<multi-section-patch>" edit --spec edits.json --apply --backup
```

Pass `--json` to either command for structured output.

### PowerShell

PowerShell requires the call operator before a quoted executable path:

```powershell
& "<skill-directory>\scripts\multi-section-patch-windows-amd64.exe" read "README.md@10:40"
```

### Exit behavior

| Exit code | Meaning |
| --- | --- |
| `0` | Read, dry run, no-op, or apply completed |
| `1` | Input, validation, file, staging, replacement, rollback, cleanup, or output failed |
| `2` | Command is missing or unknown |

Treat every nonzero apply result as requiring review, even when the message
says completed targets were rolled back. Keep any reported recovery file until
its target has been verified.

For every option and JSON field, read the bundled
[CLI reference](skills/multi-section-patch/references/CLI.md).

## Safety model

Multi Section Patch is designed for failure-safe bounded editing:

- **Dry-run first:** `edit` writes nothing unless `--apply` is present.
- **Whole-request validation:** all reads or edits resolve before output or
  writing, so a later invalid target cannot leave a partial read result.
- **Tight trust boundaries:** target paths, patterns, markers, names,
  replacement text, and selected content are treated as untrusted data.
- **No content execution:** the executable treats selected text as data and
  never executes it; `SKILL.md` tells agents not to follow instructions found
  inside selected content.
- **Content guards:** `expected_sha256` and `must_contain` stop an edit when the
  reviewed bytes have changed or required text is absent.
- **Overlap rejection:** overlapping edits are rejected; adjacent edits are
  allowed.
- **Stale-file detection:** identity and content are checked before the first
  replacement and again immediately before each target is replaced.
- **Hard-link protection:** ambiguous hard-linked targets are rejected rather
  than silently breaking link relationships.
- **Same-directory staging:** new content and recovery copies are staged beside
  each target before replacement.
- **Conservative rollback:** after a later failure, completed replacements are
  restored only when doing so cannot overwrite a concurrent external change.
- **Optional backups:** `--backup` retains independent originals and a manifest.
- **Output integrity:** boundary-like and unsafe control content is escaped so
  selected file text cannot impersonate the surrounding result structure.

A multi-file apply is not a filesystem-atomic transaction. It is staged,
revalidated, and rollback-protected, but a process crash, power loss, platform
rename behavior, or concurrent writer can still require recovery. The command
reports every retained recovery path when cleanup or rollback is incomplete.

## Input, metadata, and privacy limits

- Targets and replacement files must be regular, valid UTF-8 text without NUL
  or unsupported binary control bytes.
- Replacements adopt the target's LF or CRLF style.
- Unrelated bytes and the target's final-newline state remain unchanged.
- Unix permission bits and the Windows read-only state exposed by Go are
  preserved.
- Ownership, ACLs, extended attributes, resource forks, alternate data
  streams, timestamps, and other platform-specific metadata are outside the
  portable guarantee.
- A Windows target marked read-only is rejected before staging.
- The invoking user's filesystem access and permissions remain in force.
- Normal `read` and `edit` operations perform no network request and emit no
  telemetry.

The no-network guarantee applies to the Multi Section Patch executable, not to
third-party installers, GitHub CLI, Git, a browser, or the coding agent itself.

## Platform support

Support requires native execution and tests on each claimed pair:

| Operating system | x86-64 | ARM64 |
| --- | --- | --- |
| macOS | Supported | Supported |
| Linux | Supported | Supported |
| Windows | Supported | Supported |

Cross-compilation alone is not treated as support. The GitHub Actions matrix
runs the full suite and smoke-tests the matching native executable on all six
pairs.

The initial binaries are not Apple Developer ID-notarized or Windows
Authenticode-signed. Native CI verifies that they execute, but an operating
system or organization may still block unsigned software. Follow that policy;
do not bypass it.

The `github.com/rudra2112/...` path in `go.mod` identifies the canonical Go
source module. It does not make the built executable contact GitHub, and it
does not prevent anyone from forking or extending the project.

## Troubleshooting

| Problem | Resolution |
| --- | --- |
| The skill is not listed | Confirm the complete directory contains `SKILL.md`, is in the host's supported scope, and reload or restart the agent. |
| `permission denied` on macOS or Linux | Run `chmod +x` on only the executable matching the current OS and architecture. |
| The OS blocks an unsigned binary | Follow local security policy. Use a source build if permitted; do not bypass an organizational control. |
| `exec format error` or equivalent | The wrong OS/architecture executable was selected. Use the platform table above. |
| `npx skills` cannot create a symlink | Repeat the install command with `--copy`. |
| A SHA-256 guard fails | The selected bytes changed. Read the section again, review it, and rebuild the edit specification. |
| A marker is missing or ambiguous | Use a JSON specification, tighter markers, `occurrence`, or `end_occurrence`. |
| A Windows target is read-only | Deliberately remove the read-only attribute before applying, then restore it if required. |
| A PowerShell quoted path does not run | Prefix the quoted executable path with `&`. |
| Apply exits nonzero | Review every reported target and retain any recovery file until the target is verified. |

The `npx skills` installer currently has an
[open global-linking issue](https://github.com/vercel-labs/skills/issues/537).
Prefer its project scope or use GitHub CLI user scope until the issue is
resolved. The installer may collect its own anonymous usage data; its
documented `DISABLE_TELEMETRY=1` or `DO_NOT_TRACK=1` setting disables that
behavior. Multi Section Patch itself sends no telemetry.

## Update and remove

### GitHub CLI installations

Update an unpinned installation:

```text
gh skill update multi-section-patch
```

Pinned installations remain on their selected tag or commit until deliberately
repinned or unpinned.

GitHub CLI currently has no `gh skill remove` or `gh skill list` command. Keep
the destination printed by `gh skill install`. To uninstall, delete only that
exact `multi-section-patch` directory, never its shared parent skills
directory.

Install a reviewed tagged version with:

```text
gh skill install rudra2112/multi-section-patch multi-section-patch --agent codex --scope user --pin <tag>
```

### `npx skills` installations

Update a project installation:

```text
npx --yes skills@1.5.19 update multi-section-patch --project --yes
```

Remove it from one agent:

```text
npx --yes skills@1.5.19 remove multi-section-patch --agent codex --yes
```

Replace `codex` with the agent ID used during installation.

### Manual installations

Replace the complete `multi-section-patch` directory with the reviewed copy
from a newer release. To uninstall, delete only that complete skill directory.

## Build, test, and contribute

Installed users do not need Go. Development and tests require Git and Go 1.23
or newer. Release artifacts use the exact Go 1.26.1 toolchain selected by the
build script.

```text
go test ./...
go vet ./...
git diff --check
```

Build all native artifacts and checksums with:

```text
./scripts/build-artifacts.sh
```

The build uses `CGO_ENABLED=0`, `-trimpath`, `-buildvcs=false`, and stripped
linker flags. It writes the six executables and sorted checksums. Do not edit
generated binaries or `SHA256SUMS` manually.

The current artifact budget is at most 3 MiB per executable and 18 MiB for all
six. Growth beyond either limit is release-blocking until measured, explained,
and deliberately accepted.

Read [CONTRIBUTING.md](CONTRIBUTING.md) before changing behavior.

### Repository layout

```text
cmd/multi-section-patch/          CLI entry point
internal/multisectionpatch/       tested private implementation
scripts/                          reproducible artifact build
skills/multi-section-patch/       installable, self-contained Agent Skill
.github/workflows/ci.yml          native and reproducibility checks
```

The layout is intentionally small. Add another package or directory only when
it owns a distinct responsibility the current structure cannot express.

### Forking

The Go module path does not prevent forks. A contributor normally leaves it
unchanged and opens a pull request to this repository. A permanent independent
fork published as a different Go module should change `go.mod` and the internal
import paths to its own repository URL.

### GitHub Actions usage

CI uses standard public-repository GitHub-hosted runners for the six native
targets and one Linux reproducibility job. It does not use larger runners,
dependency caching, or temporary artifact uploads. GitHub currently provides
free, unlimited standard-runner minutes for public repositories; private
repositories use their included Actions minutes. See GitHub's
[runner reference](https://docs.github.com/en/actions/reference/runners/github-hosted-runners#standard-github-hosted-runners-for-public-repositories)
and [Actions billing](https://docs.github.com/en/billing/concepts/product-billing/github-actions#free-use-of-github-actions).

## Security

Review `SKILL.md`, the selected executable's checksum, and every dry-run diff
before applying changes. Report suspected vulnerabilities privately through
the repository's
[security advisory form](https://github.com/rudra2112/multi-section-patch/security/advisories/new)
as described in [SECURITY.md](SECURITY.md).

## License

Multi Section Patch is available under the [MIT License](LICENSE).
