# ADR 0002: Reuse `.arg` syntax with the Cludia validation profile

- Status: Accepted
- Date: 2026-08-23
- Amended: 2026-08-30 for the Cludia 1.0 release

## Context

The proposed workspace and Concludia share statements, roles, truth tokens,
multi-premise junctors, undermines, undercuts, and recursive counterpoints. The
principal difference is topology: Concludia imports a rooted connected
argument, while inquiry begins with disconnected and isolated statements.

Concludia's implementation already separates DSL parsing from connectivity and
role validation.

## Decision

Use the existing `.arg` syntax and identify the relaxed Cludia validation
profile with stable metadata:

```text
meta profile="cludia"
```

The pre-1.0 value `profile="workspace"` remains an input alias. Reads preserve
the file unchanged, dry runs report the proposed metadata update, and the next
successful durable save rewrites it atomically to `profile="cludia"`. CLI
profile overrides use only the canonical names `cludia` and `concludia`.

New Cludia workspaces do not add graph artifact `meta version`. Imported or
explicitly authored graph-version metadata remains preserved; it is distinct
from both the Cludia software version and any future `.arg` syntax version.

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
