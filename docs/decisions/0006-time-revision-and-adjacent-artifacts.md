# ADR 0006: Keep revision history outside the `.arg` model in v1

- Status: Accepted
- Date: 2026-08-24

## Context

A living inquiry must distinguish at least three questions:

- What does the current document say happened or follow?
- When was a proposition true in the modeled world?
- What did the document say in an earlier revision, and how did it change?

Incident analysis makes the distinction concrete. A current best account of an
outage may be corrected as evidence improves. A discovery chronology may also
be revised. Remediation work has dependency structure rather than entailment
structure. Forensic reconstruction of prior reasoning may itself become a
separate subject of inquiry.

Git already preserves committed revisions of tracked files, while the shared
Concludia `.arg` syntax has no temporal or provenance objects. Adding recorded
and valid-time fields now would extend the public format before dogfooding has
shown which temporal queries are actually needed.

The connected argument supporting this decision is
[0006-time-revision-and-adjacent-artifacts.arg](0006-time-revision-and-adjacent-artifacts.arg).

## Decision

For v1:

- A `.arg` document represents the best reasoning expressed by that particular
  file revision; it is not an embedded revision log or event store.
- Git remains the revision-history mechanism for tracked project artifacts.
- When the period in which a proposition was true matters, statement text
  expresses that temporal scope explicitly.
- Primarily chronological narratives remain in timeline-appropriate documents,
  and dependency-ordered work remains in Dagim or another task system.
- Cludia does not add per-statement recorded-time, valid-time, supersession, or
  append-only event semantics in v1.
- A later decision may add structured temporal or provenance data when
  dogfooding identifies a concrete query that Git, explicit wording, snapshots,
  and adjacent artifacts cannot answer adequately.

This decision does not determine when a semantic rewrite should retain a
statement ID rather than create a new statement. That identity policy remains a
separate question to resolve through editing dogfood.

## Consequences

- The shared `.arg` syntax remains compatible with Concludia.
- Ordinary corrections can revise the current model without forcing historical
  claims into the active graph solely for audit purposes.
- Git history is useful for tracked work but is not required for every private
  local workspace.
- Commit time does not stand in for the time a proposition was true; durable
  statements should avoid unscoped words such as “now,” “currently,” and “yet”
  when that distinction matters.
- Investigation chronology, current causal explanation, remediation planning,
  and reconstruction of prior reasoning may be separate but cross-referenced
  artifacts.
- Historical rationale snapshots can be ordinary immutable `.arg` exports
  without requiring a new temporal graph model.

## Alternatives considered

- Add bitemporal fields to every statement now: potentially queryable, but
  premature and duplicative of part of Git's role.
- Keep every superseded belief in one active workspace: preserves history in a
  single file, but mixes current reasoning with revision audit and can make the
  graph harder to interpret.
- Require multiple fixed document modes: may eventually prove useful as views,
  but no exhaustive Cludia equivalent of Diátaxis has yet been established.
- Treat Git as the complete temporal model: rejected because revision time does
  not express when a proposition was true in the modeled world.
