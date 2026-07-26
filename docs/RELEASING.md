# Publishing and releasing

This checklist is for the repository owner. It separates local preparation,
initial GitHub publication, native release proof, and the tagged release. Do not
claim a supported platform until its native CI job passes.

## Prerequisites

- GitHub account `rudra2112` with GitHub CLI authentication.
- Git and Go 1.26.1 for release builds; `go.mod` records the language floor.
- A clean review of the complete local tree.
- No Git LFS: installers must receive real executable bytes, not pointer files.

Check authentication without changing GitHub:

```text
gh auth status
```

If GitHub CLI shows another active account, stop before any publication
command. After `rudra2112` has been authenticated locally, the repository owner
can select it and verify again:

```text
gh auth switch --user rudra2112
gh auth status
```

If that account is not yet authenticated, the repository owner can add it
interactively with:

```text
gh auth login --hostname github.com --web
```

Do not create the repository from a different account and plan to move it
later; repository ownership is part of the public installation URL.

## Prepare the local release candidate

Run from the repository root:

```text
go test ./...
go vet ./...
go test ./internal/multisectionpatch -run '^$' -bench 'Benchmark(Read|Edit)SizeDoubling|Benchmark(ManyEdits|DiffManyHunks)' -benchtime=100ms
./scripts/build-artifacts.sh
(cd skills/multi-section-patch && shasum -a 256 -c SHA256SUMS)
cmp LICENSE skills/multi-section-patch/LICENSE.txt
gh skill publish --dry-run .
git diff --check
git status --short
```

On Linux, use `sha256sum --check SHA256SUMS` instead of `shasum`.
Confirm that the skill contains `SKILL.md`, `LICENSE.txt`, the focused
`references/CLI.md`, `SHA256SUMS`, and the six documented native executables.
Confirm that the four macOS/Linux executables are executable, each executable
is at most 3 MiB, and all six executables total at most 18 MiB.

The initial binaries are not Apple Developer ID-notarized or Windows
Authenticode-signed. Native tests prove execution on the hosted runners, not
acceptance by every organizational security policy. Do not instruct users to
bypass platform protections.

## Create the public repository

These commands intentionally create the first commit, GitHub repository, and
push. They are owner-run publication actions:

```text
test "$(gh api user --jq .login)" = "rudra2112"
git add .
git diff --cached --check
git commit -m "Initial release of Multi Section Patch"
gh repo create rudra2112/multi-section-patch --public --source=. --remote=origin
git push -u origin main
```

Then add discoverability metadata:

```text
gh repo edit rudra2112/multi-section-patch --description "Self-contained Agent Skill for exact, failure-safe multi-file reads and edits" --add-topic agent-skills --add-topic coding-agents --add-topic developer-tools --add-topic golang
```

In repository settings, enable private vulnerability reporting and protect
`main` after the required CI checks exist.

## Require native evidence

The CI matrix must pass natively on:

- Linux amd64 and arm64;
- macOS amd64 and arm64;
- Windows amd64 and arm64.

Cross-compilation and the local macOS smoke test are build evidence, not a
substitute for those six native jobs. If any hosted runner is unavailable,
leave the release untagged and describe that platform as unverified.

After CI is green, inspect the reproducibility job. It rebuilds every committed
binary twice, compares both builds with the checked-in files, and verifies
`SHA256SUMS`. The workflow intentionally uploads no temporary Actions artifact,
and `setup-go` caching is disabled because the module has no third-party
dependencies. It therefore consumes neither artifact nor cache storage. If
source changes are needed, regenerate all binaries and repeat the full matrix
before tagging.

## Tag the first release

Run only after all release gates pass:

```text
git tag -a v0.1.0 -m "Multi Section Patch v0.1.0"
git push origin v0.1.0
gh release create v0.1.0 --repo rudra2112/multi-section-patch --verify-tag --title "Multi Section Patch v0.1.0" --generate-notes
```

GitHub automatically provides source archives for the tag. Because the six
binaries are part of the skill directory, those archives are also the manual
installation package with no separately installed language runtime.

## Verify adoption paths

After the tag is public:

```text
gh skill preview rudra2112/multi-section-patch multi-section-patch
npx --yes skills@1.5.19 add rudra2112/multi-section-patch --list
npx --yes skills@1.5.19 add rudra2112/multi-section-patch --skill multi-section-patch --agent "*" --yes
```

In disposable projects, test the documented install and update commands plus
the manual removal procedure for every installer target claimed in the README.
On macOS/Linux, verify the documented GitHub CLI `chmod` fallback. On a
filesystem where installer symlinks are unavailable, verify the documented
`--copy` fallback. Run the installed executable with Python, Node.js, and Go
removed from `PATH`, then repeat the read and edit smoke cases with network
access blocked.

Do not publish an adoption announcement until those checks match the README.
