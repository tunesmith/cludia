# Cludia

`Cludia` is the permanent project and repository name. The final binary name
and workspace-profile identifier remain open decisions.

Cludia is a proposed local, file-first reasoning workbench for growing
disconnected observations into explicit arguments.

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

This repository currently contains the design packet, not an implementation.
It is the permanent project home from which implementation will begin.

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

Command names are provisional:

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

## Documents

- [VISION.md](VISION.md) explains the product motivation and philosophy.
- [SPEC.md](SPEC.md) defines the normative v1 behavior.
- [ROADMAP.md](ROADMAP.md) sequences implementation and dogfooding.
- [docs/arg-workspace-profile.md](docs/arg-workspace-profile.md) defines the
  proposed `.arg` workspace profile.
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

## Relationship to neighboring projects

Cludia borrows Dagim's local-file durability, stable IDs, atomic mutations,
fast CLI, and scriptable JSON. It does not reuse Dagim's format or completion
semantics.

Cludia borrows Concludia's statements, junctors, logical-force discipline, and
defeat model. Its workspace topology is more permissive: disconnected and
unfinished structures are normal rather than import errors.
