# Cludia 1.0 Specification

> `Cludia` is the project and repository name, `cludia` is the command-line
> binary, and `profile="cludia"` is the stable permissive inquiry profile.

## 1. Status and normative language

This document specifies Cludia 1.0. `MUST`, `SHOULD`, and `MAY` are normative.

Cludia 1.0 is a Go CLI over a local `.arg` file. It also includes a terminal
navigator over the same query and
persistence layers, with focused durable Top reordering. A later local web UI
MUST have operation and information parity with the CLI.

## 2. Terms

- **Workspace:** one `.arg` document containing statements, support structures,
  and defeats. It may contain disconnected components and isolated statements.
- **Statement:** an identified, atomic, truth-apt natural-language assertion.
- **Junctor:** an identified inference object joining multiple source
  statements to one target statement.
- **Direct support:** a legacy Concludia relation from one source statement to
  one target without a junctor.
- **Defeat:** a counterpoint targeting a premise, inference, or counterpoint.
- **Rooted structure:** the selected root, its complete upstream justification
  structure, and all attached defeat chains.
- **Concludia profile:** the validation rules required for importing a rooted,
  connected argument into Concludia.
- **Cludia profile:** the relaxed validation rules used during inquiry.

## 3. Durable model

### 3.1 Document

A workspace MUST have:

- a stable document ID;
- a title;
- optional string metadata;
- at least one statement.

Statement sequence is the document's durable general order. Document-ordered
queries, rooted export, and deterministic mutation plans MUST observe it.
Interfaces MAY apply hard dependency constraints over that preference, as the
support Ledger does, but MUST NOT persist a competing view-specific order.

New workspaces MUST declare `profile="cludia"`. Legacy
`profile="workspace"` is an input alias: reads MUST NOT rewrite it, dry runs
MUST report the proposed metadata migration, and the next successful durable
save MUST atomically rewrite it to `profile="cludia"`. Failed operations MUST
leave the legacy marker unchanged.

New workspaces MUST NOT add graph artifact `meta version`. An imported or
explicitly authored version MUST be preserved. That optional value is a
Concludia graph/artifact revision, not Cludia's software version or a version
of the `.arg` syntax.

New Cludia workspaces MUST record the exact next numeric identifier for every
focused namespace in versioned `cludia-next-ids` metadata. A legacy document
without this metadata remains valid and bootstraps it from current canonical
IDs on its first successful ID-creating or ID-deleting mutation. Deletion MUST
NOT lower a recorded next value. Failed mutations and dry runs MUST NOT consume
an ID.

### 3.2 Statement

A statement MUST have:

- a stable local ID;
- an optional mutable slug;
- a role supported by the shared `.arg` syntax;
- a kind (`fact` or `value`);
- text;
- a truth token where the shared syntax permits it.

New captured statements MUST default to:

```text
premise[fact] <id> ::T "<text>"
```

Automatically generated non-empty slugs MUST satisfy workspace slug
validation. A caller-supplied invalid slug MUST still be rejected.

The CLI MUST provide explicit ways to capture unknown, false, and value
statements. It MUST NOT assign numerical confidence, credibility, or weight.
Truth `U` is for a recorded leaf proposition whose truth is genuinely
unsettled—for example, a previously accepted claim now placed in doubt. It MUST
NOT be presented as the normal persistence mechanism for speculative
hypotheses, rival theories, possibilities, or brainstorming. Those remain in
conversation or adjacent notes until explicit recorded premises are intended
to establish them.

Authored truth MAY be assigned only to an unsourced premise or unsourced
counterpoint. Statements with incoming junctor or direct support derive their
effective truth and MUST store `U`. Imported sourced statements carrying `T` or
`F` remain readable with a compatibility warning until explicit state-bound
normalization under ADR 0014.

Statement text SHOULD be atomic and SHOULD avoid embedding multiple premises
that need separate examination.

Statement identity, wording, and slug semantics follow ADR 0007:

- the ID is the durable identity of one proposition record;
- a text edit preserves that ID only when the caller explicitly asserts that
  the new wording expresses the same proposition;
- focused premise-to-lemma promotion assigns the exact next monotonic `L` ID,
  retires the previous `P` ID, rewrites modeled references atomically, and
  reports the mapping under ADR 0011;
- materially different propositions receive new IDs;
- the slug is an optional mutable human-readable alias with at most one current
  value and no retained alias history in v1;
- Cludia MUST NOT claim to determine semantic equivalence mechanically.

Identifier allocation and exceptional rewriting follow ADR 0009:

- focused creation MUST author role-appropriate canonical numeric IDs;
- `P`, `L`, `C`, `CP`, and `J` namespaces MUST advance independently and MUST
  NOT reuse deleted numbers during ordinary mutation;
- a caller-supplied ID MUST equal the exact next canonical ID for the relevant
  namespace;
- existing custom or noncanonical IDs MUST remain readable and preserved by
  ordinary round trips;
- only an explicit whole-document renumber operation may compact identifiers.

Agent-facing CLI guidance and future MCP descriptions MUST expose this contract
without assuming a particular workspace organization or use case.

### 3.3 Junctor

The focused v1 authoring operation MUST create only `AND` junctors with:

- a stable junctor ID;
- at least two distinct source statements;
- exactly one target statement.

An authored junctor asserts that the conjunction of all sources is sufficient
for the target. Removing a source changes that assertion and MUST be validated
as a mutation.

The durable presence of a junctor records that authored claim; it does not mean
the claim is currently unchallenged or accepted. An undercut MAY coexist with
the junctor while the workspace's reasoning is being reviewed or repaired.
Structural validation MUST NOT reject a junctor merely because it is undercut,
and an undercut MUST NOT silently retract or modify the junctor.

The tool MUST read, preserve, inspect, remove, and export ordinary Concludia
`OR` junctors. Creating or editing `OR` junctors MAY be deferred.

An `AND` junctor with more than three sources remains structurally valid, but
focused authoring SHOULD prefer intermediate lemmas for clarity. `derive` and
batch authoring MUST emit the stable `concludia_junctor_sources_many` warning
when they create such a junctor. `add-source` MUST emit it when an ordinary
workspace junctor crosses from three to four sources. The advisory MUST NOT
block or roll back an otherwise valid mutation and SHOULD NOT be repeated by
ordinary workspace reads.

### 3.4 Legacy direct support

The current Concludia `.arg` syntax permits one-source support clauses and
compiles them as direct support edges. V1 MUST:

- parse and preserve existing direct supports losslessly;
- expose them in human and JSON reads;
- include them in traversal and export;
- avoid creating new direct supports through its focused authoring commands.

Whether a future strict Concludia profile should prohibit direct supports is a
separate, deferred Concludia decision.

### 3.5 Defeats

V1 MUST represent:

- premise-scope defeats (undermines);
- inference-scope defeats (undercuts);
- counterpoint-scope defeats (a counterpoint targeting another counterpoint).

The tool MUST use "counterpoint of a counterpoint" in user-facing language. It
MAY accept the legacy `rejoinder` authoring alias for compatibility but MUST
normalize it to an ordinary counterpoint and MUST NOT introduce a distinct
durable rejoinder type.

An undercut MUST identify the junctor and its target. An undermine MUST identify
the target statement. A counterpoint-scope defeat MUST identify the prior
counterpoint.

Defeats MUST NOT be represented as scores or probabilities. Cludia MUST
calculate versioned grounded acceptance and effective truth under ADR 0014.
Accepted undermines force their premise target false; accepted undercuts disable
only their exact junctor-to-target justification.

When an undercut exposes a missing premise or an obsolete inference, adding the
missing source, replacing the junctor, or removing it is preferred hygiene. A
living workspace MAY retain the challenged junctor until that explicit
mutation. Its presence MUST NOT by itself prevent saving or rooted export;
export reports structural validity and preserves reachable contestation rather
than silently adjudicating it.

## 4. Structural invariants

The Cludia profile MUST enforce:

- all referenced statement and junctor IDs exist;
- IDs are unique within the document;
- statement slugs are unique within their namespace;
- junctors have exactly one target;
- newly authored junctors have two or more distinct sources;
- no self-support or self-defeat relation;
- no directed cycle through support, junctor, or defeat relationships;
- traversal is cycle-safe even when reading malformed external input;
- mutations leave the document parseable and structurally valid under its
  selected profile;
- saves are atomic.

The Cludia profile MUST allow:

- isolated statements;
- multiple disconnected components;
- no conclusion;
- multiple conclusions;
- targetless lemmas during inquiry;
- counterpoint chains of arbitrary finite depth.

Defeat relations count as connections for component discovery. Logical
dependency and cycle diagnostics MUST still distinguish support relations from
defeat relations.

## 5. Role behavior

On capture, a statement defaults to `premise`.

When an existing premise first becomes the target of a support structure, the
CLI MUST automatically promote it to `lemma` and assign the exact next
monotonic `L` ID. The former `P` ID MUST be retired, every modeled internal
reference and recognized root-metadata reference MUST be rewritten atomically,
and the mapping MUST be reported. A selected export root MUST be emitted as
`conclusion`. Other included targets MUST be emitted as `lemma`, and unsupported
non-counterpoint leaves MUST be emitted as `premise`.

Counterpoint roles MUST remain counterpoints during role reconciliation.

Automatic role changes MUST be reported in mutation results. Premise-to-lemma
promotion is the role-consistent reidentification exception defined by ADR
0011; other ordinary same-role mutations preserve the stable statement ID.

When promotion to `lemma` makes an explicit premise truth token unavailable in
the shared syntax, the durable truth value MUST normalize to `U` and the
statement update MUST be reported with the role change.

When a counterpoint first gains incoming support, its role remains
`counterpoint` but its durable truth MUST normalize to `U`. Removing that
support does not restore an earlier authored truth.

## 5.1 Effective truth evaluation

Every valid read surface MUST calculate effective truth without modifying the
workspace file:

- `AND`: any false source yields `F`; otherwise any unknown source yields `U`;
  otherwise `T`.
- `OR`: any true source yields `T`; otherwise any unknown source yields `U`;
  otherwise `F`.
- Alternative incoming justifications and legacy direct supports combine with
  `OR`.
- Unsourced lemmas and conclusions evaluate `U`.
- A sourced statement with every incoming inference disabled evaluates `F`.
- Base-effective `T` counterpoints participate in grounded acceptance;
  counterpoint defeats produce `in`, `out`, or `undecided` labels.

The compact `!` marker and the `challenged` field on Top and Ledger entries MUST
mean that grounded defeats changed the statement's truth from its base
support-propagated value. This effect propagates to downstream targets. A
direct defeat that is rebutted, inactive, or outcome-neutral MUST NOT produce
the marker. Directly attached defeats remain inspectable as typed relations in
Statement Detail and structured relation output.

Evaluation results MUST declare evaluation schema version and mode. Stored truth
and calculated effective truth remain distinct public facts.

Human read surfaces MUST present effective status according to statement role:

- premises and counterpoints retain literal `T`, `U`, and `F` truth;
- lemmas and conclusions render effective `T`, `U`, and `F` as `⊢`, `◇`, and
  `⊬`, meaning proven, possibly proven, and not proven under the authored
  argument structure.

Top and Ledger tables MUST reserve a fixed-width value column with both the
`∴` header and every truth/proof value centered; they MUST NOT label the mixed
column `TRUTH` or `STATUS`. Statement Detail MUST use the same centered width
for related-statement values so variable-length IDs do not shift adjacent text.

In the TUI, the displayed truth/proof value MUST receive warning styling under
exactly the same `challenged` condition that produces `!`: grounded defeat
changed effective status somewhere upstream. Rebutted, inactive, and
outcome-neutral counterpoints MUST style neither marker. Selection styling MAY
take precedence because `!` preserves the non-color signal.

Derived `⊬ P` MUST NOT be described as proof of the natural-language negation
of `P`. A contrary proposition must be represented and supported separately.
JSON schema 2 continues reporting the underlying `effective_truth` value so
scripts retain the complete three-valued evaluation contract.

## 6. Required capabilities

Cludia 1.0 MUST provide the following capabilities. Human CLI spelling, flags,
and JSON schemas may evolve during 1.x under the compatibility policy in
section 9.

### 6.1 Read and query

- List all statements and junctors.
- Show one statement or junctor with incoming and outgoing relations.
- Search statement text and IDs.
- List isolated statements.
- List connected components.
- Read a complete rooted structure.
- List non-counterpoint statements with no outgoing support, longest upstream
  support depth, and challenge state, with optional challenged-only filtering
  and document-order offset/limit pagination.
- Read a complete, stable, topologically ordered support ledger for a selected
  non-counterpoint statement, delaying premises toward their first use while
  using general document order to resolve equivalent choices.
- Optionally restrict that ledger at its root to one explicitly selected
  incoming junctor while retaining the complete upstream closure of the
  junctor's sources. The selected junctor MUST target the ledger root; selection
  is a read-only view and MUST NOT persist a preferred derivation.
- List defeats targeting or originating from an element.
- Validate under the Cludia or Concludia profile.
- Read versioned machine guidance for statement identity and revision behavior.

### 6.2 Capture and edit

- Add a statement.
- Atomically add statements from versioned structured input and return an
  ordered caller-key-to-statement mapping; batch schema version 2 may also
  create focused `AND` derivations and typed defeats with final mappings.
- Edit statement text without changing its stable identity only with an
  explicit same-proposition assertion.
- Change truth and kind where valid.
- Calculate complete effective truth and grounded counterpoint acceptance.
- Plan and atomically normalize legacy authored truth on sourced statements.
- Rename, regenerate, or clear an optional statement slug without changing the
  statement ID or durable relations.
- Delete a statement with a dry-run plan showing incident relations and
  resulting orphans/components.
- Plan and apply material replacement between two existing non-counterpoint
  statement records through explicit per-relation choices and a state-bound
  apply token.
- Plan and apply deterministic whole-document renumbering through a state-bound
  old-to-new mapping that rewrites all modeled internal references atomically.
- Move one statement immediately before or after another in durable general
  document order without changing identity, content, relations, or allocator
  state.

### 6.3 Construct and repair

- Create a new target and `AND` junctor from two or more existing sources.
- Add a new `AND` justification to an existing target.
- Add a source to an existing junctor.
- Atomically replace one source of an existing `AND` junctor while preserving
  source position and validating the complete resulting workspace.
- Remove a source when the remaining construct is valid.
- Remove a junctor.
- Add an undermine.
- Add an undercut.
- Add a counterpoint targeting another counterpoint.
- Route a convenience challenge by element type without changing defeat
  semantics: premises are undermined, counterpoints receive counterpoints,
  junctors are undercut, and a derived statement selects its sole incoming
  junctor or requires an explicit junctor when the choice is ambiguous.
- Remove a counterpoint with a dry-run structural plan.

### 6.4 Export

- Export the complete rooted structure for a selected root.
- Reconcile roles in the exported artifact.
- Include all upstream justifications rather than selecting one proof path.
- Include attached defeat chains recursively.
- Validate the result under the Concludia profile.
- Refuse invalid export without modifying the workspace or output file.

## 7. CLI and JSON contract

The process-level exit-status, stream, failure, collection, and mutation-timing
contract is specified in [docs/cli-json.md](docs/cli-json.md) and MUST be tested
against the compiled `main` entrypoint.

Human-readable output MUST be the default. Every scriptable read and mutation
MUST support `--json`.

JSON is a public interface and MUST:

- carry an independent schema version;
- use stable field meanings;
- distinguish statements, junctors, direct supports, and defeats;
- report the active validation profile;
- report structured diagnostics with codes, messages, severity, and element
  references;
- report all durable changes made by a mutation;
- avoid silently dropping constructs unknown to a focused command.

During 1.x, incompatible JSON changes MUST increment the applicable schema
version and be disclosed in release notes. Cludia does not promise to retain
implementations of older request or response schemas. Version 1.0.0 uses CLI
response schema 2, batch input schema 2, and evaluation schema 1.

CLI response schema version 2 adds calculated effective truth to read surfaces.
`statement.truth` remains the persisted value. Evaluation results use their own
schema version 1 and mode `grounded`. Batch input has independent schema version
2.

Reference resolution follows ADR 0012. In statement contexts, exact statement
IDs precede slugs. In statement-or-junctor contexts, every exact durable ID
precedes every slug. Imported slug/ID collisions remain readable with a stable
warning, while focused mutations MUST reject creating a new collision without
writing.

Flags SHOULD be accepted consistently before or after positional arguments.
Mutation commands that can remove or cascade through relations SHOULD support
`--dry-run`.

Successful derive output for premise-to-lemma promotion MUST report
`role_changes` entries containing `previous_id`, `current_id`, `from`, and `to`.
Its `changes` collection MUST report the statement reidentification, all
recognized metadata updates, the new junctor, and allocator metadata changes.
Diagnostics MUST warn that references outside the workspace were not rewritten.

Top-level `--help` and `-h` MUST print usage successfully. Command-specific
usage MUST be reachable through `help COMMAND` as well as the command's normal
help flag.

Agent-facing guidance MUST instruct scripted callers not to predict generated
IDs across independent mutations. Callers SHOULD omit explicit IDs and consume
the assigned ID from each successful structured mutation result; any explicit
ID MUST equal the role-appropriate exact next ID.
Guidance MUST also state that attached counterpoints are removed explicitly
before deleting their target or an incident inference. It MUST distinguish
accepted but disconnected facts and values from speculative hypotheses, keep
hypotheses and rival theories external until recorded premises are intended to
prove them, reserve truth `U` for genuinely uncertain recorded propositions,
and state that Cludia does not author confidence scores or probabilities.
Guidance MUST explain that a defeat has grounded truth consequences and is
appropriate only when accepting a case-specific counterpoint would make a
premise false or materially out of scope, make the exact inference
insufficient, or defeat an earlier counterpoint. Bare possibility, unsupported
rival explanation, absence of direct proof, or residual uncertainty MUST NOT
be presented as automatically defeating a claim.

Batch input version 2 is the atomic authoring transaction over statements,
focused `AND` derivations, and typed defeats. Its shape is:

```json
{
  "schema_version": 2,
  "statements": [
    {"key": "source-a", "text": "Source A is true."},
    {"key": "source-b", "text": "Source B is true."},
    {"key": "finding", "text": "The finding follows."},
    {"key": "objection", "role": "counterpoint", "text": "The documented exception applies in this case and is absent from the stated sources."}
  ],
  "derivations": [
    {
      "key": "finding-inference",
      "sources": [{"key": "source-a"}, {"key": "source-b"}],
      "target": {"key": "finding"}
    }
  ],
  "defeats": [
    {
      "from": {"key": "objection"},
      "scope": "inference",
      "target": {"key": "finding-inference"}
    }
  ]
}
```

Every new statement and derivation MUST have a non-empty caller key that is
unique across both collections. A reference MUST contain exactly one of:

- `key`, naming a new statement or derivation in the same transaction; or
- `id`, naming a durable statement or junctor that existed before the
  transaction.

Batch relation references MUST NOT resolve slugs or tentative generated IDs.
New statements MAY explicitly request a role; otherwise a statement targeted
by a batch derivation is created directly as a lemma and any other statement
defaults to premise. Counterpoints MUST request the counterpoint role. Newly
sourced statements store `U`; a newly derived target MUST NOT consume and
retire a transient `P` ID.

Derivations author only focused `AND` junctors with at least two sources.
Defeats explicitly declare premise, inference, or counterpoint scope. The
`from` reference must resolve to a counterpoint; an inference target resolves
to a derivation/junctor, while premise and counterpoint targets resolve to
statements. Version 2 does not add direct-support authoring or allow one
counterpoint to acquire multiple `.arg` defeat targets.

The `derivations` and `defeats` collections MAY be empty when the transaction
only captures statements. At least one statement, derivation, or defeat is
required.

The complete transaction MUST be planned on a clone, validated once as a
whole, and saved atomically only when every statement and relation is valid.
Any parse, reference, allocation, relation, cycle, or profile failure leaves
the workspace and allocator unchanged. Dry-run mappings are tentative. Applied
output MUST return final caller-key mappings for statements and junctors,
including role-consistent final IDs, plus every created defeat and any role or
truth normalization affecting pre-existing statements.

Material replacement MUST be a two-phase operation. A dry run MUST report the
old and replacement records, every incident support and defeat, each explicitly
selected downstream `AND` source retarget, each explicitly selected old
justification removal, affected inference defeats, recognized root handling,
component changes, deletion blockers, and whether the old record remains. An
applicable plan MUST return a token bound to the current document and exact
choices. Application MUST repeat those choices and present the token, and MUST
refuse without writing when the token is stale.

The replacement operation MUST NOT offer automatic retarget-all behavior. The
old statement remains by default. Explicit deletion MUST fail while any
unselected support, direct support, defeat, or recognized root reference remains
incident to the old record. Counterpoint records are excluded from focused
material replacement and continue to use counterpoint-specific operations.

Whole-document renumbering MUST also be a two-phase operation. A dry run MUST
report the complete statement and junctor mapping, recognized root-metadata
effects, before/after next-ID state, external-reference warning, and a token
bound to the current document and mapping. Application MUST require that token,
refuse stale plans without writing, validate the complete rewritten document,
and save atomically. Statements MUST be numbered by current role in document
order; junctors MUST be numbered in stored relation order. Slugs and all
non-identity content MUST remain unchanged.

Statement reordering is an immediate, non-destructive structural presentation
mutation. It MUST accept statement IDs or slugs, report canonical statement and
anchor IDs plus one-based previous and current positions, validate the complete
result, and save atomically. An already satisfied immediate placement succeeds
without rewriting the file. Reordering MAY affect later document-ordered
queries, rooted export order, and reviewed renumber mappings; it MUST NOT itself
change any ID or next-ID metadata.

The CLI MUST NOT require installation when run from a checkout. Once Go code
exists, ordinary validation will include:

```bash
go test ./...
go vet ./...
go build ./cmd/...
```

## 8. LLM boundary

V1 MUST NOT call an LLM or require an API key.

The intended integration direction is:

```text
human <-> external LLM -> CLI/MCP -> workspace file
```

not:

```text
human -> workspace tool -> embedded LLM
```

An external LLM may use any reasoning mode to propose candidates. The CLI MUST
perform only structural validation and MUST NOT pretend to mechanically prove
natural-language entailment.

Mutations proposed by an LLM SHOULD be presented to the user for semantic
review before persistence unless the user has explicitly authorized the exact
change.

## 9. File compatibility

V1 MUST use the existing Concludia `.arg` syntax as its common syntax. Stable
metadata identifies the relaxed Cludia profile:

```text
meta profile="cludia"
```

The legacy value `profile="workspace"` MUST be accepted as an input alias and
migrated only on a successful durable save as specified in section 3.1. The
profile denotes validation semantics; it is not a distinct syntax.

Cludia 1.x guarantees that prior Cludia `.arg` files and supported ordinary
Concludia `.arg` files remain readable without silent loss. This file
compatibility promise does not freeze human CLI text, command flags, or JSON
schemas. The `.arg` syntax remains unversioned. Optional `meta version` is a
Concludia graph/artifact revision and MUST NOT be interpreted as Cludia's
software version or a DSL version.

A valid ordinary Concludia `.arg` document MUST be readable as a workspace.
Round-tripping it without an explicit transforming operation MUST preserve:

- statements and IDs;
- roles, kinds, truth values, slugs, and labels;
- `AND` and `OR` junctors;
- direct supports;
- defeats and recursive counterpoint chains;
- metadata not owned by the focused operation.

Cludia's `cludia-next-ids` metadata is allocator state for the workspace layer,
not part of a published argument, and MUST be omitted from rooted Concludia
export. Concludia's server-side opaque identities are not represented in the
portable label-based `.arg` syntax.

A workspace document is not necessarily importable by Concludia. Exporting a
rooted structure is the explicit boundary.

## 10. Concludia export

Export behavior is specified in
[docs/concludia-interoperability.md](docs/concludia-interoperability.md).

In summary, export MUST compute the complete rooted support closure, include
all attached defeats recursively, reconcile roles, select the requested root,
and pass Concludia-profile validation.

Semantic validity remains a human/LLM audit. Structural export success MUST NOT
be described as proof that the argument is sound.

## 11. Interface parity

The TUI and a future web UI MUST use the same domain operations as the CLI.
They MUST NOT interpret or write the `.arg` file through independent
implementations.

Every durable mutation MUST be a clone-returning shared semantic operation with
typed result facts and stable failures as specified by ADR 0013. CLI command
files MUST NOT directly edit document fields or relation slices. Shared
workspace orchestration MUST validate the complete proposed document before any
atomic create or save. Dry-run and apply paths for a given operation MUST derive
their effects from the same shared planner.

Parity means:

- every durable mutation exposed in the UI is available through the CLI;
- every durable mutation exposed through the CLI can be initiated or
  meaningfully inspected in the UI;
- every fact shown visually is available in structured CLI output.

The TUI MUST expose Top, Statement Detail, and Derivation Ledger views and
automatically reload valid external file changes without replacing its last
valid in-memory document with invalid contents. Capital `J` and `K` in Top MAY
move the highlighted statement through the shared durable statement-order
operation. The TUI MUST refuse a stale move when external changes invalidate
the displayed Top adjacency. Text wrapping MUST measure grapheme-aware terminal
display cells, and the final rendered view MUST fit the current terminal width
and height even at extreme dimensions. PgUp and PgDn MUST move by the rendered
viewport rather than a fixed logical-item count while keeping the new selection
visible. Other durable TUI authoring remains deferred.

Pixel layout, graph rendering, keyboard navigation, and batch ergonomics need
not be identical.

## 12. V1 non-goals

- Built-in LLM calls, prompt orchestration, or API-key management.
- Embeddings or autonomous background discovery.
- Bayesian inference, confidence percentages, or evidence weights.
- Structured source-document ingestion or provenance.
- Entity extraction, timelines, maps, or OSINT collection.
- Collaboration, accounts, hosted storage, or publication.
- Native macOS packaging.
- General TUI mutation authoring, graphical graph maps, and TUI-only semantic
  queries beyond focused statement ordering.
- A universal format shared with Dagim.
- New `OR` or direct-support authoring UX.

## 13. Acceptance scenarios

V1 is complete when automated tests demonstrate at least:

1. A workspace containing isolated statements and multiple components parses,
   validates, and round-trips.
2. Two captured statements can be combined through a new `AND` junctor and
   target without manual file editing.
3. An existing premise promoted to a target receives the exact next `L` ID,
   retires its `P` ID, and atomically rewrites every modeled reference while
   reporting the mapping and external-reference warning.
4. Undermines, undercuts, and counterpoints of counterpoints round-trip and are
   discoverable through JSON.
5. Directed relation cycles are rejected and malformed cyclic input can still
   be diagnosed without nontermination.
6. A normal Concludia `.arg` containing `OR`, direct supports, and defeats can
   be read and rewritten without semantic loss.
7. Root export includes every upstream justification and attached recursive
   defeat chain while excluding unrelated workspace islands.
8. Export either produces a Concludia-profile-valid `.arg` or leaves no output.
9. Concurrent or failed mutation does not leave a partially written file.
10. An external LLM can perform the core workflow using only documented JSON
    commands.
11. A challenged junctor and a repaired replacement can coexist, save, and
    appear together in rooted export until the original is explicitly removed.
12. Generated slugs for digit-leading statement text are valid, while an
    explicitly supplied invalid slug is rejected without changing the file.
13. A multi-statement batch either returns an ordered key-to-statement mapping
    and saves every statement, or saves none when any input is invalid.
14. Material replacement requires a reviewed state-bound plan, retargets only
    selected downstream sources, reports affected defeats, and cannot delete
    the old statement while an unreviewed incident relation remains.
15. Successful focused allocation advances exact next IDs, deletion leaves
    gaps, and failed or dry-run mutation consumes no number.
16. Whole-document renumbering requires a reviewed state-bound plan, rewrites
    every modeled reference, preserves non-identity content, and reports the
    complete old-to-new mapping.
17. Statement reordering persists one general order atomically, changes no
    identity or relation, and is shared by CLI and Top `J`/`K` controls.
18. Ledger order always places sources before targets and delays a premise used
    only with a derived source until that source's derivation is complete.
19. Top-level help exits successfully, and command-specific help is available
    without executing the command.
20. Smart challenge routing preserves defeat scope, auto-selects only a sole
    incoming junctor, and refuses ambiguous or direct-support-only targets
    without writing.
21. Top challenged-only filtering and offset/limit pagination preserve document
    order and do not alter the complete rooted query contract.
22. Human and versioned machine guidance distinguish accepted disconnected
    facts and values from speculative hypotheses, reserve `U` for genuinely
    uncertain recorded propositions, and distinguish grounded semantic defeats
    from bare possibility, unsupported alternatives, lack of direct proof, and
    residual uncertainty.
23. True, false, and unknown leaves propagate through AND, OR, alternative
    justifications, and direct support with versioned grounded defeat effects.
24. Manual truth edits reject sourced statements, supported counterpoints
    normalize to `U`, and legacy sourced truth is repaired only through a
    reviewed state-bound normalization operation.
25. Batch schema version 2 can atomically create statements, role-correct
    derivation targets, generated junctors, undercuts or undermines, and
    counterpoints of counterpoints through caller keys; any invalid relation or
    final graph writes nothing and consumes no identifier.
26. Creating an `AND` junctor with more than three sources through `derive` or
    batch, or crossing that threshold through `add-source`, succeeds with one
    actionable warning while ordinary workspace reads remain quiet.
27. `ledger --inference` includes only the selected incoming junctor at the
    root, retains complete support below its sources, rejects mismatched
    junctors, marks accepted undercuts, and identifies when omitted root
    justifications change overall root truth without misaligning human columns.
28. Human reads show premise/counterpoint truth as `T/U/F` and
    lemma/conclusion provability as `⊢/◇/⊬`, without changing JSON effective
    truth or treating an unproven proposition as a proven negation. Top and
    Ledger center those values beneath `∴` without a `TRUTH` or `STATUS`
    header, and Statement Detail preserves the same fixed-width alignment.

## 14. Open decisions

Deferred decisions are tracked in [docs/open-questions.md](docs/open-questions.md).
