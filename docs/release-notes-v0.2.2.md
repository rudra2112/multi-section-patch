# Multi Section Patch v0.2.2

Multi Section Patch v0.2.2 improves release provenance, installation
guidance, and public proof without changing the CLI, file formats, safety
model, or supported platforms.

## Highlights

- Adds source and maintainer metadata to the distributed `SKILL.md`.
- Adds tag-only GitHub artifact attestations for all six native executables and
  `SHA256SUMS`.
- Adds a real two-file demonstration of the reviewed-plan workflow.
- Adds a reproducible, carefully scoped byte measurement without presenting it
  as a token, latency, cost, or model-quality benchmark.
- Makes GitHub CLI and `npx skills` installation guidance agent-neutral.
- Documents the current `gh skill list` command and keeps removal guidance
  limited to the exact installed skill directory.

## Install or upgrade

Preview the skill before installing it:

```text
gh skill preview rudra2112/multi-section-patch multi-section-patch
```

Install the tagged release for a supported coding agent:

```text
gh skill install rudra2112/multi-section-patch multi-section-patch --agent AGENT_ID --scope user --pin v0.2.2
```

Replace `AGENT_ID` with a supported host ID such as `claude-code`, `codex`,
`cursor`, `gemini-cli`, or `github-copilot`.

The alternative `npx skills` installer is:

```text
npx --yes skills@1.5.20 add rudra2112/multi-section-patch --skill multi-section-patch --agent AGENT_ID --yes
```

Node.js is required only while that optional installer runs. The installed
skill remains self-contained and requires no Python, Node.js, Go, compiler,
network connection, or background service during normal use.

## Provenance and verification

The tagged GitHub Actions workflow:

- runs the test suite and `go vet`;
- executes native smoke tests on Windows, macOS, and Linux for x86-64 and
  ARM64;
- rebuilds all six executables twice and compares them byte for byte;
- verifies every entry in `SHA256SUMS`; and
- attests the released executables and checksum manifest.

After the tagged workflow succeeds, verify the artifact matching the current
platform from an exact `v0.2.2` checkout:

```text
gh attestation verify \
  skills/multi-section-patch/scripts/multi-section-patch-darwin-arm64 \
  --repo rudra2112/multi-section-patch
```

Replace the filename with the current platform's executable. An attestation
proves where those exact bytes were built; it does not replace source review,
checksum verification, or review of every dry-run diff.

## Compatibility

This release does not change:

- selectors, edit specifications, JSON output, or exit behavior;
- the reviewed `plan_sha256` required by `--expect-plan`;
- the supported Windows, macOS, and Linux x86-64 and ARM64 matrix; or
- the documented filesystem and metadata limits.

The bundled executables and their checksums are unchanged from `v0.2.1`
because this release contains documentation and workflow changes only.

## Security

The executable does not interpret or execute selected file content. Agents
must continue to treat file names and selected content as untrusted data.

Report suspected vulnerabilities through the repository's
[private vulnerability reporting](https://github.com/rudra2112/multi-section-patch/security/advisories/new)
instead of opening a public issue.
