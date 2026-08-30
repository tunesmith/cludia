# `.arg` Workspace Profile

## Status

This document defines the relaxed validation profile implemented over
Concludia's existing text `.arg` syntax.

The literal metadata value `profile="workspace"` is provisional. It describes
format validation behavior and is not intended to name the product. It may be
renamed before the first public release without implying a change to the tool
name.

## Design

The syntax and its validation profile are separate concerns.

The shared syntax represents:

- one document header and metadata;
- statements with roles, kinds, IDs, optional slugs, truth tokens, and text;
- `AND` and `OR` support clauses;
- legacy one-source direct supports;
- premise-scope defeats;
- inference-scope defeats;
- counterpoint-scope defeats.

Concludia applies publication-oriented validation: the graph is connected,
isolated statements are errors, premises are support leaves, and a root can be
chosen.

The workspace profile relaxes topology during discovery while retaining
referential integrity and cycle safety.

## Header and metadata

An initial workspace uses the existing header:

```text
argument case-id "Case title"
meta profile="workspace", version="0.1.0"
```

`version` in current `.arg` practice describes the content artifact. It MUST
NOT silently become the syntax version. A future language-level format version
is an open design decision.

If `profile` is absent, the document retains ordinary Concludia semantics. The
new tool may still open it in workspace mode without rewriting the file.

Focused Cludia authoring records exact next numeric IDs in one metadata value:

```text
meta profile="workspace", version="0.1.0", cludia-next-ids="v1;P=2;L=1;C=1;CP=1;J=1"
```

The five namespaces advance independently. Deletion leaves gaps and does not
lower these values. Existing files without the field remain compatible and
bootstrap it from their current canonical IDs when an ID-creating or
ID-deleting mutation next succeeds. Malformed versions or values are errors;
values behind IDs already present are diagnosed and safely advanced by the
next such mutation.

## Statements

Examples:

```text
premise[fact] P1:broken-window ::T "The window was broken from inside."
premise[value] P2:avoid-accusation ::T "We should not accuse someone without a sufficient argument."
lemma[fact] L1:staged "The apparent burglary was staged."
conclusion[fact] C1:final-finding "The apparent burglary was staged."
```

Focused capture defaults to `premise[fact] ... ::T`. Truth tokens remain the
existing `T`, `F`, and `U`; no numerical confidence field is introduced.

Only unsourced premises and unsourced counterpoints carry authored truth.
Sourced statements store `U` and receive calculated effective truth under ADR
0014. Imported sourced `T`/`F` tokens are preserved and warned about until the
explicit state-bound `normalize-truth` repair is applied.

The statement ID is required durable identity. The slug remains optional and is
a mutable human-readable alias with one current value; durable relations are
resolved and canonically written by ID. Text edits that preserve an ID require
explicit same-proposition intent under ADR 0007.

Unqualified references follow ADR 0012: exact durable IDs take precedence over
statement slugs. Imported slug/ID collisions are preserved and warned about so
ordinary Concludia files remain readable. Focused slug generation and mutation
do not create new collisions with statement or junctor IDs.

Statement declaration sequence is the durable general order. Reordering moves
the complete statement block, including support clauses attached to that target,
without changing any modeled relation. Top, search, components, rooted export,
and deterministic plans observe this sequence. Dependency-constrained views may
use it only as a preference. No separate Top or Ledger order is stored.

Focused creation authors only canonical role-appropriate IDs (`P`, `L`, `C`,
or `CP` plus a positive decimal suffix) and accepts an explicit ID only when it
is the exact recorded next value. Focused premise-to-lemma promotion assigns
the exact next `L` ID, retires the former `P` ID, and rewrites all modeled
references under ADR 0011. Ordinary reads and round trips continue to preserve
broader valid IDs and legacy role-mismatched IDs from existing `.arg` files.
ADR 0009's explicit whole-document renumber remains the only operation that
compacts these labels.

The workspace allows a premise to be isolated. Once it becomes the target of a
support relation, focused mutations promote it to a lemma with the exact next
monotonic `L` ID.

Atomic batch input schema version 2 declares its complete new topology before
allocation. A new statement targeted by a batch derivation is therefore written
directly as a lemma with the next `L` ID rather than receiving and immediately
retiring a temporary `P` ID. References to new batch elements use caller keys;
references to existing workspace elements use exact durable IDs. The final
transaction must still satisfy this profile and serialize through the unchanged
`.arg` syntax.

## Multi-premise support

Focused authoring creates:

```text
lemma[fact] L1:crossed-wall "The intruder crossed the garden wall."
  <- AND#J1(P1, P2, P3)
```

The clause asserts that the conjunction of all listed sources is sufficient
for the target.

V1 focused authoring:

- creates only `AND`;
- requires at least two distinct sources;
- creates no direct statement-to-statement support.

The parser and serializer must nevertheless preserve `OR` and direct supports
found in ordinary Concludia files.

## Legacy direct support

The current syntax accepts a support clause with one reference and represents
it as direct support rather than as a junctor. The new tool follows the chosen
compatibility policy:

- read it;
- report it explicitly;
- preserve it on round-trip;
- traverse and export it;
- do not create it through focused v1 authoring.

A later Concludia decision may introduce a stricter profile that prohibits it.
That decision is not part of Cludia v1.

## Defeats

### Undermine a statement

```text
undermine CP1:rain-erased-prints ::T "Rain may have erased the returning footprints." -> premise P2
```

### Undercut an inference

```text
undercut CP2:vehicle-pickup ::T "A vehicle could have collected the intruder at the wall." -> inference J1:target L1
```

### Counterpoint of a counterpoint

```text
counterpoint CP3:no-vehicle-access ::T "A locked gate prevented vehicle access to the wall." -> counterpoint CP2
```

There is no distinct rejoinder model. A legacy `rejoinder` token may be accepted
as a parser alias, but the durable role and user-facing concept are
`counterpoint`.

Counterpoint chains may be arbitrarily deep but must be finite and acyclic.

### Contested inferences in a living workspace

An undercut challenges an authored sufficiency claim but does not delete or
rewrite the junctor it targets. Both may coexist while the reasoning is in
progress, including after a repaired replacement junctor has been added.

If the author concludes that the challenged inference is missing a premise or
is obsolete, repairing or removing it is preferred hygiene. Until that explicit
mutation, the challenged junctor remains valid workspace structure. Its
presence does not by itself prevent saving or rooted Concludia export, which
preserves reachable defeats and reports structural rather than semantic
validity.

## Validation profiles

| Rule | Workspace | Concludia |
|---|---:|---:|
| Valid references and unique IDs | required | required |
| Cycle-free directed relations | required | required |
| Isolated statements | allowed | rejected |
| Multiple disconnected components | allowed | rejected |
| No conclusion | allowed | draft warning or rejection by importing context |
| Multiple conclusions | allowed | allowed when one root can be selected |
| Targetless lemmas | allowed | warning or mutation restriction |
| Premise targeted by support | focused mutation promotes role | rejected unless promoted |
| Defeat chains | allowed | allowed |
| Authored T/F on sourced statement | compatibility warning; ignored by evaluation | compatibility warning; ignored by evaluation |

## Effective truth

Effective truth is a calculated overlay rather than serialized cache state.
Cludia evaluates strong three-valued `AND`/`OR`, disjunction across alternative
justifications and direct supports, and grounded counterpoint acceptance.
Accepted undermines force premise targets false; accepted undercuts disable the
identified inference edge. Evaluation outputs declare schema version 1 and mode
`grounded`.

On Top and Ledger reads, `!` (and JSON `challenged`) means that this grounded
overlay changed the statement's truth from the value obtained by propagating
support without defeats. The effect can originate anywhere upstream. Merely
having a direct counterpoint is insufficient when it is rebutted, inactive, or
does not change the result; direct defeat relations remain separately
inspectable.

A valid Concludia document should be a valid workspace. A valid workspace is
not necessarily a valid Concludia argument.

## Components

Component discovery treats support and defeat incidence as connection. This
keeps a counterpoint and its counterpoints-of-counterpoints with the reasoning
island they discuss.

The implementation must retain typed relations. A defeat connection must not
be mistaken for logical support, and logical support traversal must not follow
a defeat as if it were a premise.

## Round-trip requirements

Opening and saving without an explicit transforming operation must preserve:

- statement order, IDs, roles, kinds, truth, labels, slugs, and statement text;
- junctor IDs, connectors, source order, and targets;
- direct supports;
- all defeat scopes and targets;
- recognized and unrecognized string metadata;
- ordinary Concludia documents that use constructs the focused workspace UX
  does not create.

Round trips without explicit normalization preserve legacy sourced truth tokens
even though evaluation ignores them.

If exact trivia preservation such as comments or whitespace is not supported,
the first implementation must state that canonical rewrite behavior clearly
and test semantic round-trip equivalence.

The initial Go implementation uses canonical rewrite behavior. It preserves all
modeled statements, metadata entries, source order, `AND` and `OR` junctors,
legacy direct supports, and defeat scopes, but normalizes whitespace, statement
role aliases, and reference spelling and does not preserve comments. It must
reject syntax it cannot represent instead of silently dropping it. Semantic
round-trip equivalence is covered by fixtures.
