# Repository Guidance

## Required reading

Before changing behavior, read:

- `VISION.md`
- `SPEC.md`
- `docs/arg-workspace-profile.md`
- `docs/concludia-interoperability.md`
- the relevant ADRs under `docs/decisions/`

Treat those documents as authoritative for product intent. If code and the
specification disagree, report the discrepancy and resolve it explicitly; do
not silently redefine the format or semantics in implementation.

## Product invariants

- The product is local, file-first, and conversation-friendly.
- Disconnected components and isolated statements are valid workspace state.
- Authored inferences are multi-premise entailment-style sufficiency claims,
  not additive points in favor.
- V1 authors `AND` junctors with at least two sources.
- Existing Concludia `OR` junctors and direct supports must be read and
  preserved losslessly even when focused authoring does not create them.
- Undermines, undercuts, and counterpoints of counterpoints are first-class.
- An undercut does not retract its junctor. Contested or obsolete junctors may
  remain until explicit repair or removal; this is valid living-workspace state
  and does not by itself block saving or rooted export.
- Do not add confidence scores, weights, probabilities, or Bayesian machinery.
- Do not embed LLM calls or credentials in v1. External agents use the CLI or
  MCP.
- `.arg` scriptable JSON and file compatibility are public interfaces.
- Mutations must preserve structural validity and save atomically.
- Rooted export must include every upstream justification and attached defeat
  chain, then pass the Concludia validation profile.
- Do not couple this format to Dagim's `# dagim v1` format.

## Working from a checkout

- Installation should not be necessary. Use `go run ./cmd/cludia ...` or build
  the ignored local executable with `go build -o bin/cludia ./cmd/cludia`.
- `personal/` contains ignored local inquiry data and must not be committed.
- `dogfood/` contains tracked Cludia reasoning and Dagim planning artifacts;
  mutate them through their CLIs rather than editing their file syntax directly.
- Prefer `--json` when the tool is being used by a script or agent.
- Evaluate behavior through the CLI rather than editing `.arg` fixtures by hand
  unless the task is specifically about parsing or serialization.
- Preserve unrelated work and never stage or commit unless explicitly asked.

## Validation

Once implementation exists, ordinary changes must run:

```bash
go test ./...
go vet ./...
go build ./cmd/...
```

After completing a usable CLI slice, also build `bin/cludia` for local
dogfooding.

Format changes must also run compatibility and round-trip fixtures for:

- disconnected workspace files;
- normal Concludia `.arg` files;
- `OR` and legacy direct support preservation;
- undermines, undercuts, and recursive counterpoints;
- rooted Concludia export;
- cycle and atomic-write failures;
- versioned JSON contract output.

## Change discipline

- Exact command names may evolve before the first public release, but semantic
  operations and compatibility behavior require tests and documentation.
- Add or update an ADR for changes to profiles, topology, defeat semantics,
  LLM boundaries, or CLI/web parity.
- Keep the domain core independent of CLI, MCP, and web presentation layers.
- A visual UI must call the same durable operations as the CLI.
