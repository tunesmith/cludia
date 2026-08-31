# ADR 0003: Persist entailment-style linked inferences, not additive support

- Status: Accepted
- Date: 2026-08-23
- Amended: 2026-08-31 to distinguish evidence islands from hypotheses

## Context

Clues often suggest hypotheses through abduction. Many argument and evidence
systems model independent support, accumulated force, confidence, or
probability. The product intent instead follows Concludia's logical-force
discipline.

## Decision

Focused v1 authoring creates only linked, multi-premise `AND` junctors. Each
junctor asserts that its sources, taken together, are sufficient for its
target.

The product does not create confidence scores, weights, Bayesian estimates, or
relevance-only support.

Abduction may be used by the human or external LLM to invent a candidate
conclusion. Before persistence, hidden bridge premises must be made explicit so
the durable structure is an asserted entailment-style argument.

Disconnected premises are accepted factual or value statements whose relevance
is not yet known. They may remain unused or turn out to be red herrings without
becoming hypotheses. Speculative hypotheses, rival explanations, and
brainstorming remain in conversation or adjacent notes until explicit recorded
premises are intended to establish them. Authored `U` remains available for a
recorded proposition whose truth is genuinely uncertain; it is not the normal
placeholder for a theory that may or may not be true.

V1 preserves legacy direct supports and `OR` constructs found in Concludia
files but does not create them through the focused workflow.

## Consequences

- Missing premises become visible rather than hidden in a weight or suggestion.
- The LLM must be allowed to report that no conclusion follows.
- Natural-language sufficiency cannot be mechanically guaranteed; semantic
  audit remains human/LLM work.
- Some useful non-deductive evidential reasoning will not be expressible as a
  persisted inference until reconstructed with explicit premises.

## Alternatives considered

- Weighted evidence accumulation: rejected as inconsistent with product intent.
- Per-inference deductive/inductive/abductive mode: rejected as UX complexity
  and as a confusion between discovery and durable semantics.
- Single-premise convergent support: not authored in v1.
