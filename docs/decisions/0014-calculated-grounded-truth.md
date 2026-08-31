# ADR 0014: Calculate grounded effective truth from authored leaves

- Status: Accepted
- Date: 2026-08-30
- Supersedes: the V1 truth-propagation and defeat-evaluation deferral

## Context

The shared `.arg` model stores three-valued truth tokens but normally permits
authored truth only on premises and counterpoints. Cludia initially preserved
those tokens without evaluating the support graph. Consequently, true premises
connected through an `AND` junctor still appeared as an unknown lemma, and a
supported counterpoint could retain manually assigned truth.

Concludia already distinguishes authored leaf truth from calculated truth. It
uses strong three-valued `AND`/`OR`, disjunction across alternative incoming
justifications, and a grounded counterpoint overlay that activates undermines
and undercuts.

## Decision

- Persist authored `T`, `F`, or `U` only on unsourced premises and unsourced
  counterpoints.
- Calculate effective truth on every read; never serialize propagated values as
  cache state.
- Use strong three-valued logic: false dominates `AND`, true dominates `OR`,
  and unknown otherwise propagates.
- Treat alternative incoming junctors and direct supports as disjunctive
  justifications.
- Use base propagated counterpoint truth to compute grounded `in`, `out`, and
  `undecided` acceptance. Accepted undermines force their premise target false;
  accepted undercuts disable only their exact junctor-to-target edge.
- A sourced statement with no remaining active justification evaluates false.
- A sourced counterpoint normalizes stored truth to `U`; removing its support
  does not restore an earlier authored value.
- Imported sourced statements carrying `T` or `F` remain readable and
  losslessly preserved on read. Evaluation ignores the token, validation warns,
  and the state-bound `normalize-truth` operation performs explicit repair.
- JSON response schema version 2 retains persisted `statement.truth` and adds
  effective truth, truth source, evaluation metadata, and counterpoint
  acceptance to read surfaces. Evaluation results have their own schema version
  1 and declare mode `grounded`.
- The compact `!` marker and `challenged` summary field compare propagated truth
  before defeats with effective truth after grounded defeats. They therefore
  report a material counterpoint effect anywhere upstream, not merely a direct
  attached challenge. A rebutted or outcome-neutral counterpoint does not mark
  the statement. Direct defeats remain available in detail and relation reads.
- This deterministic structural evaluation is not confidence, probability, or
  proof that natural-language entailment is sound.

## Consequences

- Leaf edits produce small authored diffs while every dependent truth is
  immediately recomputed.
- The same file may later support other versioned evaluation overlays without
  rewriting its authored content.
- Human and TUI views prioritize effective truth and show authored truth when it
  differs.
- Scripts can inspect the complete overlay through `evaluate --json`.
- Evaluation failures are treated as invalid read state and never produce a
  partial write.

## Alternatives considered

- Persist propagated truth on every statement: rejected because redundant
  values can become stale, create cascading diffs, and cannot represent
  multiple evaluation modes safely.
- Implement support propagation without defeats: rejected because typed
  counterpoints are first-class and Concludia parity requires their grounded
  effects.
- Rewrite legacy sourced truth while reading: rejected because reads and
  compatibility round trips must not mutate files.
