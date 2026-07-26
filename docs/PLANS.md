# Implementation Plan Contract

A plan defines how an accepted specification will be delivered. It is a
self-contained, living record that another person or coding agent can resume
using only repository content.

## When to write one

Write a durable plan for multi-step, multi-file, risky, ambiguous, or
long-running work. Skip it for a trivial change whose implementation and
validation are already obvious.

Store each plan as `docs/plans/<short-change-name>.md`, paired by filename with
its specification. Use repository-relative paths and ordinary commands that
identify their working directory.

## Required qualities

- Link the accepted specification and state the user-visible purpose.
- Explain relevant current behavior, constraints, terminology, and files.
- Record the chosen approach and materially different alternatives rejected.
- Divide work into independently verifiable milestones.
- For each milestone, name the outcome, affected components, dependencies, and
  exact proof.
- Describe validation, expected results, retries, rollback, and interruption
  recovery.
- Keep Progress, Discoveries, Decisions, and Outcome current at every
  meaningful pause.
- Keep tasks inside the plan by default. Add `tasks.md` only when a genuinely
  large or parallel queue would otherwise make the plan hard to use.
- Never claim completion from stale or indirect evidence.

## Lifecycle

Use `Draft`, `Active`, `Blocked`, or `Completed`.

- Draft: design is still being reviewed.
- Active: its related specification is accepted and implementation is underway.
- Blocked: progress requires a named decision or external change.
- Completed: acceptance evidence is recorded and remaining gaps are explicit.

## Template

```md
# <Action-oriented outcome>

Status: Draft

Related specification: `docs/specs/<change>.md`

This is a living plan. Keep Progress, Discoveries, Decisions, and Outcome
current whenever work pauses or reality changes.

## Purpose

<What becomes possible and how a person can observe it?>

## Context

<Current behavior, relevant repository paths, constraints, and terms>

## Approach

<Chosen design and material alternatives rejected>

## Milestones

### M1 — <Independently verifiable outcome>

Changes:

- <Component and intended result>

Dependencies:

- <Prerequisite or safe parallel work>

Proof:

- <Exact behavior or command>

## Validation and acceptance

<Commands, working directory, expected results, and acceptance mapping>

## Safety and recovery

<Idempotence, retries, backups, rollback, and interruption handling>

## Progress

- [ ] <Concrete task and expected result>

## Discoveries

- <Unexpected fact with concise evidence>

## Decisions

- Decision: <Choice>.
  Rationale: <Why>.
  Date: <YYYY-MM-DD>.

## Outcome

<Validated behavior, remaining gaps, and lessons>
```

When reality changes, revise the affected plan sections so the document remains
coherent; do not append corrections that leave stale instructions in place.

## Guidance basis

This contract is an original, agent-neutral synthesis of
[OpenAI ExecPlans](https://github.com/openai/openai-agents-python/blob/main/PLANS.md)
and [GitHub Spec Kit](https://github.github.com/spec-kit/). Those sources inform
the structure; neither tool is required to use this repository.
