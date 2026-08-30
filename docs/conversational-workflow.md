# Conversational Workflow

## Interaction direction

The first product is designed for this direction:

```text
human <-> external LLM -> CLI -> `.arg` workspace
```

The user talks with an LLM that can inspect and operate the tool. The user need
not keep the visual graph open or manually translate every conversational
decision into file edits.

The inverse direction is deliberately excluded from v1:

```text
human -> tool -> embedded LLM provider
```

## Why embedded LLM integration was considered

Once the core loop was described, several possible built-in features suggested
themselves:

- `discover` could send the corpus to a model and propose compatible statement
  sets;
- `combine` could generate candidate conclusions;
- `audit` could search for missing premises or counterexamples;
- embeddings could retrieve relevant statements from a large corpus;
- a background process could surface new combinations as statements arrive;
- non-agent users could receive generative help without bringing their own LLM
  conversation.

Those could eventually be valuable. They are not needed to test the product's
central hypothesis, and embedding them early would add:

- API keys, providers, model selection, cost, and network behavior;
- privacy questions for potentially sensitive investigations;
- prompt and model output semantics inside the product contract;
- duplicated conversation UX when the user already has an agent;
- pressure to treat generated suggestions as tool results rather than
  proposals requiring judgment.

Therefore v1 keeps the tool deterministic and model-agnostic. The external LLM
is the generative layer.

## Core loop

### 1. Capture

Cludia captures truth-apt propositions, not questions or prompts. Questions stay
in conversation or adjacent notes until the human and agent can formulate a
proposition worth recording. The user may mention an observation, and the agent
may propose an atomic formulation:

> You said that the footprints stop at the garden wall. Shall I capture “The
> footprints terminate at the garden wall” as a true factual premise?

After approval:

```bash
cludia add case.arg \
  --text "The footprints terminate at the garden wall." \
  --json
```

An unresolved hypothesis or disputed proposition is still truth-apt, but it
should be captured explicitly as unknown rather than receiving the default true
token:

```bash
cludia add case.arg \
  --text "The intruder may have left by vehicle." \
  --truth U --json
```

Cludia does not assign confidence scores or probabilities. The human supplies
truth state only for leaf premises and leaf counterpoints. Cludia calculates
grounded effective truth for sourced statements and typed challenges without
persisting propagated values.

Use `cludia evaluate case.arg --json` for the complete versioned overlay.
Existing imported sourced statements with stored `T` or `F` are evaluated from
their sources and diagnosed until reviewed through `normalize-truth --dry-run`
and `--apply-token`.

For scripted or agent-driven capture, a generated ID becomes authoritative
only in the successful mutation result. An agent must not predict a sequence of
future IDs across independent mutations: a rejected add does not reserve its
candidate ID. Omit `--id` and read `statement.id` from each successful
`add --json` response before using it in a later command. If an explicit ID is
provided, focused authoring accepts it only when it is the role-appropriate
exact next canonical ID.

When several statements and their relationships are already known, prefer one
atomic schema 2 transaction over a sequence whose later references depend on
predicted allocations. For example, `installment.json` may contain:

```json
{
  "schema_version": 2,
  "statements": [
    {"key": "phase-status", "text": "The migration phase is complete."},
    {"key": "ticket-42061", "text": "42061 blocks the next phase."},
    {"key": "next-phase", "text": "The next migration phase cannot begin."}
  ],
  "derivations": [
    {
      "key": "phase-block",
      "sources": [{"key": "phase-status"}, {"key": "ticket-42061"}],
      "target": {"key": "next-phase"}
    }
  ],
  "defeats": []
}
```

Then preview or apply it with:

```bash
cludia add-batch case.arg --input installment.json --dry-run --json
cludia add-batch case.arg --input installment.json --json
```

The result returns final statement mappings, generated junctor mappings, and
defeats. `{"key":"..."}` references a new element in this transaction;
`{"id":"P17"}` references a durable element already in the workspace. Slugs
are intentionally not accepted as transaction references. If any entry or the
complete resulting graph is invalid, nothing is written and no generated ID is
consumed. Treat dry-run mappings as tentative and use the applied result later.

Because schema 2 sees the complete topology before allocating IDs, a new
derivation target is created directly as a lemma with an `L` ID. A
statement-only transaction uses the same schema with empty `derivations` and
`defeats` arrays.

Ordinary deletion leaves numeric gaps because Cludia does not reuse retired IDs
during focused authoring. When a user explicitly wants compact labels, first
review the complete mapping from `renumber --dry-run`; apply it only with the
returned state-bound token and review unknown external references afterward.

### 2. Inspect

The agent queries the file rather than asking the user to restate it:

```bash
cludia list case.arg --state isolated --json
cludia show case.arg footprints-at-wall --relations --json
cludia component case.arg footprints-at-wall --json
cludia top case.arg --challenged --limit 20 --offset 0 --json
```

### 3. Propose

The agent performs generative reasoning in conversation:

> P1 and P2 suggest that the intruder crossed the wall, but they do not entail
> it: a vehicle could have collected the intruder there. A bridge premise would
> be that crossing the wall or retracing the path were the only physically
> possible exits. Do you accept that statement?

The LLM may have reached this proposal abductively. No inference mode needs to
appear in the CLI.

### 4. Audit

Before mutation, the human and LLM ask:

- Are all proposed source statements true as written?
- If all sources hold, can the target still be false?
- What hidden premise would that counterexample reveal?
- Is every selected source necessary?
- Is the target merely a restatement?
- Is there a material undermine or undercut worth preserving?

### 5. Persist

After approval:

```bash
cludia derive case.arg \
  --source P1 \
  --source P2 \
  --source P3 \
  --target-text "The intruder crossed the garden wall." \
  --json
```

The mutation result reports the new target, junctor, role changes, and component
changes. More than three sources remains valid but produces
`concludia_junctor_sources_many`, recommending intermediate lemmas while the
reasoning is still easy to factor. Batch derivations give the same warning, and
`add-source` emits it when crossing from three to four sources.

### 6. Challenge or repair

Defeats are semantic, not annotations. Before attaching one, ask what accepting
the counterpoint would do:

- use an undermine only if it makes a premise false or materially out of scope;
- use an undercut only if the stated sources no longer suffice for that target;
- use a counterpoint of a counterpoint only if it defeats the earlier objection.

No direct eyewitness, a request for caution, or residual uncertainty may be an
important qualification without defeating the argument. Keep such a caveat in
conversation or an adjacent note, or capture it as an unattached truth-apt
statement. If a real objection remains:

```bash
cludia undercut case.arg J1 \
  --text "A vehicle could have collected the intruder at the wall." \
  --json
```

If the objection merely exposes a missing premise, the preferred action is to
repair the inference rather than using a counterpoint to excuse it. The old
junctor and its undercut may remain while that work is in progress, or even
alongside a repaired replacement until the author explicitly removes it. This
is valid living-workspace state, not a save or export error.

For example, suppose `J1(P1, P2) -> C1` is undercut because a bridge premise is
missing. A repair can proceed in either of two explicit ways:

- add the bridge premise to `J1`, changing the sufficiency claim; or
- create `J2(P1, P2, P3) -> C1`, then remove `J1` when its history is no longer
  useful in the active workspace.

Until removal, `J1` still means that its sources were authored as sufficient;
the undercut records that this claim is contested or presently rejected. The
tool preserves both claims and does not decide which one is semantically right.

The `challenge` convenience command preserves those distinctions while routing
from the selected element. It undermines a premise, undercuts a junctor,
counterpoints a counterpoint, and undercuts the sole incoming junctor of a lemma
or conclusion. When a derived statement has multiple incoming junctors, the
caller must select one explicitly:

```bash
cludia challenge case.arg crossed-wall \
  --inference J1 \
  --text "The sources leave another physically possible exit." \
  --json
```

When one source of an `AND` junctor was selected incorrectly, replace it as one
atomic operation rather than temporarily adding a third source and then
removing the old one:

```bash
cludia replace-source case.arg J1 \
  --from wrong-source \
  --to corrected-source \
  --dry-run --json
```

The command preserves the source position, validates the complete result, and
writes nothing when replacement would create a duplicate, self-support, cycle,
or other invalid structure.

### 7. Export

When the structure is mature:

```bash
cludia export case.arg --root crossed-wall \
  --output crossed-wall.arg --json
```

The agent can then import the result into Concludia through its normal tools.

## Approval boundary

Read-only inspection is safe to perform autonomously during a conversation.
Semantic mutations should normally be proposed before persistence because:

- atomic wording matters;
- truth is supplied by the user, not inferred from model confidence;
- a junctor asserts sufficiency;
- deleting or repairing a relation may change a larger rooted structure.

Before changing statement text, the agent must distinguish reformulation from
material replacement. `edit --same-proposition` asserts continuity of one
proposition record. A materially different proposition receives a new ID and
each affected relation is audited explicitly. Slug refactoring is a separate
alias-only operation and must not be presented as a semantic edit. These rules
apply regardless of how the user organizes one or more workspaces.

Before deleting a statement or inference with an attached counterpoint, remove
the counterpoint through `remove-counterpoint`; Cludia refuses to detach it
implicitly. Use `delete --dry-run --json` or the applicable relation-removal
dry run to inspect structural effects before the destructive mutation.

For a materially different proposition, first add and justify the new record.
Then use `replace` to plan only the relation changes that have been semantically
reviewed:

```bash
cludia replace case.arg old-finding --with corrected-finding \
  --retarget-source J7 \
  --remove-justification J3 \
  --delete-old --dry-run --json
```

The dry run reports all incident relations and returns a token only when the
plan is applicable. Apply by repeating the exact choices with
`--apply-token TOKEN`. Any intervening workspace change invalidates the token.
Unselected relations block deletion, the old record is retained by default,
and Cludia never assumes that all relations about the old proposition remain
valid for the new one.

The user may explicitly authorize broader mutation, but the tool itself should
remain transparent through dry-run and structured mutation results.

## Corpus-wide discovery

The initial CLI does not need a model-powered `discover` command. An external
agent can request the corpus or filtered candidates and reason over them.

As corpora grow, deterministic retrieval commands may be added without
embedding a model:

- full-text search;
- tags or user-defined collections, if later specified;
- components and isolated statements;
- statements sharing words or explicit references;
- recently added or unused statements;
- statements near a chosen rooted structure.

Only after dogfooding should the project decide whether embeddings or built-in
generation offer enough value to justify a new boundary.

## MCP

MCP is a later transport over the same domain operations. It must not introduce
a second data model or hidden mutations. The CLI remains sufficient for local
agent use and scripting.
