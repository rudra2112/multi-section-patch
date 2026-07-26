# Specification Contract

A specification defines what must be true and why. It must not prescribe
implementation details unless they are externally observable constraints.

## When to write one

Write a durable specification for a new feature, public behavior change,
compatibility change, security-sensitive change, or ambiguous multi-file task.
Skip it for a trivial, well-understood correction.

Store each change as `docs/specs/<short-change-name>.md`. Use plain Markdown
without vendor frontmatter, model names, slash commands, or machine-specific
paths.

## Required qualities

- Name the affected users, problem, and observable outcome.
- Separate in-scope behavior from explicit non-goals.
- Give every requirement a stable ID and a MUST, SHOULD, or MAY level.
- Express acceptance as observable Given/When/Then scenarios.
- Cover invalid input, boundaries, interruption, concurrency, and recovery
  where relevant.
- State supported platforms, compatibility limits, security properties,
  performance constraints, assumptions, and measurable success.
- Mark unresolved behavior or scope questions explicitly. Do not start a
  materially divergent or irreversible implementation while one is blocking.
- Keep the document concise. Delete unused prompts instead of leaving
  placeholders.

## Lifecycle

Use `Draft`, `Accepted`, or `Superseded`.

- Draft: under review; it does not override current behavior.
- Accepted: authoritative for its scope.
- Superseded: retained for history and linked to its replacement.

If intended behavior changes, update and re-accept the specification before
changing the implementation plan.

## Template

```md
# <Feature or change>

Status: Draft

## Problem and outcome

<Who is affected, what problem exists, and what becomes observable?>

## Users

- <User or operator>

## Scope

### In scope

- <Behavior>

### Out of scope

- <Non-goal>

## Requirements

- REQ-001 (MUST): <Testable requirement>.

## Scenarios and acceptance

- AC-001: Given <state>, when <action>, then <observable result>.

## Edge and failure cases

- <Boundary, invalid input, interruption, concurrency, or recovery case>

## Success measures

- <Measurable evidence>

## Assumptions

- <Explicit assumption>

## Open questions

None, or list each question and whether it blocks implementation.
```

Before acceptance, verify that every MUST requirement has acceptance coverage,
no sections contradict each other, and no blocking question remains.

## Guidance basis

This contract is an original, agent-neutral synthesis of
[GitHub Spec Kit](https://github.github.com/spec-kit/) and
[Kiro specifications](https://kiro.dev/docs/cli/v3/specs/). Those sources inform
the structure; neither tool is required to use this repository.
