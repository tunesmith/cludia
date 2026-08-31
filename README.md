# Cludia

`Cludia` is the project and repository name, and `cludia` is the command-line
binary. Version 1.0 uses `profile="cludia"` for its permissive inquiry profile.

Cludia is a local, file-first reasoning workbench for growing disconnected
observations into explicit arguments.

The central workflow is premise-up:

1. Capture short, truth-apt propositions; record unresolved hypotheses with
   truth `U` rather than storing questions as statements.
2. Accumulate them without requiring one connected graph or a conclusion.
3. Select statements that may work together.
4. Propose a new conclusion or lemma.
5. Persist the relationship only as a multi-premise sufficiency claim:
   `A AND B ... -> C`.
6. Record material challenges as undermines or undercuts.
7. Export the entire rooted structure around a mature conclusion as a
   Concludia argument.

Cludia 1.0 is intentionally a Go CLI operating on a readable
`.arg` file. An external LLM may use the JSON CLI during a conversation, but
the tool does not call an LLM itself. A local web interface is planned only
after conversational dogfooding identifies the right visual operations.

## Status

This repository is the project home for Cludia v1.0.0. The Go CLI implements the v1
file workflow: capture and editing, inspection and search, multi-premise
derivation and repair, defeat authoring, lifecycle operations, validation, and
rooted Concludia export. A terminal navigator adds deterministic Top, Statement
Detail, and Derivation Ledger views over the same query layer, plus focused
durable Top reordering. The project remains under active conversational
dogfooding after its first stable source release.

The executable's exit-status, stdout/stderr, structured-failure, collection,
and mutation-timing guarantees are documented in
[docs/cli-json.md](docs/cli-json.md).

Cludia 1.x guarantees file compatibility: it continues reading prior Cludia
`.arg` files and supported ordinary Concludia `.arg` files without silent loss.
Human CLI text, flags, and JSON response shapes may evolve during 1.x.
Incompatible JSON changes receive a schema-version bump and release-note
disclosure; older JSON schemas need not remain implemented. The current CLI
response schema is 2, batch input schema is 2, and evaluation schema is 1.

The principal decisions are:

- CLI and data file before a visual interface.
- Keep every durable mutation in clone-returning shared operations, with CLI
  files limited to parsing, presentation, and persistence decisions.
- Reuse Concludia's `.arg` syntax with a relaxed workspace validation profile.
- Allow disconnected and isolated statements in a workspace.
- Author only multi-premise `AND` inferences in the initial UX.
- Read and preserve ordinary Concludia constructs, including `OR`, legacy
  direct supports, and defeats.
- Include undermines, undercuts, and recursively chained counterpoints in v1.
- Preserve contested or obsolete junctors until an explicit repair or removal;
  their presence is valid in a living workspace and does not block export.
- Allocate canonical statement and junctor IDs monotonically, preserve deletion
  gaps, and compact them only through an explicit reviewed whole-document
  renumber.
- Reidentify an existing `P` target with the next `L` ID when derivation
  promotes it to a lemma, rewriting modeled references and reporting the
  mapping.
- Use statement sequence as one durable general order, with proof dependencies
  constraining Ledger presentation.
- Keep abductive discovery in the human/LLM conversation while persisting only
  asserted entailment-style inferences.
- Store truth only on leaf premises/counterpoints and calculate grounded
  three-valued effective truth across support and defeat structure.
- Do not add probabilistic scores, evidence weights, or Bayesian machinery.

## Illustrative session

```bash
cludia add case.arg --text "The footprints terminate at the garden wall."
cludia list case.arg --state isolated --json
cludia evaluate case.arg --json
cludia derive case.arg \
  --source footprints-at-wall \
  --source no-returning-prints \
  --target-text "The intruder crossed the garden wall."
cludia undercut case.arg J1 \
  --text "A vehicle could have collected the intruder at the wall."
cludia challenge case.arg crossed-wall \
  --text "The stated sources leave another route open."
cludia export case.arg --root crossed-wall --output crossed-wall.arg
```

The expected human workflow is conversational: an LLM reads the corpus through
`--json`, proposes a conclusion and audits missing premises in conversation,
then performs a mutation only after the user approves it.

Open a workspace in the terminal navigator:

```bash
cludia case.arg
```

The default Top view lists non-counterpoint statements with no outgoing support
in document order, with longest support depth. `!` means grounded counterpoints
changed the statement's truth somewhere in its upstream derivation; rebutted or
outcome-neutral counterpoints do not produce it.
Its title shows the selected row as `current of total` for a quick sense of place.
All statement text wraps without truncation. Enter follows the selected
statement into exact justification, challenge, and downstream-use detail; `f`
opens the complete support ledger to that statement. `j/k` selects rows;
capital `J/K` in Top moves the highlighted statement down/up in durable general
document order. Escape returns to the previous view, and `t` jumps directly to
Top. Valid external CLI or agent changes reload automatically; invalid contents
leave the last valid in-memory view intact. A stale reorder is refused when an
external change has altered the displayed Top adjacency.

## Installation

Install the current release with Homebrew:

```bash
brew install tunesmith/tap/cludia
cludia version
```

Or install the tagged source with Go 1.26.4 or newer:

```bash
go install github.com/tunesmith/cludia/cmd/cludia@v1.0.0
cludia version
```

From a source checkout:

```bash
go test ./...
go build -o bin/cludia ./cmd/cludia
bin/cludia version
```

`go install` writes to `GOBIN`, or to `$(go env GOPATH)/bin` when `GOBIN` is
unset. That directory must be on `PATH`. Checked-in source, tagged Go installs,
and the Homebrew formula all identify this release as `cludia v1.0.0`.

## Development

Installation is unnecessary. Validate the supplied workspace from a checkout:

```bash
go build -o bin/cludia ./cmd/cludia

bin/cludia validate examples/broken-window-workspace.arg
bin/cludia validate examples/broken-window-workspace.arg --json
bin/cludia validate examples/broken-window-conclusion.arg \
  --profile concludia
```

`bin/` is ignored and intended for local builds. To begin a workspace and add
more disconnected premises:

```bash
bin/cludia init inquiry.arg \
  --title "How should people work with Cludia?" \
  --text "Marlow completed Cludia's first implementation milestone."

bin/cludia add inquiry.arg \
  --text "Cludia supports direct CLI use and conversational agent use."

bin/cludia add-batch inquiry.arg --input statements.json --dry-run --json
bin/cludia add-batch --example

bin/cludia edit inquiry.arg P1 \
  --text "Marlow completed Cludia's first format-core milestone." \
  --same-proposition \
  --truth T \
  --kind fact

bin/cludia rename-slug inquiry.arg P1 --from-text
bin/cludia guidance --json

bin/cludia delete inquiry.arg P3 --dry-run

bin/cludia replace inquiry.arg L1 --with L2 \
  --retarget-source J2 \
  --remove-justification J1 \
  --delete-old --dry-run --json

bin/cludia renumber inquiry.arg --dry-run --json
bin/cludia renumber inquiry.arg --apply-token REVIEWED_TOKEN --json
bin/cludia normalize-truth inquiry.arg --dry-run --json

bin/cludia move-statement inquiry.arg L2 --before L1 --json

bin/cludia derive inquiry.arg \
  --source P1 \
  --source P2 \
  --target-text "Cludia's first two implementation increments are complete."

bin/cludia add-source inquiry.arg J1 --source P3
bin/cludia replace-source inquiry.arg J1 --from P2 --to P3 --dry-run
bin/cludia remove-source inquiry.arg J1 --source P3 --dry-run
bin/cludia remove-junctor inquiry.arg J1 --dry-run

bin/cludia undermine inquiry.arg P1 --text "The observation may be unreliable."
bin/cludia undercut inquiry.arg J1 --text "The sources leave another possibility open."
bin/cludia challenge inquiry.arg L1 --inference J1 --text "The sources do not suffice."
bin/cludia counterpoint inquiry.arg CP1 --text "The challenge is answered by later evidence."
bin/cludia remove-counterpoint inquiry.arg CP2 --dry-run

bin/cludia root inquiry.arg L1 --json
bin/cludia export inquiry.arg --root L1 --output conclusion.arg \
  --id conclusion-id --title "Published conclusion" --json

bin/cludia list inquiry.arg --state isolated
bin/cludia top inquiry.arg --challenged --limit 20 --offset 0 --json
bin/cludia ledger inquiry.arg L1 --json
bin/cludia ledger inquiry.arg L1 --inference J1 --json
bin/cludia show inquiry.arg P1 --relations
bin/cludia components inquiry.arg
bin/cludia component inquiry.arg P1 --json
bin/cludia search inquiry.arg "implementation milestone" --json
```

Every command shown above also supports `--json`. `init` refuses to overwrite
an existing file, and `add` validates the complete result before saving it
atomically.

Components are computed views rather than durable objects. `components` lists
every reasoning island in deterministic document order; `component` returns the
complete typed island containing a statement, statement slug, or junctor.
Support and defeat incidence both connect elements for grouping without
conflating their logical meanings.

`search` performs a case-insensitive substring match over statement IDs, slugs,
and text. Results preserve document order and report which fields matched.

`top` lists non-counterpoint statements with no outgoing support in document
order, including longest upstream support depth and propagated counterpoint
impact. Its `challenged` field and `!` marker mean that grounded defeats changed
the statement's base propagated truth. It accepts
`--challenged`, `--limit`, and `--offset` for ordered summary reads; these do not
alter `root`, whose contract remains the complete rooted structure. `ledger`
shows the complete support derivation to a selected statement in stable,
proof-local topological order: every source precedes its target, while premises
are delayed toward their first use and document order resolves equivalent
choices. Their human output preserves full statement text and their versioned
JSON is the shared read model for agents and the TUI.

`ledger --inference J1` narrows only the selected root statement to that exact
incoming junctor; each source still brings its complete upstream support. The
operation is a read-only branch view, not a persisted preferred proof. An
accepted undercut appears as `[undercut]`. If hidden root-level justifications
change the root's overall truth, the truth cell receives a compact `*` and a
footnote; the fixed-width truth column keeps statement and derivation columns
aligned. The TUI continues to use the default complete ledger.

`move-statement` moves one statement immediately before or after another in the
single durable general order. It accepts IDs or slugs, changes no identity or
relation, and validates and saves atomically. This order also influences search,
components, rooted export, and later reviewed `renumber` mappings.

`add-batch` schema 2 atomically creates new statements, focused `AND`
derivations, and typed defeats. New elements refer to one another by caller key;
relations may also name pre-existing durable IDs. The relation collections may
be empty for a statement-only transaction.
The result returns every statement key with its final role-consistent statement,
every derivation key with its generated junctor, and every created defeat. Any
invalid field, reference, relation, cycle, or final graph rejects the complete
transaction without writing or consuming IDs. Run `add-batch --help` to see the
contract or `add-batch --example` to print a complete transaction.

Schema 2 deliberately distinguishes `{"key":"finding"}` from
`{"id":"P17"}`. Keys name new elements in the same transaction; IDs name
elements that existed before it. Slugs and tentative generated IDs are not
relation references. A new statement targeted by a derivation receives its
final `L` ID directly rather than being created and immediately promoted from a
temporary `P` ID.

When a batch or `derive` creates an `AND` junctor with more than three sources,
the successful receipt warns that intermediate lemmas may improve clarity.
`add-source` gives the same advisory when it crosses from three to four. Large
junctors remain valid, and normal workspace reads do not repeat the warning.

Focused creation assigns role-appropriate canonical IDs from the exact next
values stored in `cludia-next-ids`. Automatic IDs increase monotonically;
deletion leaves gaps, and failed mutations or dry runs consume nothing. An
explicit ID is accepted only when it equals the relevant exact next value.
Existing custom IDs remain readable and survive ordinary round trips, but new
focused authoring does not create them.

Exact durable IDs take precedence over mutable statement slugs in every
unqualified reference. Imported slug/ID collisions remain readable with a
warning; focused authoring refuses to create new collisions.

When `derive --target` first justifies an existing premise, Cludia promotes it
with the next `L` ID, retires its former `P` ID, and atomically rewrites modeled
references. Structured output reports `previous_id` and `current_id`; external
references remain outside the checked scope. Use `derive --target-text` to
create a new lemma directly with an `L` ID.

Focused inference repair uses `add-source`, `replace-source`, `remove-source`,
and `remove-junctor`. Source editing applies only to `AND` junctors and must
leave at least two distinct sources. `replace-source` substitutes one source in
place and validates the result atomically instead of requiring an intermediate
three-source junctor. Source and junctor removal commands support `--dry-run`;
junctor removal refuses to orphan an attached inference undercut.

Defeat authoring uses `undermine` for premise scope, `undercut` for a specific
junctor and target, and `counterpoint` for a counterpoint of a counterpoint.
`challenge` routes to those same semantics from a premise, junctor,
counterpoint, or derived statement. It selects a derived statement's sole
incoming junctor but requires `--inference` when multiple justifications or
legacy direct support make the choice ambiguous.

A defeat is not a caution label. If its counterpoint is accepted by grounded
evaluation, it changes effective truth: an undermine falsifies its premise and
an undercut disables its exact inference edge. Use one only when accepting the
counterpoint really has that consequence. Absence of direct proof, a request
for caution, or residual uncertainty is not automatically a defeat; keep a
mere qualification in conversation or an adjacent note, or capture it as an
unattached truth-apt statement. `cludia guidance` exposes the same contract in
human and structured form.

`remove-counterpoint` is leaf-first, supports `--dry-run`, and refuses to remove
a statement that still has dependent counterpoints or support relations.

Text-changing `edit` requires `--same-proposition` and reports that declared
identity-continuity intent; truth- and kind-only edits do not require it.
`rename-slug` can explicitly rename, regenerate, or clear the optional current
slug while preserving ID and relations. `guidance` explains that statements
must be truth-apt, questions stay in conversation or adjacent notes, unresolved
propositions use `--truth U`, and confidence scores are outside the model. It
also exposes the use-case-neutral identity and replacement contract in human or
versioned JSON form. Its
scripted-authoring guidance requires callers to consume the returned
`statement.id` instead of predicting allocations across mutations; any
explicit ID must equal the exact next canonical value. Its deletion guidance
explains counterpoint-removal preconditions.

`delete` supports `--dry-run`, removes all incident relations rather than
silently changing an inference's sources, reports component changes and newly
isolated statements, delegates counterpoint deletion to `remove-counterpoint`,
and refuses to detach any counterpoint targeting the statement or an incident
junctor.

Material statement replacement is a two-phase `replace` operation between two
existing non-counterpoint records. The caller explicitly selects every
downstream `AND` source retarget, old incoming justification removal, recognized
root update, and old-record deletion. A dry run reports all incident relations,
affected defeats, blockers, and component changes and returns a plan token.
Application repeats the same choices with `--apply-token`; any intervening
workspace or choice change makes the token stale. The old statement remains by
default, and there is no retarget-all mode.

`renumber` is the sole numbering reset. Its dry run reports every statement and
junctor old-to-new mapping, root-metadata effects, resulting next-ID state, and
a token bound to the current file. Applying that token rewrites all modeled
references atomically and warns that references in Markdown, scripts, other
workspaces, prior exports, or published graphs remain outside Cludia's checked
scope. Slugs and non-identity content are preserved. The TUI has no renumber
hotkey.

`root` computes the complete upstream support closure and attached recursive
defeat chains for a selected non-counterpoint statement, reconciles roles, and
reports whether the result passes the strict Concludia profile. `export` writes
that artifact atomically only when strict validation succeeds and refuses to
overwrite an existing output file.

Local dogfooding workspaces may live under the ignored `personal/` directory so
private inquiry data is not committed accidentally.

Tracked project dogfooding artifacts live under `dogfood/`, including the
forward-looking Cludia discovery workspace and Ninth Room playtests.

The `check` command is an alias for `validate`. A file declaring
`meta profile="cludia"` selects the permissive Cludia profile by default;
otherwise validation defaults to the Concludia profile. Legacy files declaring
`profile="workspace"` remain readable and are rewritten to `profile="cludia"`
on their next successful durable save. Reads and failed operations never
rewrite them. `--profile` accepts `cludia` or `concludia`.

New Cludia files omit graph artifact `meta version`. If that optional metadata
is imported or explicitly authored, Cludia preserves it. It is a revision of
the graph artifact used by Concludia—not Cludia's software version and not a
version of the `.arg` syntax.

## Documents

- [VISION.md](VISION.md) explains the product motivation and philosophy.
- [SPEC.md](SPEC.md) defines the normative v1 behavior.
- [ROADMAP.md](ROADMAP.md) tracks forward-looking post-1.0 work.
- [CHANGELOG.md](CHANGELOG.md) records release changes and compatibility notes.
- [docs/arg-workspace-profile.md](docs/arg-workspace-profile.md) defines the
  implemented `.arg` Cludia profile.
- [docs/concludia-interoperability.md](docs/concludia-interoperability.md)
  specifies rooted export and round-trip behavior.
- [docs/conversational-workflow.md](docs/conversational-workflow.md) describes
  human/LLM/CLI collaboration.
- [docs/prior-art.md](docs/prior-art.md) records relevant systems and lessons.
- [docs/open-questions.md](docs/open-questions.md) tracks intentionally deferred
  choices.
- [docs/releasing.md](docs/releasing.md) defines the release and Homebrew
  publication procedure.
- [docs/decisions/](docs/decisions/) contains architectural decision records.
- [examples/](examples/) contains a disconnected workspace and its exported
  Concludia closure.
- [dogfood/](dogfood/) contains tracked reasoning and implementation-planning
  artifacts maintained through the Cludia and Dagim CLIs.
- [docs/use-cases/](docs/use-cases/) records optional workflow patterns without
  turning them into required format profiles or agent assumptions.

## Relationship to neighboring projects

Cludia borrows Dagim's local-file durability, stable IDs, atomic mutations,
fast CLI, and scriptable JSON. It does not reuse Dagim's format or completion
semantics.

Cludia borrows Concludia's statements, junctors, logical-force discipline, and
defeat model. Its workspace topology is more permissive: disconnected and
unfinished structures are normal rather than import errors.

## License

Cludia is licensed under
[GPL-3.0-or-later](LICENSE). The license covers the entire repository,
including code, documentation, examples, dogfood reasoning, playtests, and the
Ninth Room mystery content. Copyright 2026 KeenWorks.

Compatibility with the `.arg` format and Concludia does not license Concludia
source code or branding and does not imply that Concludia is part of this
GPL-covered program. See [CONTRIBUTING.md](CONTRIBUTING.md) for the current
contribution policy.
