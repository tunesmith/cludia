# ADR 0002: Reuse `.arg` syntax with a workspace validation profile

- Status: Accepted, with provisional profile naming
- Date: 2026-08-23

## Context

The proposed workspace and Concludia share statements, roles, truth tokens,
multi-premise junctors, undermines, undercuts, and recursive counterpoints. The
principal difference is topology: Concludia imports a rooted connected
argument, while inquiry begins with disconnected and isolated statements.

Concludia's implementation already separates DSL parsing from connectivity and
role validation.

## Decision

Use the existing `.arg` syntax and introduce a relaxed workspace validation
profile through provisional metadata such as:

```text
meta profile="workspace"
```

The literal profile value is not the product name and remains open to renaming
before implementation.

A valid Concludia file must be readable as a workspace. A workspace may be
disconnected and is promoted to Concludia only through explicit rooted export.

The tool reads and preserves `OR`, direct supports, and all defeat forms even
though focused v1 authoring creates only multi-premise `AND` junctors.

## Consequences

- Mature reasoning can become Concludia without a second translation DSL.
- Existing Concludia fixtures can serve as compatibility tests.
- Workspace files may parse in Concludia yet fail its stricter validator; error
  messages and profile documentation must make that distinction clear.
- The Go implementation must follow a language-neutral conformance contract
  rather than copying incidental Scala behavior.
- Unknown metadata and unfocused constructs cannot be silently dropped.

## Alternatives considered

- New workspace-only syntax: cleaner topology naming, but duplicates most of
  `.arg` and increases conversion risk.
- Make Concludia itself accept disconnected graphs: expands its product and
  persistence invariants before the new workflow is proven.
- Universal graph format shared with Dagim: rejected because Dagim's blocking
  and completion semantics are materially different.

