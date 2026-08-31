# Remaining Suggestions from the Cludia–Dagim Review

This file tracks only unfinished work from the original cross-project review.
Completed recommendations are removed rather than retained as project history.

## Recoverable durable TUI reorder

Top `J`/`K` is currently the TUI's focused durable mutation. It checks displayed
adjacency against current disk state, refuses stale moves, validates the result,
saves atomically, and reloads the document.

A future bounded undo could make that gesture easier to recover from if
dogfooding demonstrates a need. Undo must itself be a normal validated durable
reorder; it must not restore an old in-memory document or overwrite intervening
CLI, agent, or external edits.

### Required behavior

- Record enough state after a successful TUI reorder to propose its inverse.
- Reload and verify expected current adjacency and identities before undo.
- Apply the inverse through the shared statement-order operation and atomic
  persistence path.
- Refuse a stale inverse, retain the refreshed workspace, and ask the user to
  review its current order.
- Keep history bounded; persistent cross-session history is not required
  without further dogfood evidence.

### Acceptance criteria

- A fresh successful reorder can be undone without changing statement content,
  IDs, relations, or allocator state.
- External edits between reorder and undo cannot be overwritten.
- Failed or stale undo leaves the file unchanged.
- CLI and TUI continue to share the same durable reorder operation.
