# ADR 0010: Use one durable statement order with proof-local ledgers

- Status: Accepted
- Date: 2026-08-28
- Supersedes: ADR 0008's strict read-only TUI boundary

## Context

Dogfooding the initial navigator made Top useful not only as a computed set of
support sinks but also as a human recovery surface: its order communicates
which open findings or conclusions should be encountered first. That order was
already the document's statement order, which also supplies deterministic order
to search, components, rooted export, renumbering, and Ledger tie-breaking.

A separate Top preference would create two competing orders and require new
workspace-only metadata. Reordering the existing statement sequence instead is
durable in the shared `.arg` syntax and preserves a readable file.

The first Ledger scheduler used document order whenever more than one statement
was topologically available. Because every unsupported premise is available at
the beginning, it could introduce a premise long before the derived statement
with which that premise is first combined. The result was valid but less like a
goal-directed proof narrative.

## Decision

- Statement sequence is the workspace's one durable general order. No Top- or
  Ledger-specific order metadata is added.
- `move-statement` moves one statement immediately before or after another,
  validates the complete document, and saves atomically. It changes no ID,
  relation, content, or allocator state.
- Top exposes this shared operation through capital `J` and `K`. Lowercase
  `j`/`k` remain navigation, the selected statement follows its move, and
  boundary moves do nothing.
- A Top move reloads and validates the latest file before writing. If the two
  displayed Top statements are no longer adjacent, the move is refused and the
  refreshed order is shown for review.
- This is a narrow exception to ADR 0008's initial read-only boundary. General
  TUI authoring, undo history, and a renumber hotkey remain deferred.
- Ledger uses a reverse topological schedule over the complete upstream support
  closure. Among reverse-ready statements it prefers lower support depth, then
  reverse document order; reversing the result yields a dependency-valid,
  proof-local display with document order as the stable preference among
  equivalent choices.
- Manual Ledger reordering is not added. Any future control must change the
  same general statement order, while proof dependencies continue to override
  impossible display positions.

## Consequences

- Top order survives restart and remains visible to scripts through the normal
  document-ordered query surfaces.
- Moving a statement can affect a later reviewed `renumber` mapping, including
  global junctor numbering when target blocks and their attached support clauses
  move. Reordering itself never changes identity.
- Rooted export preserves the general order of included statements while role
  reconciliation and structural validation remain unchanged.
- Ledger JSON retains its schema and complete rows, but the deterministic row
  order becomes more proof-local.
- External edits can race any local file mutation, but stale Top intent is not
  silently applied to a changed visible ordering and atomic save still prevents
  partial files.

## Alternatives considered

- Store a separate Top order in metadata: rejected because it creates competing
  orders and weakens CLI, export, and future-interface parity.
- Reorder only the in-memory TUI list: rejected because the result disappears
  on restart and external reload.
- Apply document order as the only Ledger priority: dependency-safe but
  front-loads premises that are not yet relevant.
- Add manual Ledger keys immediately: deferred until the shared-order behavior
  is dogfooded.
