# Multi Section Patch v0.1.0

Multi Section Patch v0.1.0 is the first public release of a vendor-neutral
Agent Skill for precise, guarded reads and edits across multiple local text
files.

The installed skill is self-contained. It uses a bundled native executable and
needs no Python, Node.js, Go, compiler, network connection, or background
service during normal use.

## Highlights

- Read exact sections from multiple files in one request.
- Select whole files, inclusive line ranges, literal markers, headings, or
  Go/RE2 regular-expression bounds.
- Receive canonical paths, resolved line ranges, exact content, and SHA-256
  digests for every selection.
- Preview a complete multi-file diff by default; write only with explicit
  `--apply`.
- Guard edits with `expected_sha256` and `must_contain`.
- Reject invalid bounds, overlapping edits, stale targets, binary input, and
  ambiguous hard-linked files before writing.
- Stage replacement and recovery files beside their targets and report any
  incomplete rollback or cleanup.
- Preserve unrelated bytes, LF or CRLF style, final-newline state, and portable
  permission bits.
- Run locally without network requests or telemetry from the Multi Section
  Patch executable.

## Install

Review the complete skill before installing:

```text
gh skill preview rudra2112/multi-section-patch multi-section-patch@v0.1.0
```

Install the tagged release for one agent with GitHub CLI 2.91.0 or newer:

```text
gh skill install rudra2112/multi-section-patch multi-section-patch --agent <agent-id> --scope user --pin v0.1.0
```

Common agent IDs are `claude-code`, `codex`, `cursor`, `gemini-cli`, and
`github-copilot`.

For a project-scoped installation through the third-party `skills` CLI, run
this from the target repository root:

```text
npx --yes skills@1.5.19 add rudra2112/multi-section-patch --skill multi-section-patch --agent <agent-id> --yes
```

Use the GitHub CLI command above when the installation must remain pinned to
`v0.1.0`. The `skills@1.5.19` installer requires Node.js 22.20.0 or newer only
during installation. Node.js is not a runtime dependency of Multi Section
Patch.

On macOS or Linux, a GitHub-based copy may not preserve executable mode bits.
If the first run reports `permission denied`, apply `chmod +x` only to the
binary matching the current OS and architecture.

See the
[installation guide](https://github.com/rudra2112/multi-section-patch#quick-start)
for manual installation, updates, removal, and host-specific verification.

## Included skill

The release contains:

- `SKILL.md`, the agent workflow and platform selection instructions;
- `references/CLI.md`, the complete selector, JSON, output, and failure
  contract;
- `SHA256SUMS`, covering every bundled executable;
- the MIT license; and
- six native executables.

## Supported platforms

| Operating system | Architecture |
| --- | --- |
| macOS | x86-64 |
| macOS | ARM64 |
| Linux | x86-64 |
| Linux | ARM64 |
| Windows | x86-64 |
| Windows | ARM64 |

Each pair is tested on a matching native GitHub-hosted runner. Cross-compilation
alone is not treated as support.

## Safety and privacy

- `edit` is a dry run unless `--apply` is present.
- Every target, selector, replacement, and guard is validated before the first
  write.
- Target identity, bytes, and applicable permissions are rechecked after
  staging and immediately before replacement.
- A later replacement failure triggers conservative reverse-order rollback.
- Selected content is treated as data and is never executed by the bundled
  command.
- Normal `read` and `edit` operations make no network requests and emit no
  telemetry.

Review `SKILL.md`, verify `SHA256SUMS`, and inspect every dry-run diff before
applying changes.

## Known limitations

- Multi-file apply is staged and rollback-protected, but it is not a
  filesystem-atomic transaction.
- Inputs must be regular, valid UTF-8 text files without unsupported binary
  control bytes.
- The tool performs bounded text editing, not syntax-aware AST refactoring.
- Ownership, ACLs, extended attributes, resource forks, alternate data
  streams, and timestamps are outside the portable metadata guarantee.
- Windows read-only targets are rejected before staging.
- The binaries are not Apple Developer ID-notarized or Windows
  Authenticode-signed. Follow local security policy rather than bypassing
  platform protections.
- Hosts must support Agent Skills discovery and local process execution.

## Release verification

The release candidate must pass:

- `go test ./...`;
- `go vet ./...`;
- native test and smoke jobs for all six supported platform pairs;
- two reproducible all-target builds compared byte for byte;
- validation of every entry in `SHA256SUMS`; and
- the enforced size budget of 3 MiB per executable and 18 MiB in total.

## Security

Report suspected vulnerabilities through the repository's
[private vulnerability reporting](https://github.com/rudra2112/multi-section-patch/security/advisories/new)
instead of opening a public issue.
