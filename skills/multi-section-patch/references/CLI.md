# Multi Section Patch CLI reference

Multi Section Patch reads exact sections from UTF-8 text files and previews or
applies guarded section replacements. In every example,
`"<multi-section-patch>"` is the quoted absolute path to the bundled executable
selected in `SKILL.md`.

## Read command

```text
"<multi-section-patch>" read [--spec FILE] [--context N] [--json] [--no-line-numbers] [SELECTOR...]
```

Selectors use these forms:

| Form | Meaning |
| --- | --- |
| `FILE` | The complete file |
| `FILE@START:END` | Inclusive, one-based line range |
| `FILE@START..END` | Start at a literal marker and stop before a literal end marker |
| `FILE@/START/../END/` | Start and end with Go/RE2 regular expressions |
| `FILE@START` | Start at a literal marker and continue to end of file |

Either endpoint of a numeric range may be omitted. Use a JSON specification
when a file name or marker would make the compact `@` or `..` syntax
ambiguous. A literal marker matches when it occurs anywhere on a line. Regex
matching uses Go's RE2 syntax and is evaluated one line at a time.

Read options:

| Option | Behavior |
| --- | --- |
| `--spec FILE` | Read a JSON specification from `FILE`; use `-` for standard input |
| `--context N` | Show `N` surrounding lines in human output |
| `--json` | Emit machine-readable success JSON on stdout and failure JSON on stderr |
| `--no-line-numbers` | Omit line prefixes from human output |
| `--` | Treat every remaining argument as a selector |

With no selectors and no named specification file, Multi Section Patch reads
JSON from standard input. It resolves every requested section before emitting
output, so a later invalid section cannot leave a partial result.

The recommended JSON shape is:

```json
{
  "sections": [
    {
      "name": "optional display name",
      "file": "README.md",
      "start_line": 10,
      "end_line": 20
    },
    {
      "file": "CONTRIBUTING.md",
      "start": "## Development setup",
      "end": "## Code and documentation",
      "include_start": true,
      "include_end": false,
      "occurrence": 1,
      "end_occurrence": 1
    },
    {
      "file": "src/main.go",
      "start_regex": "^func main",
      "end_regex": "^}"
    }
  ]
}
```

A top-level list containing the same items is also accepted. Use exactly one
selector family in each item: line fields, literal marker fields, or regex
marker fields. Read items reject edit-only fields such as `replacement`,
`replacement_file`, `expected_sha256`, and `must_contain`. `occurrence`
requires a start marker, while `end_occurrence` requires an end marker; both
values are one-based. `include_start` and `include_end` follow the same
corresponding-bound rule. Marker defaults include the start line and exclude
the end line.

Each result contains the canonical path, name, resolved one-based range,
SHA-256 of the exact selected bytes, and content. An empty whole file is
reported as range `1:0`. Human output uses
`<<<MULTI_SECTION_PATCH ...>>>` and `<<<END_MULTI_SECTION_PATCH ...>>>`
boundaries. A selected line beginning with either boundary is escaped so file
content cannot forge the output structure.

## Edit command

```text
"<multi-section-patch>" edit [--spec FILE] [--json] [--backup] [--apply --expect-plan SHA256]
```

Edit is always a dry run unless `--apply` is present. The dry run resolves all
edits, prints the complete proposed diff, and returns an opaque `plan_sha256`
without changing target files. Apply requires that exact identifier.

The recommended JSON shape is:

```json
{
  "edits": [
    {
      "file": "README.md",
      "start": "## Old section",
      "end": "## Next section",
      "replacement": "## Old section\nReplacement text.\n",
      "expected_sha256": "optional lowercase SHA-256",
      "must_contain": ["required text"]
    },
    {
      "file": "src/main.go",
      "start_line": 20,
      "end_line": 35,
      "replacement_file": "replacement.txt"
    }
  ]
}
```

Edit fields:

| Field | Meaning |
| --- | --- |
| `file` | Required target path |
| `name` | Optional diagnostic name |
| `start_line`, `end_line` | Inclusive, one-based line bounds |
| `start`, `end` | Literal marker bounds |
| `start_regex`, `end_regex` | Go/RE2 marker bounds |
| `include_start` | Replace the start marker line; defaults to `true` |
| `include_end` | Replace the end marker line; defaults to `false` |
| `occurrence`, `end_occurrence` | One-based marker occurrences |
| `replacement` | Inline UTF-8 replacement text |
| `replacement_file` | Non-empty path to UTF-8 replacement text |
| `expected_sha256` | Required non-empty digest of the selected current bytes |
| `must_contain` | Required non-empty string or non-empty list of non-empty strings |

Specify exactly one of `replacement` and `replacement_file`.
`replacement_file` and `expected_sha256` cannot be empty when present. The
same selector rules as `read` apply. Overlapping edits are rejected; adjacent
edits are allowed. Replacements adopt the target's LF or CRLF style, while
unrelated bytes and the target's final-newline state remain unchanged.

Edit options:

| Option | Behavior |
| --- | --- |
| `--spec FILE` | Read JSON from `FILE`; use `-` or omit it for standard input |
| `--json` | Emit structured success JSON on stdout and failure JSON on stderr |
| `--backup` | With `--apply`, retain independent originals and a manifest |
| `--apply` | Write only when accompanied by the reviewed plan identifier |
| `--expect-plan SHA256` | Require the rebuilt plan to match this dry-run identifier |

Dry-run and successful-apply JSON use this shape:

```json
{
  "diffs": ["complete unified diff"],
  "changed_files": 1,
  "plan_sha256": "64-character lowercase SHA-256",
  "applied": false
}
```

Successful apply output sets `applied` to `true`; with `--backup`, it also
includes `backup_directory`.

The plan identifier binds the ordered canonical targets, file identities,
permission modes, complete original and final bytes, and resolved edits. On
apply, Multi Section Patch rebuilds and compares it before diff output, staging,
backup creation, or writes. A mismatch exposes no replacement plan; run and
review a new dry run.

After the plan comparison, Multi Section Patch stages replacement and recovery
files beside each target and rechecks every target. It rechecks a target again
immediately before its replacement. A multi-file batch is not
filesystem-atomic; after a later failure, Multi Section Patch restores completed
replacements when that cannot overwrite a concurrent change. Any incomplete
rollback or cleanup reports the exact retained recovery path.

## Input and platform limits

- Target and replacement files must be regular, valid UTF-8 text without NUL
  or unsupported binary control bytes.
- Normal reads and edits perform no network request and emit no telemetry.
- Unix permission bits and the Windows read-only state exposed by Go are
  preserved. Ownership, ACLs, extended attributes, resource forks, alternate
  data streams, and timestamps are outside the portable guarantee.
- A Windows target marked read-only is rejected before staging. Remove that
  attribute deliberately before applying an edit, then restore it if needed.
- The current release matrix is Windows, macOS, and Linux on x86-64 and ARM64.

## Exit behavior

- Exit `0`: the read, dry run, no-op, or apply completed.
- Exit `1`: input, validation, file, staging, replacement, rollback, cleanup,
  or output failed.
- Exit `2`: the command is missing or unknown.

Errors are concise and contain no stack trace. Without `--json`, stderr keeps
the human `multi-section-patch: error:` form. With `--json`, a recognized
command emits this envelope on stderr:

```json
{
  "error": {
    "code": "invalid_spec",
    "command": "read",
    "message": "diagnostic text",
    "item_index": 1,
    "name": "optional item name",
    "file": "optional affected path",
    "field": "optional affected field"
  }
}
```

The four context fields are omitted when unknown, and `item_index` is
one-based. Stable error codes are `invalid_option`, `spec_read_failed`,
`invalid_spec`, `section_resolution_failed`, `edit_plan_failed`,
`plan_mismatch`, `apply_failed`, `output_failed`, and `operation_failed`.

Treat a nonzero apply result as requiring review even when the message says
targets were rolled back, and keep any reported recovery file until its target
is verified. An output failure can occur after a successful apply, so exit `1`
does not prove that targets are unchanged.
