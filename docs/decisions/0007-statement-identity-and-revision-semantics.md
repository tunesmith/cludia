# ADR 0007: Distinguish statement identity, wording, and slug aliases

- Status: Accepted
- Date: 2026-08-24

## Context

Cludia statements have a required local ID, optional slug, and editable natural-
language text. The first dogfood workspace exposed three materially different
operations that were all easy to describe informally as “editing”:

- reformulating one proposition without intending to change its meaning;
- renaming a stale human-readable slug after wording evolves;
- replacing one proposition with another whose truth conditions differ.

Those operations have different consequences for support relations, defeats,
external references, and Git history. In particular, changing “Cludia cannot
yet create inferences” to “Before the derive slice, Cludia could not create
inferences” changes temporal scope and therefore may create a new proposition
rather than merely clarify the old one.

An external LLM can help classify and refactor such changes, but Cludia cannot
mechanically prove that two natural-language formulations are semantically
equivalent. The caller must make that judgment explicitly.

The connected argument supporting this decision is
[0007-statement-identity-and-revision-semantics.arg](0007-statement-identity-and-revision-semantics.arg).

## Decision

### Use-case neutrality

- These identity rules apply within any `.arg` document regardless of how the
  document is used.
- Cludia does not require a current-state/history pair, an append-only ledger,
  or any other fixed document arrangement.
- CLI guidance and future MCP descriptions MUST explain identity semantics
  without assuming a particular workspace topology or documentation workflow.
- A current-state workspace paired with a historical-reasoning workspace is a
  documented incident-investigation use case, not a format profile or product
  invariant.
- Other valid uses include a single evolving current model, a premise notebook,
  an ADR companion argument, a rooted publication draft, and multiple related
  workspaces organized by the user.

### Statement IDs

- A statement ID is the required durable identity of one proposition record.
- A text edit preserves the ID only when the caller intentionally asserts that
  the new wording expresses the same proposition.
- Text-changing CLI operations MUST require an explicit identity-continuity
  assertion such as `--same-proposition`.
- Truth- and kind-only edits do not require that assertion because their changed
  attributes are explicit in the request and result.
- A materially different proposition receives a new statement ID.
- Cludia MUST NOT claim to determine natural-language semantic equivalence
  structurally or automatically.

### Identifier lifetime after deletion

- This ADR does not decide whether a deleted statement or junctor ID may be
  reused in a later revision of the same document lineage.
- “Durable identity” here means continuity while a proposition record exists
  and across explicitly meaning-preserving edits; it does not introduce
  tombstones or permanent ID reservation.
- Concludia's separation between durable server identity and renumberable
  display labels is relevant prior art for a future decision.
- The full-investigation pilot should determine whether cross-revision or
  external references create enough ambiguity to require retired-ID metadata or
  a separate opaque identity/display-label model.

### Slugs

- A slug is an optional, mutable, human-readable alias; it is not identity.
- A statement has at most one current slug in v1.
- Cludia does not retain old slugs as permanent aliases in the `.arg` model.
- Slug refactoring updates the declaration and recognized in-scope metadata
  references, including root metadata, while durable relations remain ID-based.
- The operation reports the project scope it checked and warns that unknown
  external references may break.
- A statement without a slug remains fully addressable by ID, and rooted
  metadata falls back to the ID.

### Agent guidance

- CLI help and a versioned structured guidance surface MUST expose the identity
  contract before mutation.
- Future MCP tool descriptions MUST carry the same contract.
- Mutation output reports the caller's declared continuity intent and every
  durable identity or alias change.
- Repository guidance remains authoritative for agents operating from a
  checkout, but the CLI cannot assume every external agent has read it.

### Material replacement

- Material replacement is conceptually distinct from text editing and slug
  renaming.
- In v1, it may be performed explicitly from existing primitives: add the new
  statement, audit and repair each relation, then retain or delete the old
  statement.
- A future `replace` convenience command MUST begin with a structural dry run
  covering the proposed new identity, every affected support and defeat,
  relation-retargeting choices, old-statement retention or deletion, and
  component changes.
- Replacement MUST NOT retarget every relation automatically; each relation may
  depend on the exact old proposition.
- The command name, selection UX, and defaults for retaining the old statement
  remain deferred until manual replacement is dogfooded.

Git remains the history of earlier tracked file revisions under ADR 0006. The
current `.arg` revision contains the current proposition records and current
slugs, not an embedded alias or replacement history.

## Consequences

- Stable IDs continue to keep graph relations and external references durable
  across genuine reformulation.
- Text edits gain friction because continuity must be declared explicitly.
- Slugs can remain semantically useful instead of becoming permanent stale
  labels.
- Project-local slug refactoring is safe and inspectable, but cannot guarantee
  unknown external consumers are updated.
- Material semantic changes become visible graph operations rather than hidden
  text substitutions.
- Git history and explicit mutation results provide revision evidence without
  extending shared `.arg` syntax with alias collections or supersession edges.
- Existing callers of `edit --text` require migration to declare
  `--same-proposition`.
- Agents can apply the identity contract without being instructed to create or
  maintain files that are irrelevant to the user's chosen workflow.
- Deleted-ID reuse remains unchanged and explicitly unresolved; this decision
  adds no allocator metadata or tombstone semantics.

## Alternatives considered

- Preserve IDs for every text edit: simple, but conflates reformulation with a
  materially new proposition.
- Mint a new ID for every text edit: semantically conservative, but causes
  needless relation churn for ordinary corrections and clearer wording.
- Keep slugs immutable: protects external aliases but leaves misleading names
  after legitimate reformulation.
- Retain all old slugs as aliases: convenient for compatibility, but requires a
  new public alias model, collision rules, and serialization semantics.
- Let an LLM decide silently whether meaning is preserved: rejected because the
  judgment is semantic, fallible, and consequential for every incident
  relation.
- Automatically retarget all relations during replacement: rejected because an
  inference or defeat authored against the old wording may not remain valid for
  the new proposition.
- Require separate current-state and historical-reasoning files: useful for
  some incident investigations, but rejected as a universal convention because
  Cludia supports many valid workspace organizations.
