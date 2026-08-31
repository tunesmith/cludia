# ADR 0004: Include undermines, undercuts, and recursive counterpoints in v1

- Status: Accepted
- Date: 2026-08-23
- Amended: 2026-08-24 (retention of contested junctors)
- Amended: 2026-08-31 (grounded, exact-scope authoring guidance)
- Partial supersession: ADR 0014 adds calculated grounded acceptance and
  effective truth while retaining the prohibition on numerical strength.

## Context

Inquiry produces material reasons to doubt both statements and attempted
inferences. Omitting challenges from the first durable model would either lose
investigative history or force challenges into ordinary support structures.

Concludia's `.arg` format already distinguishes premise undermines, inference
undercuts, and counterpoints targeting counterpoints.

## Decision

V1 stores, queries, creates, removes, traverses, and exports all three defeat
scopes.

User-facing language will say "counterpoint of a counterpoint." There is no
special rejoinder type. A legacy `rejoinder` parser alias may be accepted and
normalized for compatibility.

Defeat relations count when discovering components. Directed cycles involving
support or defeat relations are rejected, and all traversal is cycle-safe.

ADR 0014 adds deterministic grounded acceptance over these defeat forms. No
numerical strength model is introduced.

A focused counterpoint should identify a case-specific, record-grounded defect
in the exact premise, inference, or earlier counterpoint it targets. Bare
possibility, unsupported rival explanation, generic fallibility, absence of
direct proof, and residual uncertainty do not by themselves justify a defeat.
This is semantic authoring guidance rather than a lexical validator.

An undercut does not retract or mutate its target junctor. A living workspace
may retain the challenged junctor while it is evaluated, after it is presently
rejected, or alongside a repaired replacement. This coexistence is
structurally valid and does not by itself block saving or rooted export.

When an undercut reveals a missing premise or obsolete inference, explicit
repair or removal remains preferred hygiene. The tool preserves authored and
challenging claims until the user performs that mutation; it does not silently
adjudicate or clean up the reasoning.

## Consequences

- Failed and contested reasoning can be preserved and revisited.
- Rooted export can carry contestation into Concludia.
- Mutation and deletion planning must account for recursive counterpoint
  chains.
- Contested and repaired junctors may coexist, so inspection should make their
  defeat state clear and removal must remain explicit.
- A counterpoint must not be presented as repairing a sufficiency claim; adding
  a missing premise, replacing the junctor, or removing it remains preferred.

## Alternatives considered

- Defer all defeats: simpler implementation, but weakens investigation and
  `.arg` compatibility.
- Store only free-form notes: loses typed challenge targets.
- Model a special rejoinder role: rejected because recursive counterpoints are
  sufficient.
