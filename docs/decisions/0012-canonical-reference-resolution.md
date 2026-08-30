# ADR 0012: Resolve durable IDs before statement slugs

- Status: Accepted
- Date: 2026-08-30

## Context

Cludia accepts durable statement IDs and mutable statement slugs in many
arguments. Imported `.arg` files can contain a slug equal to another statement
ID or a junctor ID. The parser already preferred an exact statement ID, but the
in-memory statement lookup returned the first declaration whose ID or slug
matched. Commands accepting either statements or junctors also disagreed about
which element type to check first.

Consequently, one unqualified token could resolve differently in parsing,
`show`, `component`, `challenge`, domain operations, or the TUI. Rejecting every
existing collision would make Cludia unable to read otherwise valid Concludia
files, contrary to the compatibility contract.

## Decision

- All statement reference resolution checks exact statement IDs before slugs,
  independent of declaration order.
- Resolution in a typed statement context considers statement IDs and slugs;
  an unrelated junctor ID does not change the meaning of syntax that requires a
  statement.
- Resolution in a statement-or-junctor context checks every exact durable ID
  before any statement slug. The document's existing cross-element ID
  uniqueness rule makes the exact result singular.
- Parser, CLI, query, domain, and TUI code use these document-level resolution
  rules rather than implementing their own precedence.
- Imported slug/ID collisions remain readable and losslessly preserved. Both
  validation profiles emit the stable `statement_slug_shadows_id` warning.
- Focused slug generation reserves every statement and junctor ID. Focused
  operations reject an explicit slug that would shadow another element ID with
  `statement_slug_id_collision` and do not write.
- Explicit `id:` and `slug:` qualification is deferred until a compatibility
  case demonstrates that a shadowed slug must itself remain directly
  addressable. Its owning statement remains addressable by durable ID.

## Consequences

- An exact durable ID has one stable meaning across commands.
- Existing Concludia documents are not rejected merely because an alias is
  shadowed.
- A shadowed imported slug is informative metadata but cannot override the
  durable ID in an unqualified reference.
- New focused Cludia mutations do not introduce additional shadowed aliases.

## Alternatives considered

- Reject collisions during every parse: unambiguous, but breaks the promise to
  read and preserve ordinary Concludia files that currently permit them.
- Give slugs precedence: makes mutable aliases override durable identities and
  leaves scripts vulnerable to declaration-order changes.
- Preserve command-specific precedence: retains compatibility only by keeping
  the correctness defect.
