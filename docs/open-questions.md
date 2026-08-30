# Open Questions

This file records choices intentionally left unresolved. They should not be
answered accidentally through implementation details.

## Naming

- Chosen: the permanent project and repository name is `Cludia` (`cludia` in
  filesystem and repository contexts).
- Chosen: the command-line binary name is `cludia`.
- Workspace profile identifier.

The `profile="workspace"` identifier remains provisional. It identifies
validation semantics, not the product, so it need not match the binary name.

## Format ownership and versioning

- Does Concludia remain the owner of the common `.arg` specification, or should
  a stable shared format eventually live in a neutral package/repository?
- How is the syntax version declared independently from the content artifact's
  current `meta version`?
- Should a shared conformance corpus be copied, vendored, or released as a
  separate module?

The v1 implementation should avoid forcing these decisions while maintaining
explicit compatibility fixtures.

## Future strict treatment of direct supports

Chosen v1 behavior:

- read and preserve current one-source/direct support;
- do not create it in focused authoring.

Deferred Concludia question:

- Should a future strict profile reject direct support entirely?

Any removal requires Concludia migration and compatibility analysis and is not
a Cludia v1 task.

## `OR` authoring

V1 reads and preserves `OR` but does not need to create it. Dogfooding should
determine whether users need focused `OR` creation or whether multiple complete
`AND` justifications into one target are sufficient.

## Structured provenance

V1 has no source-document, citation, or passage model. Before adding one,
dogfooding should determine whether source information belongs:

- in statement text;
- as optional per-statement metadata;
- as distinct source and citation objects;
- in an adjacent evidence-management tool.

If added, source truth and statement truth must not be conflated.

## Built-in generative assistance

V1 uses an external LLM. Possible future product-owned capabilities include
retrieval, embeddings, candidate combination discovery, missing-premise audits,
and background suggestions.

Before crossing that boundary, decide:

- what deterministic value cannot be provided by an external agent;
- privacy and local-model requirements;
- provider and credential policy;
- whether generated output remains a proposal queue or becomes a command;
- reproducibility and cost expectations.

## Web technology

The web UI is deliberately unspecified until CLI dogfooding. It should be local
first and share the domain core. Possible implementations include a local Go
server with a browser UI or a separately built frontend served by the Go
binary.

The eventual choice should optimize semantic parity and iteration speed rather
than native packaging.

## Defeat evaluation

Resolved by [ADR 0014](decisions/0014-calculated-grounded-truth.md): Cludia
calculates Concludia-compatible grounded effective truth locally while keeping
propagated values out of the `.arg` artifact.

## Time, revision, and statement identity

[ADR 0006](decisions/0006-time-revision-and-adjacent-artifacts.md) establishes
keeping revision history outside the `.arg` model in v1, using Git for tracked
file revisions and explicit wording when valid time matters.

[ADR 0007](decisions/0007-statement-identity-and-revision-semantics.md)
establishes the distinction between durable proposition identity, mutable text,
and optional slugs, with explicit caller intent for meaning-preserving text
edits.

Dogfooding must still determine:

- when an edit is a correction of one stable statement versus a materially new
  proposition that should receive a new ID;
- whether named snapshots need conventions beyond ordinary files and Git refs;
- which concrete temporal or provenance query, if any, justifies structured
  metadata in the shared format.

ADR 0007's proposed 2026-08-25 refinement resolves the single-workspace
replacement command, explicit per-relation selection, old-record retention, and
state-bound plan/application workflow. Replacement coordination across multiple
known files remains open.

[ADR 0009](decisions/0009-monotonic-identifiers-and-renumbering.md) resolves
identifier lifetime after deletion for focused authoring: compact next-ID
metadata prevents ordinary reuse, and an explicit state-bound whole-document
renumber is the sole reset. A separate opaque identity remains deferred until a
concrete bidirectional or incremental Concludia synchronization use case
requires it.

## License and distribution

The module path is `github.com/tunesmith/cludia`, and the current private
distribution method is cloning and installing from source. The license and
public release process remain deferred and must be chosen before the first
public release.
