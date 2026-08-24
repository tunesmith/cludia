# Roadmap

This roadmap optimizes for learning the reasoning workflow before investing in
a graphical interface.

## Milestone 0: Design packet

- Agree on vision, v1 scope, and non-goals.
- Record `.arg` workspace-profile and Concludia-closure decisions.
- Provide paired example files.
- Establish `Cludia` as the permanent project and repository name.
- Choose the final binary and workspace-profile names.

Exit criterion: a new conversation in this repository can explain the product
and begin implementation without relying on the original brainstorming
transcript.

## Milestone 1: Go format core

Status: implemented on 2026-08-24; pending review and commit.

- Create the Go module and command entry point.
- Implement shared `.arg` parsing for statements, metadata, `AND`, `OR`, direct
  supports, and all defeat scopes.
- Implement lossless or explicitly diagnosed serialization.
- Implement workspace and Concludia validation profiles.
- Implement stable-ID generation and atomic writes.
- Add fixture-based compatibility tests using Concludia `.arg` examples.

Exit criterion: the paired files in `examples/` parse and validate under their
expected profiles, and ordinary Concludia fixtures round-trip.

## Milestone 2: Complete file CLI

- Add/list/show/search/edit statements.
- List components and isolated statements.
- Create, edit, and remove multi-premise `AND` inferences.
- Add and remove undermines, undercuts, and counterpoints of counterpoints.
- Add dry-run deletion and structural diagnostics.
- Version the JSON contract and add contract tests.

Exit criterion: every v1 durable operation is usable without manual file
editing and has equivalent structured output.

## Milestone 3: Rooted Concludia export

- Compute the complete rooted support closure.
- Include every attached recursive defeat chain.
- Reconcile roles and root metadata.
- Validate with the Concludia profile.
- Add conformance tests shared conceptually with Concludia.

Exit criterion: `broken-window-workspace.arg` exports reproducibly to the
Concludia example and fails atomically when the requested structure is invalid.

## Milestone 4: Conversational dogfooding

- Use the CLI through ordinary LLM conversations on real non-sensitive
  workspaces.
- Grow at least one workspace to 50–200 statements.
- Track repeated queries, awkward mutations, missing diagnostics, and semantic
  failure patterns.
- Revise the spec only through explicit decisions and compatibility notes.

Questions to answer through use:

- Are IDs pleasant enough to discuss conversationally?
- How often does a proposed inference reveal a missing premise?
- Which views are repeatedly reconstructed in prose?
- Are component and defeat traversals sufficient?
- Does the role model create churn?
- When does a visual interface become materially necessary?

Exit criterion: a short dogfooding report identifies the minimum visual UX and
any necessary CLI contract changes.

## Milestone 5: MCP adapter

- Expose the stable CLI/domain operations through MCP.
- Keep MCP thin: no separate model, mutations, or hidden LLM calls.
- Preserve explicit user-approval and dry-run boundaries.

Exit criterion: an MCP client and the CLI produce equivalent reads and
mutations against the same fixtures.

## Milestone 6: Minimal local web UI

- Statement inbox/list and search.
- Multi-selection and combine/justify flow.
- Target and missing-premise editing.
- Component and defeat inspection.
- Rooted export.
- Operation and information parity tests against the shared domain core.

Exit criterion: the web UI improves the workflows observed during dogfooding
without adding a second persistence or semantics implementation.

## Deferred

- Structured provenance and source documents.
- Model-assisted background discovery or embeddings.
- New `OR` and direct-support authoring.
- Collaboration and hosted publication.
- TUI and native application packaging.
- Advanced timelines, entities, maps, and investigation ingestion.
- Any universal graph format spanning Dagim.
