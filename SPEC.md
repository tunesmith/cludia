# Cludia v1 Specification

> `Cludia` is the permanent project and repository name, and `cludia` is the
> command-line binary. The `profile="workspace"` identifier remains
> provisional.

## 1. Status and normative language

This document specifies the intended first implementation. `MUST`, `SHOULD`,
and `MAY` are normative.

The first implementation is a Go CLI over a local `.arg` file. After CLI
dogfooding, v1 also includes a read-only terminal navigator over the same query
and persistence layers. A later local web UI MUST have operation and information
parity with the CLI.

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
- **Workspace profile:** the relaxed validation rules used during inquiry.

## 3. Durable model

### 3.1 Document

A workspace MUST have:

- a stable document ID;
- a title;
- optional string metadata;
- at least one statement.

The workspace-profile metadata value is provisional. Format profile and
content version MUST remain conceptually distinct even if the initial syntax
stores both in `meta`.

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

Statement text SHOULD be atomic and SHOULD avoid embedding multiple premises
that need separate examination.

Statement identity, wording, and slug semantics follow ADR 0007:

- the ID is the durable identity of one proposition record;
- a text edit preserves that ID only when the caller explicitly asserts that
  the new wording expresses the same proposition;
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

Defeats MUST NOT be represented as scores or probabilities. V1 is required to
store and traverse them but is not required to implement a new acceptance or
truth-propagation calculus.

When an undercut exposes a missing premise or an obsolete inference, adding the
missing source, replacing the junctor, or removing it is preferred hygiene. A
living workspace MAY retain the challenged junctor until that explicit
mutation. Its presence MUST NOT by itself prevent saving or rooted export;
export reports structural validity and preserves reachable contestation rather
than silently adjudicating it.

## 4. Structural invariants

The workspace profile MUST enforce:

- all referenced statement and junctor IDs exist;
- IDs are unique within the document;
- junctors have exactly one target;
- newly authored junctors have two or more distinct sources;
- no self-support or self-defeat relation;
- no directed cycle through support, junctor, or defeat relationships;
- traversal is cycle-safe even when reading malformed external input;
- mutations leave the document parseable and structurally valid under its
  selected profile;
- saves are atomic.

The workspace profile MUST allow:

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
CLI SHOULD automatically promote it to `lemma`. A selected export root MUST be
emitted as `conclusion`. Other included targets MUST be emitted as `lemma`, and
unsupported non-counterpoint leaves MUST be emitted as `premise`.

Counterpoint roles MUST remain counterpoints during role reconciliation.

Automatic role changes MUST be reported in mutation results and MUST preserve
stable statement identity.

When promotion to `lemma` makes an explicit premise truth token unavailable in
the shared syntax, the durable truth value MUST normalize to `U` and the
statement update MUST be reported with the role change.

## 6. Required capabilities

Exact command spelling is provisional, but v1 MUST provide the following
capabilities.

### 6.1 Read and query

- List all statements and junctors.
- Show one statement or junctor with incoming and outgoing relations.
- Search statement text and IDs.
- List isolated statements.
- List connected components.
- Read a complete rooted structure.
- List non-counterpoint statements with no outgoing support, longest upstream
  support depth, and challenge state.
- Read a complete, stable, topologically ordered support ledger for a selected
  non-counterpoint statement.
- List defeats targeting or originating from an element.
- Validate under the workspace or Concludia profile.
- Read versioned machine guidance for statement identity and revision behavior.

### 6.2 Capture and edit

- Add a statement.
- Atomically add a non-empty batch of statements from versioned structured
  input and return an ordered caller-key-to-statement mapping.
- Edit statement text without changing its stable identity only with an
  explicit same-proposition assertion.
- Change truth and kind where valid.
- Rename, regenerate, or clear an optional statement slug without changing the
  statement ID or durable relations.
- Delete a statement with a dry-run plan showing incident relations and
  resulting orphans/components.
- Plan and apply material replacement between two existing non-counterpoint
  statement records through explicit per-relation choices and a state-bound
  apply token.
- Plan and apply deterministic whole-document renumbering through a state-bound
  old-to-new mapping that rewrites all modeled internal references atomically.

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
- Remove a counterpoint with a dry-run structural plan.

### 6.4 Export

- Export the complete rooted structure for a selected root.
- Reconcile roles in the exported artifact.
- Include all upstream justifications rather than selecting one proof path.
- Include attached defeat chains recursively.
- Validate the result under the Concludia profile.
- Refuse invalid export without modifying the workspace or output file.

## 7. CLI and JSON contract

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

Flags SHOULD be accepted consistently before or after positional arguments.
Mutation commands that can remove or cascade through relations SHOULD support
`--dry-run`.

Agent-facing guidance MUST instruct scripted callers not to predict generated
IDs across independent mutations. Callers SHOULD omit explicit IDs and consume
the assigned ID from each successful structured mutation result; any explicit
ID MUST equal the role-appropriate exact next ID.
Guidance MUST also state that attached counterpoints are removed explicitly
before deleting their target or an incident inference.

The version 1 batch-capture input contract is:

```json
{
  "schema_version": 1,
  "statements": [
    {
      "key": "caller-owned-correlation-key",
      "text": "Required statement text.",
      "id": "optional-explicit-id",
      "slug": "optional-explicit-slug",
      "truth": "T",
      "kind": "fact"
    }
  ]
}
```

Within a batch, `key` MUST be non-empty and unique but is not durable workspace
identity. `text` MUST be non-empty. Omitted `id` and `slug` values are generated
in batch order; omitted `truth` and `kind` values default to `T` and `fact`.
Unknown fields, unsupported schema versions, duplicate keys, or any invalid
resulting statement MUST reject the whole batch without writing. Successful and
dry-run output MUST preserve input order and return each key with its complete
assigned statement. A dry-run mapping is tentative; only the result of the
applied mutation is authoritative for later references.

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

V1 MUST use the existing Concludia `.arg` syntax as its common syntax. A
provisional metadata field identifies the relaxed workspace profile:

```text
meta profile="workspace"
```

The literal profile value remains renameable before the first public release.
It denotes validation semantics, not the product name.

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

The read-only TUI and a future web UI MUST use the same domain operations as the
CLI. They MUST NOT interpret or write the `.arg` file through independent
implementations.

Parity means:

- every durable mutation exposed in the UI is available through the CLI;
- every durable mutation exposed through the CLI can be initiated or
  meaningfully inspected in the UI;
- every fact shown visually is available in structured CLI output.

The initial TUI MUST remain read-only, expose Top, Statement Detail, and
Derivation Ledger views, and automatically reload valid external file changes
without replacing its last valid in-memory document with invalid contents.

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
- TUI mutation authoring, graphical graph maps, and TUI-only semantic queries.
- A universal format shared with Dagim.
- New `OR` or direct-support authoring UX.
- A new truth-propagation or defeat-adjudication engine.

## 13. Acceptance scenarios

V1 is complete when automated tests demonstrate at least:

1. A workspace containing isolated statements and multiple components parses,
   validates, and round-trips.
2. Two captured statements can be combined through a new `AND` junctor and
   target without manual file editing.
3. An existing premise promoted to a target becomes a lemma without changing
   ID.
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

## 14. Open decisions

Deferred decisions are tracked in [docs/open-questions.md](docs/open-questions.md).
