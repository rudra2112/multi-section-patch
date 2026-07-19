# Multi Section Patch

Multi Section Patch is a vendor-neutral
[Agent Skill](https://agentskills.io/) for reading and safely replacing exact
sections across multiple local text files. It keeps agent context focused and
makes multi-file edits reviewable.

The skill needs no separately installed language runtime after installation.
It bundles native executables, so normal use needs no Python, Node.js, Go,
compiler, network connection, or background service. An optional installer can
have its own prerequisites and network behavior.

## What it provides

- One command can select line ranges, literal markers, or regular-expression
  bounds from multiple UTF-8 files.
- Every selected section includes its resolved range and SHA-256 digest.
- Editing is a dry run by default and writes only with explicit `--apply`.
- Hash, content, overlap, stale-snapshot, staging, rollback, and optional backup
  checks protect multi-file writes.
- Normal `read` and `edit` operations are local-only and send no telemetry.

## Install

The remote commands below become usable after
`rudra2112/multi-section-patch` is public. Before publication, contributors can
build and validate the local skill, but remote installation will fail because
the repository does not exist on GitHub yet.

### `npx skills`

The open-source [`skills` CLI](https://github.com/vercel-labs/skills) installs
the skill into an agent-specific project directory. After publication, replace
`<agent-id>` with the coding agent you use:

```text
npx --yes skills@1.5.19 add rudra2112/multi-section-patch --skill multi-section-patch --agent <agent-id> --yes
```

Repeat that command for another agent. People who intentionally want every
agent target recognized by the installer can use `--agent "*"`, but that may
create directories for agents they do not use.

Update the project installation or remove it from one agent with:

```text
npx --yes skills@1.5.19 update multi-section-patch --project --yes
npx --yes skills@1.5.19 remove multi-section-patch --agent <agent-id> --yes
```

Node.js is required only to run this installer. It is not required to execute
the installed skill. The installer also supports global installation, but its
current global-linking behavior has an
[open cross-agent bug](https://github.com/vercel-labs/skills/issues/537).
Use project scope above or GitHub CLI user scope below until that issue is
resolved. If the local filesystem cannot create symlinks, repeat the add command
with `--copy`. The third-party installer may collect its own anonymous usage
data; its documented `DISABLE_TELEMETRY=1` or `DO_NOT_TRACK=1` setting disables
that behavior. The Multi Section Patch executable itself sends no telemetry.

### GitHub CLI

[`gh skill`](https://cli.github.com/manual/gh_skill_install) is a preview feature
in GitHub CLI 2.90.0 or newer. Preview the skill, then run one install command
for each agent you use:

```text
gh skill preview rudra2112/multi-section-patch multi-section-patch
gh skill install rudra2112/multi-section-patch multi-section-patch --agent <agent-id> --scope user
```

Use `gh skill install --help` to see the agent IDs supported by the installed
preview version. Update an installation with:

```text
gh skill update multi-section-patch
```

If the installed preview version has no remove command, keep the destination
printed by `gh skill install` and delete only its `multi-section-patch`
directory, not the shared parent.

On macOS and Linux, a GitHub-based install may not preserve executable mode
bits. If execution reports `permission denied`, make only the selected
platform binary executable:

```text
chmod +x <skill-directory>/scripts/multi-section-patch-darwin-arm64
```

Replace the filename with the one matching your platform.

### Manual project installation

After the first tagged release, download its source archive from
[GitHub Releases](https://github.com/rudra2112/multi-section-patch/releases),
verify the bundled binaries against
`skills/multi-section-patch/SHA256SUMS`, and copy the complete
`skills/multi-section-patch` directory to the appropriate project path:

| Agent | Exact project path |
| --- | --- |
| Claude Code | `.claude/skills/multi-section-patch/` |
| Codex, Cursor, GitHub Copilot, Gemini CLI | `.agents/skills/multi-section-patch/` |
| Another compatible agent | That agent's documented Agent Skills directory |

To update, replace that complete `multi-section-patch` directory with the one
from a newer tagged archive. To remove, delete that exact directory. On macOS
and Linux, apply `chmod +x` to the one native binary selected below after each
manual update.

### What gets installed

Every installation must preserve this complete directory:

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

An Agent Skills-compatible host initially discovers the skill name and
description from `SKILL.md`. When a matching task arrives, the agent reads the
instructions, selects exactly one bundled executable for the current platform,
and loads `references/CLI.md` only when it needs detailed syntax. The other
five binaries remain files on disk and are not executed.

## Target platforms

Version 0.1 targets these native pairs:

| Operating system | Architecture | Bundled executable |
| --- | --- | --- |
| macOS | Intel 64-bit (`amd64`) | `scripts/multi-section-patch-darwin-amd64` |
| macOS | Apple silicon (`arm64`) | `scripts/multi-section-patch-darwin-arm64` |
| Linux | x86-64 (`amd64`) | `scripts/multi-section-patch-linux-amd64` |
| Linux | ARM64 (`arm64`) | `scripts/multi-section-patch-linux-arm64` |
| Windows | x86-64 (`amd64`) | `scripts/multi-section-patch-windows-amd64.exe` |
| Windows | ARM64 (`arm64`) | `scripts/multi-section-patch-windows-arm64.exe` |

A platform becomes supported only after its native test suite passes for the
release; cross-compilation alone is not treated as support.

macOS and Linux executables conventionally have no filename extension; Windows
executables use `.exe`. They are native programs, not Python scripts, so a
`.py` suffix would be incorrect. The `github.com/rudra2112/...` Go module name
is the source package's globally unique identity; the built executable does not
contact GitHub during normal use.

The initial binaries are not Apple Developer ID-notarized or Windows
Authenticode-signed. Native CI proves that they execute on the supported
platforms, but an operating-system or organizational policy may still block
unsigned software. Do not bypass that policy; report the restriction so signing
can be prioritized if it becomes an adoption blocker.

## Agent compatibility

Multi Section Patch has one `SKILL.md` and one CLI contract for all compatible
agents. It does not contain vendor-specific prompts or call a vendor API.
Current installer IDs include:

| Agent | Installer ID |
| --- | --- |
| Claude Code | `claude-code` |
| Codex | `codex` |
| Cursor | `cursor` |
| Gemini CLI | `gemini-cli` |
| GitHub Copilot | `github-copilot` |
| Generic Agent Skills directory | `universal` |

The compatibility boundary is explicit: an agent must support the Agent Skills
format, be able to invoke a local command, and run on one of the six target
platforms above. An agent without automatic skill discovery can still use the
tool if instructed to read `SKILL.md` directly and invoke the matching binary.
That is how new agents can adopt the repository without code changes here.

## Use

Ask your coding agent to use the `multi-section-patch` skill when relevant, or
invoke the selected bundled executable directly.

The agent workflow is:

1. Match the task against the skill's name and description.
2. Read `SKILL.md` and select the current platform's native executable.
3. Run `read` to gather exact bounded context and hashes.
4. Run `edit` without `--apply` to produce a complete dry-run diff.
5. Review the diff, then rerun with `--apply` only when it is accepted.

For example:

```text
"<multi-section-patch>" read "README.md@10:40" "CONTRIBUTING.md@## Development setup..## Code and documentation"
"<multi-section-patch>" edit --spec edits.json
"<multi-section-patch>" edit --spec edits.json --apply
```

`<multi-section-patch>` means the absolute path to the executable matching the
current OS and architecture; keep the executable path quoted when it contains
spaces. The dry run prints the complete proposed diff and changes no target.
See
[`SKILL.md`](skills/multi-section-patch/SKILL.md) for the agent workflow and
the bundled [CLI reference](skills/multi-section-patch/references/CLI.md) for
selectors, JSON fields, guards, output, and exit behavior.

PowerShell requires its call operator before a quoted executable path:

```powershell
& "<skill-directory>\scripts\multi-section-patch-windows-amd64.exe" read "README.md@10:40"
```

## Forking and extending

The Go module path identifies the canonical upstream source; it does not stop
anyone from forking, changing, building, or contributing to the repository. A
contributor normally leaves the module path unchanged and opens a pull request
to upstream. Someone publishing a permanent independent fork as a different Go
module should change `go.mod` and its internal import paths to that fork's
repository URL.

## Security and privacy

Multi Section Patch treats paths, patterns, names, and file content as
untrusted data. It does not execute selected content. Normal CLI use reads only
the requested local files, performs no network requests, and emits no
telemetry.

The no-network guarantee applies to the Multi Section Patch executable, not to
third-party installers such as `npx skills`, GitHub CLI, Git, or a browser.
Report security issues privately as described in
[SECURITY.md](SECURITY.md).

## Build and test

Development and tests require Go 1.23 or newer. Release artifacts use the exact
Go 1.26.1 toolchain selected by the build script. Installed users need no Go.

```text
go test ./...
go vet ./...
git diff --check
```

Build all release artifacts with:

```text
./scripts/build-artifacts.sh
```

The build pins Go 1.26.1 and uses `CGO_ENABLED=0`, `-trimpath`,
`-buildvcs=false`, and stripped linker flags, then writes the six binaries and
sorted checksums. Do not edit generated binaries or `SHA256SUMS` manually.

The v0.1 release budget is at most 3 MiB per executable and 18 MiB for all six
executables. Treat growth beyond either limit as a release-blocking regression
until it is measured, explained, and the budget is deliberately revised.

### GitHub Actions cost

The workflow executes all six binaries on standard GitHub-hosted Linux, macOS,
and Windows runners, then runs one Ubuntu reproducibility job. For a public
repository, GitHub currently provides free, unlimited minutes on these standard
runners. The matrix peaks at six concurrent jobs, including two macOS jobs,
which is below GitHub Free's limits of 20 concurrent standard jobs and five
concurrent macOS jobs.

The workflow does not use larger runners, which are billed separately, and it
disables dependency caching and temporary artifact uploads. Repeated CI runs
therefore consume neither Actions cache storage nor artifact storage. A private
repository would consume its included Actions minutes. See GitHub's
[runner reference](https://docs.github.com/en/actions/reference/runners/github-hosted-runners#standard-github-hosted-runners-for-public-repositories),
[concurrency limits](https://docs.github.com/en/actions/reference/limits#job-concurrency-limits-for-github-hosted-runners),
and [Actions billing](https://docs.github.com/en/billing/concepts/product-billing/github-actions#free-use-of-github-actions).

## Repository layout

```text
cmd/multi-section-patch/          CLI entry point
internal/multisectionpatch/       tested implementation
scripts/                          reproducible artifact build
skills/multi-section-patch/       installable, self-contained Agent Skill
.github/workflows/ci.yml          native platform and reproducibility checks
```

The structure is intentionally small. Add another package or directory only
when it owns a distinct responsibility that the current layout cannot express
clearly.

## License

[MIT](LICENSE)
