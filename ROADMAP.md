# Roadmap

Cludia 1.0 establishes the local, file-first CLI and TUI, grounded truth,
shared mutation architecture, stable IDs, and rooted Concludia export. Future
work is driven by concrete dogfood needs rather than by a fixed feature
schedule.

## Explanation through dogfooding

Existing reads provide Statement Detail, complete and selected Ledgers, rooted
structure, and versioned grounded evaluation. Continue using those surfaces on
large inquiries and capture any truth result they cannot explain clearly.

If a concrete gap appears, design a defeat-inclusive rooted explanation view.
It must state whether it includes inactive counterpoints, every accepted defeat
chain, decisive versus outcome-neutral support, and all or selected true,
false, and unknown routes. It must not silently present one path as the
argument.

## Shared-counterpoint format design

One counterpoint may conceptually challenge multiple explicit locations, but
the portable `.arg` syntax currently gives each counterpoint one target. Any
reuse capability requires a coordinated `.arg` and Concludia decision covering
syntax, target order, removal, rooted export, and older-reader diagnostics.
Cludia will not invent an incompatible local-only attachment operation.

## Interface evolution

- Consider bounded, stale-safe undo for Top `J`/`K` only if durable reorder
  dogfooding demonstrates that it is valuable.
- Add an MCP adapter only when it materially improves external-agent use. It
  must remain a thin interface over the same domain operations and files.
- Build a local web interface only when repeated workflows justify it. It must
  use the shared query and mutation layers and maintain CLI information parity.

## Model and interoperability evolution

- Revisit focused `OR` authoring if multiple complete `AND` justifications are
  insufficient in real inquiries.
- Design structured provenance only after a use case establishes whether
  citations belong in statement metadata, separate objects, or an adjacent
  evidence tool.
- Consider stable cross-system identities only if incremental Cludia/Concludia
  synchronization becomes a concrete requirement.
- Keep `.arg` syntax versioning separate from optional Concludia graph/artifact
  `meta version` metadata.

## Deferred capabilities

- Built-in LLM calls, embeddings, and autonomous background discovery.
- Collaboration, accounts, and hosted argument publication.
- Probabilities, confidence scores, evidence weights, or Bayesian machinery.
- Native application packaging, graphical maps, timelines, entity extraction,
  and general investigation ingestion.
- A universal format spanning Cludia and Dagim.
