# Open Questions

This file contains only unresolved product and interoperability choices. Closed
decisions belong in ADRs, the specification, and tests rather than remaining as
historical notes here.

## Shared format ownership and versioning

- Does Concludia remain the owner of the common `.arg` specification, or should
  a stable shared format eventually live in a neutral package or repository?
- How should syntax version be declared independently from an artifact's
  content-oriented `meta version`?
- Should the shared compatibility corpus be copied between projects, vendored,
  or released as a separately versioned module?

Cludia must continue preserving ordinary Concludia `.arg` files losslessly
while these ownership and versioning questions remain open.

## Focused support authoring

- Does dogfooding justify focused `OR` creation, or are multiple complete `AND`
  justifications into one target sufficient?
- Should a future strict Concludia profile reject legacy one-source direct
  support? Any removal requires a migration and compatibility plan; Cludia must
  continue reading and preserving it in the meantime.

## Defeat representation and explanation

### One counterpoint with multiple targets

Concludia's internal registry can represent multiple defeat edges originating
from one counterpoint, but portable `.arg` syntax gives a counterpoint one
explicit target. Before Cludia exposes shared-counterpoint attachment, decide:

- how multiple targets are written and parsed in `.arg`;
- whether target order is meaningful;
- how removal and rooted export report each edge; and
- how old Concludia and Cludia versions diagnose the extended syntax.

This is a coordinated Concludia/format decision, not a Cludia-only command.

### Defeat-inclusive explanation

Current reads provide Statement Detail, complete and selected Ledger views,
rooted structure, and versioned grounded evaluation. Dogfooding has not yet
shown a concrete truth result those surfaces cannot explain adequately.

If such a case appears, decide whether a new explanation view should show:

- every reachable counterpoint or only defeats causally changing the result;
- inactive and `out` counterpoints for context;
- one accepted defeat chain or every accepted chain;
- branch-local facts alongside the statement's overall effective truth; and
- direct support, alternative justifications, and cycles in explanation order.

Do not implement an `--effective` proof filter until policies for true `OR`,
false `AND`, unknown paths, and outcome-neutral active justifications are
explicit. The tool must not silently present one path as the argument.

## Structured provenance

No current dogfood case requires source-document, citation, or passage objects
inside `.arg`. If one does, decide whether source information belongs:

- in statement text;
- as optional per-statement metadata;
- as distinct source and citation objects; or
- in an adjacent evidence-management tool.

Source reliability and statement truth must remain distinct concepts.

## Time, revision, and identity workflow

ADRs 0006, 0007, and 0009 establish external revision history, durable
proposition identities, explicit same-proposition edits, monotonic role-bearing
IDs, and reviewed whole-document renumbering. Remaining dogfood questions are:

- Which real edits remain difficult to classify as correction versus materially
  new proposition?
- Does any concrete workflow need named snapshots beyond files and Git refs?
- Does a concrete multi-workspace or synchronization workflow justify
  coordinated replacement or an opaque cross-system identity?

Do not add temporal, alias-history, or cross-file machinery without such a use
case.

## Interface scope

- Does repeated use of durable Top `J`/`K` reorder justify bounded, stale-safe
  undo? Any undo must apply a normal validated inverse mutation and refuse to
  overwrite intervening external edits.
- When a web UI becomes useful, should it use a local Go server with a browser
  interface or a separately built frontend served by the Go binary?

Any visual interface must remain local-first and use the shared domain and
validate-before-persist operations.

## Built-in generative assistance

Cludia currently relies on an external LLM using the CLI or future MCP surface.
Before adding product-owned retrieval, embeddings, background suggestions, or
model calls, identify:

- deterministic value an external agent cannot provide;
- privacy and local-model requirements;
- provider and credential policy;
- whether generated output is a proposal queue or executable command; and
- reproducibility and cost expectations.
