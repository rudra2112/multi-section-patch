# Security policy

## Supported versions

Before the first tag, no version is supported. After publication, security
fixes are provided for the latest tagged release. Unreleased branch content
and older releases should not be treated as supported.

## Report a vulnerability

After the public repository enables private vulnerability reporting, do not
open a public issue for a suspected vulnerability. Use the repository's
[private vulnerability report](https://github.com/rudra2112/multi-section-patch/security/advisories/new)
and include:

- the affected version, operating system, and architecture;
- a minimal reproduction using non-sensitive files;
- expected and observed behavior;
- the security or data-loss impact; and
- any known workaround.

Do not include credentials, private repository content, or personal data.
Maintainers will coordinate validation, remediation, and disclosure in the
private advisory. No fixed response time is promised.

## Security boundary

Multi Section Patch operates only on caller-selected local files. Normal
`read` and `edit` operations make no network requests, send no telemetry, start
no daemon, and execute no selected file content.

Security-sensitive behavior includes:

- path, symlink, hard-link, traversal, and backup containment handling;
- UTF-8 and binary-input rejection;
- regex and range validation;
- dry-run, stale-snapshot, overlap, staging, rollback, and recovery behavior;
- structured-output and terminal-control injection resistance; and
- preservation of unrelated bytes, line endings, and applicable permissions.

The local files and permissions available to the invoking user remain outside
Multi Section Patch's control. Third-party installers, GitHub, Git, and
browsers have their own network and telemetry policies and are not part of the
normal-operation no-network guarantee.

## Supply-chain guidance

Install tagged releases, verify the bundled `SHA256SUMS`, and review
`SKILL.md` before use. Build artifacts must be generated from the tagged source
with the repository build script and pass the native platform matrix. Initial
release binaries are not Apple Developer ID-notarized or Windows
Authenticode-signed; follow local security policy rather than bypassing an
operating-system warning.
