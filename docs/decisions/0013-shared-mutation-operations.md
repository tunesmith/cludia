# ADR 0013: Put durable mutation semantics in shared operations

- Status: Accepted
- Date: 2026-08-30

## Context

Cludia began as a CLI, and several early commands cloned documents and edited
their public slices directly before validating and saving. That made command
files the effective application layer even though the specification requires
CLI, TUI, and future web interfaces to share durable behavior.

Presentation-level reuse is not enough. A future interface must receive the
same target resolution, invariant failures, allocator changes, relation
rewrites, blockers, mappings, and dry-run plans as the CLI without duplicating
command implementation details.

## Decision

- Every durable semantic mutation is implemented as a shared operation over the
  argument model.
- Operations accept semantic inputs, clone the caller's document, and return a
  proposed document plus typed result facts or stable typed failures.
- Domain operations do not parse flags, render human or JSON output, or perform
  file I/O.
- Statement initialization, capture, batch capture, editing, slug mutation,
  derivation, defeat creation, junctor repair/removal, counterpoint removal,
  statement deletion, material replacement, renumbering, and statement order
  movement all follow this contract.
- Replacement and renumbering compute their state-bound opaque plan tokens and
  complete proposed effects in the shared operation. Dry-run and apply invoke
  the same planner; the interface only compares the reviewed token and decides
  whether to persist.
- A shared workspace application layer owns profile selection, parse/load
  validation, complete-result validation, and validate-before-atomic-create or
  save orchestration.
- Interface code may compute presentation-only summaries such as component
  counts, newly isolated statements, diagnostic wording, and JSON change lists
  from the shared result. It must not edit document fields or relation slices.
- The TUI's focused reorder uses the same statement-order operation and shared
  persistence orchestration as the CLI while retaining its additional stale
  adjacency and disk-version checks.

## Consequences

- CLI command tests primarily cover flags, public response contracts, and
  presentation; domain tests cover success, no-op behavior where applicable,
  invariant failures, clone isolation, mappings, and state binding.
- Invalid and dry-run operations leave the caller and workspace unchanged.
- A future web or MCP layer can call the same operations without importing CLI
  packages.
- Domain result types contain semantic facts rather than presentation-ready
  strings, although public data structures may carry JSON field tags where they
  are themselves part of an existing response contract.
- The workspace application layer depends on parsing, validation, and atomic
  persistence; the argument domain remains independent of those layers.

## Alternatives considered

- Keep mutations in CLI files and copy them into later interfaces: rejected
  because behavior and failure contracts would drift.
- Build one generic graph mutation framework shared with Dagim: rejected because
  prerequisite/completion semantics and authored inference/defeat semantics are
  materially different.
- Put validation and file I/O inside every domain operation: rejected because it
  would couple semantic transformations to one persistence and profile context.
