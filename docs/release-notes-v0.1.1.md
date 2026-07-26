# Multi Section Patch v0.1.1

Multi Section Patch v0.1.1 fixes regular-expression selectors that use an
end-of-line anchor on ordinary text files.

## Fixed

- Patterns ending in `$`, such as `^BEGIN$`, now match the logical content of
  newline-terminated lines.
- The fix applies to both LF and CRLF files.
- The shared selector fix applies to both `read` and `edit`.
- Selected content and original line endings remain byte-for-byte unchanged.
- Literal markers, headings, line ranges, edit guards, and write behavior are
  unchanged.
- There is no CLI, JSON schema, or output-format change.

## Documentation

- Every production Go function now has a concise doc comment describing its
  purpose and important mechanism or safety invariant.
- Package and command documentation remain available through standard Go
  tooling.

## Install or upgrade

Review the exact tagged skill before installing:

```text
gh skill preview rudra2112/multi-section-patch multi-section-patch@v0.1.1
```

Install or upgrade a user-scoped copy for a supported coding agent:

```text
gh skill install rudra2112/multi-section-patch multi-section-patch --agent <agent-id> --scope user --pin v0.1.1
```

Common agent IDs include `claude-code`, `codex`, `cursor`, `gemini-cli`, and
`github-copilot`.

The installed skill remains self-contained and needs no Python, Node.js, Go,
compiler, network connection, or background service during normal use.

## Verification

This patch includes rebuilt native executables and updated SHA-256 checksums
for all six supported operating-system and architecture pairs:

- macOS x86-64 and ARM64;
- Linux x86-64 and ARM64; and
- Windows x86-64 and ARM64.

The release candidate passed the full Go test suite, `go vet`, LF and CRLF
regression tests, checksum validation, native command smoke testing, and two
byte-for-byte identical all-target builds.

## Security

Report suspected vulnerabilities through the repository's
[private vulnerability reporting](https://github.com/rudra2112/multi-section-patch/security/advisories/new)
instead of opening a public issue.
