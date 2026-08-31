# ADR 0009: Use monotonic identifiers with explicit whole-document renumbering

- Status: Accepted
- Date: 2026-08-28
- Partial supersession: ADR 0011 permits monotonic role-driven reidentification
  when a focused derivation promotes a premise to a lemma.

## Context

ADR 0007 made statement IDs durable proposition-record identities during
ordinary editing but deliberately left their lifetime after deletion open. In
the tracked dogfood lineage, automatic allocation later reused deleted junctor
IDs. The resulting file remained structurally valid, but references such as
“J2” in conversations, commits, reports, or adjacent files could identify
different inferences at different revisions.

Keeping a catalogue of every deleted number is unnecessary when focused
authoring uses canonical numeric IDs. A compact next-ID record can prevent
ordinary reuse while allowing deletion gaps. Users may still want a tidy file,
but filling one gap at a time would reintroduce the ambiguity the policy is
intended to prevent.

Concludia provides relevant but non-identical precedent. Its server stores
opaque step and junctor identities separately from display labels, while its
portable `.arg` DSL substitutes those labels into declaration and relation
positions. Cludia models that portable file layer and currently has no separate
opaque identity beneath an `.arg` ID.

The connected argument supporting this decision is
[0009-monotonic-identifiers-and-renumbering.arg](0009-monotonic-identifiers-and-renumbering.arg).

## Decision

### Ordinary allocation

- Focused authoring creates canonical `P`, `L`, `C`, `CP`, and `J` IDs with a
  positive decimal suffix and no leading zeroes.
- Each namespace advances independently and monotonically.
- The document records the exact next number for every namespace in one
  versioned metadata value:

  ```text
  cludia-next-ids="v1;P=73;L=37;C=1;CP=9;J=36"
  ```

- An omitted ID receives the recorded next value. An explicit ID must equal
  that same value and use the role-appropriate prefix.
- Successful creation advances the affected value. Failed mutations and dry
  runs consume nothing.
- Focused premise-to-lemma promotion allocates the exact next `L` ID, retires
  the prior `P` ID, and reports the mapping under ADR 0011.
- Deletion never lowers a next value, so gaps remain and deleted numbers cannot
  be requested individually.
- Existing custom or noncanonical IDs remain readable and are preserved by
  ordinary round trips. Focused creation no longer authors them.
- A legacy document without allocator metadata bootstraps each next value from
  the greatest matching canonical ID currently present when its first
  ID-creating or ID-deleting mutation succeeds. Cludia does not claim to infer
  deletions that predate that adoption point from Git.

### Explicit renumbering

- `renumber` is the sole supported numbering reset.
- It is a two-phase operation: a dry run reports the complete deterministic
  old-to-new mapping and a state-bound token; application requires that token.
- Statements are assigned contiguous IDs from their current roles in document
  order. Junctors are assigned contiguous `J` IDs in stored relation order.
- The operation rewrites every modeled internal reference and recognized root
  metadata atomically, preserves all non-identity content, validates the
  complete result, and resets the next-ID metadata.
- Unknown external references are outside the checked scope and produce a
  warning whenever an ID changes.
- There is no single-ID reuse override.

### Identity and interoperability

- IDs remain durable during ordinary mutations that do not change statement
  role. ADR 0011's monotonic premise-to-lemma reidentification is the focused
  exception. Renumbering remains the exceptional, explicit whole-file numbering
  reset.
- Slugs remain mutable aliases and are not changed by renumbering.
- Cludia does not add opaque per-statement identities to the shared `.arg`
  syntax in this slice.
- Rooted Concludia export omits Cludia allocator metadata. Imported local IDs
  become Concludia labels over server-owned opaque identities.
- Opaque cross-system provenance remains deferred until an actual incremental
  or bidirectional synchronization requirement exists.
- The TUI gains no renumber hotkey; an external CLI renumber is handled by its
  existing live-reload behavior. ADR 0010's later Top-order mutation does not
  change this boundary.

## Consequences

- Conversations and adjacent artifacts can rely on ordinary IDs without
  silent reuse after deletion. Premise promotion is a visible exception with a
  returned `P`-to-`L` mapping and external-reference warning.
- Deleted or intentionally removed records leave visible gaps until a user
  explicitly reviews a complete renumber plan.
- Scripted callers should consume successful generated IDs or atomic batch
  mappings rather than predict future allocations.
- Explicit custom IDs cease to be a focused authoring feature, although
  compatibility reads remain broader than focused writes.
- Renumbering can invalidate unknown references outside the workspace, but the
  risk occurs at one named, reviewable boundary with a complete mapping.
- A copied legacy file cannot carry history that was never recorded; the
  monotonic guarantee begins when its next-ID metadata is first persisted.

## Alternatives considered

- Reuse any currently free number: compact, but creates silent cross-revision
  ambiguity.
- Allow one-off reuse with a force flag: rejected because it makes the identity
  rule contextual and easy for agents to bypass.
- Store every retired ID: unnecessary growth when focused authoring is
  canonical and all missing numbers below the next value are reserved.
- Never permit compaction: safest for external references, but unnecessarily
  forbids deliberate document-wide cleanup.
- Add opaque IDs before changing allocation: deferred because the shared DSL is
  label-based and no bidirectional synchronization requirement yet justifies a
  format extension.
