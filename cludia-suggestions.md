# Remaining Suggestions from the Cludia–Dagim Review

## Context

This file tracks only unfinished work from the original cross-project review.
Completed recommendations are removed rather than retained as project history.

## Remaining work

| Suggestion | Disposition |
|---|---|
| Recoverable durable TUI reorder | Deferred until the mutable TUI gesture warrants undo history |
| Publication and release discipline | Required before public distribution |

## Recoverable durable TUI reorder

Top `J`/`K` is currently the TUI's focused durable mutation. It already checks
displayed adjacency against current disk state, refuses stale moves, validates
the result, saves atomically, and reloads the document.

A future bounded undo could make that gesture easier to recover from. Undo must
itself be a normal validated durable reorder. It must not restore an old
in-memory document or overwrite intervening CLI, agent, or external edits.

### Required behavior

- Record enough state after a successful TUI reorder to propose its inverse.
- Before applying undo, reload and verify that the inverse still describes the
  expected current adjacency and statement identities.
- Apply the inverse through the same shared statement-order operation and
  atomic persistence path as every other reorder.
- If the workspace changed incompatibly, refuse undo, retain the refreshed
  workspace, and ask the user to review its current order.
- Keep the history bounded. Persistent cross-session history is not required
  unless later dogfooding demonstrates a need.

### Acceptance criteria

- A fresh successful reorder can be undone without changing statement content,
  IDs, relations, or allocator state.
- External edits between reorder and undo cannot be overwritten.
- Failed or stale undo leaves the file unchanged.
- CLI and TUI continue to share the same durable reorder operation.

## Publication and release discipline

Cludia remains a single-user pre-release tool, so release machinery should not
slow current semantic and workflow iteration. Before public distribution,
however, the project should define and automate the following.

### Product and compatibility contract

- Version injection and useful `cludia version` output.
- A concise compatibility statement distinguishing durable `.arg` file
  compatibility from replaceable pre-release CLI/JSON request contracts.
- Required Concludia, workspace, OR, direct-support, defeat-chain, cycle,
  atomic-write, truth-evaluation, and JSON/process fixtures.
- A changelog or equivalent release-note practice once external users exist.

### Repository and distribution

- An explicit open-source license and contribution policy.
- Public GitHub repository/readme preparation and a clean-tree release
  checklist.
- Reproducible tagged builds for supported platforms.
- Homebrew packaging only after the binary name, install path, version output,
  and release artifact layout are stable.
- A preflight that runs tests, vet, command builds, compatibility fixtures, and
  process-contract checks before publishing artifacts.

## Priority

1. Continue dogfooding immediate TUI reorder before deciding whether bounded
   undo is worth its state-management complexity.
2. Complete licensing, GitHub, versioning, release artifacts, and Homebrew work
   as one coordinated publication project when Cludia is ready for users beyond
   its current owner.
