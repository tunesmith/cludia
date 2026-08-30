# ADR 0011: Keep focused statement identifiers consistent with current roles

- Status: Accepted
- Date: 2026-08-30
- Supersedes in part: ADR 0007's claim that every same-proposition transition
  preserves the statement ID
- Supersedes in part: ADR 0009's claim that only whole-document renumbering may
  change an ID during ordinary focused work

## Context

Focused Cludia identifiers use role-bearing namespaces: `P` for premise, `L`
for lemma, `C` for conclusion, and `CP` for counterpoint. The original derive
operation preserved the ID when an existing premise became the target of a
justification. That produced declarations such as `lemma P182`.

The proposition record and slug remained continuous, and internal relations
remained valid, but the visible ID contradicted the statement's current role.
This was especially misleading in Top and Ledger views, which prominently show
the ID. Agent workflows that batch-captured candidate hypotheses and then
derived into them could produce an entire workspace of `lemma P...` records
without ever allocating an `L` ID.

The shared `.arg` syntax has no opaque identity beneath its file-local labels.
Keeping a role-bearing label unchanged across a role transition therefore
cannot preserve both literal ID continuity and truthful role notation.

## Decision

- Focused canonical statement IDs describe the statement's current role.
- When a focused derive operation first promotes an existing premise to a
  lemma, it assigns the exact next monotonic `L` ID.
- The old `P` ID is retired and is not reused by ordinary allocation.
- The operation atomically rewrites every modeled reference in the workspace,
  including junctor sources and targets, direct supports, defeats, and
  recognized root metadata.
- The mutation result reports the complete previous-to-current ID mapping, the
  role change, and any recognized metadata rewrite.
- Because references outside the workspace cannot be rewritten, promotion
  emits a structured external-reference warning.
- Failed promotion consumes neither the `L` ID nor the junctor ID and leaves the
  caller's document and file unchanged.
- Text, truth, kind, and slug edits that do not change role continue to preserve
  the statement ID under ADR 0007.
- `renumber` remains the only numbering reset and compaction operation. A
  role-driven `P`-to-`L` change is a monotonic reidentification, not a reset.
- Existing Concludia files, custom IDs, and legacy Cludia files with
  role-mismatched canonical IDs remain readable and losslessly preserved.
  The reviewed whole-document `renumber` operation is the migration path when a
  user wants those IDs reconciled.

## Consequences

- A focused promotion no longer leaves a lemma with a `P` prefix.
- Conversations or adjacent artifacts that cited the retired `P` ID may require
  review after promotion; the mutation output supplies the mapping.
- Batch-capturing a future target and then deriving into it is still supported,
  but callers must consume the promoted `L` ID from the derive result.
- Slugs provide conversational continuity across promotion but remain mutable
  aliases rather than durable identity.
- Legacy and imported files can temporarily contain role/ID mismatches without
  becoming unreadable merely because Cludia adopted the stronger focused-write
  invariant.

## Alternatives considered

- Preserve the `P` ID and display the role separately: structurally safe, but
  leaves the public role-bearing label misleading.
- Change the ID silently: rejected because internal and external references are
  public interfaces and every rewrite must be reported.
- Add a new opaque identity field beneath `.arg` labels: potentially useful for
  future synchronization, but unnecessary for this focused role transition and
  incompatible with the current shared portable syntax.
- Require a separate reviewed renumber after every promotion: preserves the old
  derive behavior but makes the ordinary premise-to-lemma workflow cumbersome
  and allows misleading IDs to accumulate.
