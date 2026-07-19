# Contributing

Thanks for improving Multi Section Patch. Keep changes small, portable, and
grounded in an accepted requirement.

## Before changing behavior

1. Read `README.md`, `skills/multi-section-patch/SKILL.md`, and the focused
   `skills/multi-section-patch/references/CLI.md` contract.
2. State any intended public-contract change explicitly in the issue or pull
   request.
3. Add the smallest CLI-boundary test that fails for the intended reason.
4. Implement the minimum standard-library solution.

Report vulnerabilities privately through [SECURITY.md](SECURITY.md), not in a
public issue.

## Development setup

Development and tests require Git and Go 1.23 or newer. Release artifacts use
the exact Go 1.26.1 toolchain selected by the build script. End users need
neither after installation.

```text
go test ./...
go vet ./...
git diff --check
```

Run the smallest relevant test while iterating, then all three checks before a
pull request.

Release maintainers also need a POSIX-compatible shell and either `sha256sum`
or `shasum` to run the canonical all-platform artifact build. Windows
contributors can run that release step in an environment such as Git Bash or
WSL; ordinary Go development and installed Multi Section Patch execution do
not require it.

## Code and documentation

- Preserve dry-run-by-default behavior and all validation, staging, rollback,
  backup, line-ending, permission, and local-only guarantees.
- Use the Go standard library unless a dependency is required for a measured,
  documented platform guarantee.
- Add Go doc comments to exported declarations. Comment invariants, portability
  constraints, and recovery decisions; do not narrate obvious syntax.
- Keep the bundled `skills/multi-section-patch/LICENSE.txt` byte-identical to
  the repository-root `LICENSE`.
- Return concise contextual errors without stack traces, command evaluation, or
  unescaped untrusted content.
- Keep public instructions vendor-neutral and all paths repository-relative.
- Keep the hierarchy responsibility-based: command entry points under `cmd/`,
  private implementation under `internal/`, and distributable skill files under
  `skills/`.
- Extract another package or directory only when it has a clear independent
  responsibility or platform boundary.

## Release artifacts

Generate, never hand-edit, bundled executables and checksums:

```text
./scripts/build-artifacts.sh
```

The release set is:

```text
skills/multi-section-patch/scripts/multi-section-patch-darwin-amd64
skills/multi-section-patch/scripts/multi-section-patch-darwin-arm64
skills/multi-section-patch/scripts/multi-section-patch-linux-amd64
skills/multi-section-patch/scripts/multi-section-patch-linux-arm64
skills/multi-section-patch/scripts/multi-section-patch-windows-amd64.exe
skills/multi-section-patch/scripts/multi-section-patch-windows-arm64.exe
skills/multi-section-patch/SHA256SUMS
```

Cross-compilation is only a build check. A release requires the native suite
to pass on every claimed OS/architecture pair.

## Pull request checklist

- The change maps to an accepted requirement and acceptance scenario.
- New behavior was test-driven at the CLI boundary.
- `go test ./...`, `go vet ./...`, and `git diff --check` pass.
- Platform-sensitive changes run on every affected native platform.
- Public documentation and generated artifacts are current.
- The diff contains no secrets, machine-specific paths, unrelated changes, or
  new telemetry/network behavior.
