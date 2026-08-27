# Cludia

`Cludia` is the permanent project and repository name, and `cludia` is the
command-line binary. The workspace-profile identifier remains provisional.

Cludia is a local, file-first reasoning workbench for growing disconnected
observations into explicit arguments.

The central workflow is premise-up:

1. Capture short statements believed to be true.
2. Accumulate them without requiring one connected graph or a conclusion.
3. Select statements that may work together.
4. Propose a new conclusion or lemma.
5. Persist the relationship only as a multi-premise sufficiency claim:
   `A AND B ... -> C`.
6. Record material challenges as undermines or undercuts.
7. Export the entire rooted structure around a mature conclusion as a
   Concludia argument.

The first implementation is intentionally a Go CLI operating on a readable
`.arg` file. An external LLM may use the JSON CLI during a conversation, but
the tool does not call an LLM itself. A local web interface is planned only
after conversational dogfooding identifies the right visual operations.

## Status

This repository is the permanent project home. The Go CLI implements the v1
file workflow: capture and editing, inspection and search, multi-premise
derivation and repair, defeat authoring, lifecycle operations, validation, and
rooted Concludia export. A read-only terminal navigator adds deterministic Top,
Statement Detail, and Derivation Ledger views over the same query layer. The
project is pre-release and remains under active conversational dogfooding. Its
`.arg` syntax and versioned JSON output remain compatibility-sensitive
interfaces even before the first public release.

The principal decisions are:

- CLI and data file before a visual interface.
- Reuse Concludia's `.arg` syntax with a relaxed workspace validation profile.
- Allow disconnected and isolated statements in a workspace.
- Author only multi-premise `AND` inferences in the initial UX.
- Read and preserve ordinary Concludia constructs, including `OR`, legacy
  direct supports, and defeats.
- Include undermines, undercuts, and recursively chained counterpoints in v1.
- Preserve contested or obsolete junctors until an explicit repair or removal;
  their presence is valid in a living workspace and does not block export.
- Keep abductive discovery in the human/LLM conversation while persisting only
  asserted entailment-style inferences.
- Do not add probabilistic scores, evidence weights, or Bayesian machinery.

## Illustrative session

```bash
cludia add case.arg --text "The footprints terminate at the garden wall."
cludia list case.arg --state isolated --json
cludia derive case.arg \
  --source footprints-at-wall \
  --source no-returning-prints \
  --target-text "The intruder crossed the garden wall."
cludia undercut case.arg J1 \
  --text "A vehicle could have collected the intruder at the wall."
cludia export case.arg --root crossed-wall --output crossed-wall.arg
```

The expected human workflow is conversational: an LLM reads the corpus through
`--json`, proposes a conclusion and audits missing premises in conversation,
then performs a mutation only after the user approves it.

Open a workspace in the read-only terminal navigator:

```bash
cludia case.arg
```

The default Top view lists non-counterpoint statements with no outgoing support
in document order, with longest support depth and `!` on challenged statements.
All statement text wraps without truncation. Enter follows the selected
statement into exact justification, challenge, and downstream-use detail; `f`
opens the complete support ledger to that statement. `j/k` navigates and
Escape returns to the previous view. Press `t` from anywhere to return directly
to Top. Valid external CLI or agent changes reload automatically; invalid
contents leave the last valid in-memory view intact. The TUI never writes the
workspace.

## Install from source

Cludia currently requires Go 1.26.4 or newer. From a checkout:

```bash
go test ./...
go install ./cmd/cludia
cludia version
```

`go install` writes to `GOBIN`, or to `$(go env GOPATH)/bin` when `GOBIN` is
unset. That directory must be on `PATH`. An untagged development build reports
its version as `dev`.

Until a public release exists, clone the repository and install from the
checkout rather than relying on `go install ...@latest`.

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
bin/cludia counterpoint inquiry.arg CP1 --text "The challenge is answered by later evidence."
bin/cludia remove-counterpoint inquiry.arg CP2 --dry-run

bin/cludia root inquiry.arg L1 --json
bin/cludia export inquiry.arg --root L1 --output conclusion.arg \
  --id conclusion-id --title "Published conclusion" --json

bin/cludia list inquiry.arg --state isolated
bin/cludia top inquiry.arg --json
bin/cludia ledger inquiry.arg L1 --json
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
order, including longest upstream support depth and challenge state. `ledger`
shows the complete support derivation to a selected statement in stable
topological order. Their human output preserves full statement text; their
versioned JSON is the shared read model for agents and the read-only TUI.

`add-batch` atomically captures multiple statements from a versioned JSON input
file. Each input item has a required unique caller `key` and statement `text`,
plus optional `id`, `slug`, `truth`, and `kind`. The result preserves input order
and maps every caller key to its complete assigned statement; any invalid item
rejects the whole batch without writing or consuming IDs.

Focused inference repair uses `add-source`, `replace-source`, `remove-source`,
and `remove-junctor`. Source editing applies only to `AND` junctors and must
leave at least two distinct sources. `replace-source` substitutes one source in
place and validates the result atomically instead of requiring an intermediate
three-source junctor. Source and junctor removal commands support `--dry-run`;
junctor removal refuses to orphan an attached inference undercut.

Defeat authoring uses `undermine` for premise scope, `undercut` for a specific
junctor and target, and `counterpoint` for a counterpoint of a counterpoint.
`remove-counterpoint` is leaf-first, supports `--dry-run`, and refuses to remove
a statement that still has dependent counterpoints or support relations.

Text-changing `edit` requires `--same-proposition` and reports that declared
identity-continuity intent; truth- and kind-only edits do not require it.
`rename-slug` can explicitly rename, regenerate, or clear the optional current
slug while preserving ID and relations. `guidance` exposes the use-case-neutral
identity and replacement contract in human or versioned JSON form. Its
scripted-authoring guidance requires callers to provide explicit IDs or consume
the returned `statement.id` instead of predicting allocations across
mutations, and its deletion guidance explains counterpoint-removal
preconditions.

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

`root` computes the complete upstream support closure and attached recursive
defeat chains for a selected non-counterpoint statement, reconciles roles, and
reports whether the result passes the strict Concludia profile. `export` writes
that artifact atomically only when strict validation succeeds and refuses to
overwrite an existing output file.

Local dogfooding workspaces may live under the ignored `personal/` directory so
private inquiry data is not committed accidentally.

Tracked project dogfooding artifacts live under `dogfood/`. The initial pair is
the Cludia reasoning workspace and its corresponding Dagim implementation plan.

The `check` command is an alias for `validate`. A file declaring
`meta profile="workspace"` selects the workspace profile by default; otherwise
validation defaults to the Concludia profile. `--profile` overrides either
choice.

## Documents

- [VISION.md](VISION.md) explains the product motivation and philosophy.
- [SPEC.md](SPEC.md) defines the normative v1 behavior.
- [ROADMAP.md](ROADMAP.md) sequences implementation and dogfooding.
- [docs/arg-workspace-profile.md](docs/arg-workspace-profile.md) defines the
  implemented `.arg` workspace profile.
- [docs/concludia-interoperability.md](docs/concludia-interoperability.md)
  specifies rooted export and round-trip behavior.
- [docs/conversational-workflow.md](docs/conversational-workflow.md) describes
  human/LLM/CLI collaboration.
- [docs/prior-art.md](docs/prior-art.md) records relevant systems and lessons.
- [docs/open-questions.md](docs/open-questions.md) tracks intentionally deferred
  choices.
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
