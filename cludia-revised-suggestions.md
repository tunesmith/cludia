# Revised Cludia Suggestions After Mystery Play Test

## Context

These suggestions come from using Cludia to track a six-installment fair-play
mystery. The resulting workspace contained 333 statements, 93 junctors, and 54
defeats. The current installed build reported:

- module: `github.com/tunesmith/cludia`
- revision: `f01e4cf85d689716ebb4954377b400a1471c2f2f`
- build state: dirty
- evaluation schema: version 1, grounded mode

The current truth evaluator was tested separately from the old mystery graph
with a fresh minimal workspace.

## Disposition summary

| # | Original suggestion | Revised disposition |
|---|---|---|
| 1 | Batch relations | Implemented as atomic batch input schema version 2 |
| 2 | Global rebuttal / shared counterpoint | Defer: Concludia's registry permits multiple edges, but the shared `.arg` syntax permits one target per counterpoint |
| 3 | Transitive challenge status | Resolved in the current build; do not forward |
| 6 | More actionable diagnostics | Forward a narrowed diagnostic and authoring-guidance request |
| 7 | Earlier large-junctor warnings | Forward |
| 8 | Compact proof slices | Clarify as selected-derivation or effective-support ledger views; forward for analysis |

## 1. Batch statements and relations atomically

### Implemented

Batch schema version 2 now adds statements and creates relations among them in
one validated atomic operation. It is the sole supported batch contract.

The existing `add-batch` caller-key mapping worked well and correctly avoided
predicting generated IDs. The main friction was that each installment required
one batch statement import followed by many separate `derive`, `undercut`,
`challenge`, and `counterpoint` commands.

An illustrative schema shape:

```json
{
  "schema_version": 2,
  "statements": [
    {"key": "puncture", "text": "A puncture marked the victim's thumb."},
    {"key": "groove", "text": "A dyed groove was present in the clasp."},
    {"key": "needle-theory", "text": "The clasp caused the puncture."}
  ],
  "derivations": [
    {
      "key": "needle-inference",
      "sources": [{"key": "puncture"}, {"key": "groove"}],
      "target": {"key": "needle-theory"}
    }
  ],
  "defeats": []
}
```

New elements use typed `{"key":"..."}` references, while existing workspace
elements use typed durable `{"id":"P1"}` references. Applied results return
final statement and junctor mappings plus created defeats. Dry-run mappings
remain tentative. New derivation targets receive final `L` IDs directly.

This is an ergonomic extension, not a relaxation of statement identity or
relation semantics.

## 2. Reusing one counterpoint at multiple explicit defeat locations

### Clarification of the original request

The original phrase "attack the proposition" mixed two different ideas:

1. Reuse one true counterpoint statement as the origin of more than one
   explicit defeat edge.
2. Rebut a derived proposition independently of its incoming inference paths.

The first is a clear CLI request. The second is a semantic question for
Concludia rather than an obviously missing Cludia command.

### Confirmed CLI limitation

The current CLI can create a new counterpoint targeted at one premise,
counterpoint, junctor, or selected incoming inference. It does not expose a way
to attach an existing counterpoint to a second explicit target.

This mattered when one later fact was relevant to two incoming inferences. The
only available workflow created two nearly identical counterpoint statements.

If the Concludia model permits one statement to originate multiple defeat
edges, Cludia could expose an operation resembling:

```text
cludia attach-counterpoint FILE COUNTERPOINT --inference JUNCTOR
```

The operation should remain explicit. It should not automatically attack every
incoming inference merely because they share a target.

### Open semantic question: conclusion-level rebuttal

Sometimes new evidence contradicts a conclusion regardless of how it was
derived. For example:

- `There is exactly one false collection object.`
- `The Psalter is that false object.`
- therefore `The parcel book is not the announced false object.`

That looks like rebuttal of the conclusion, not necessarily an undercut of
each supporting inference or an attack on one upstream premise.

If Concludia already models proposition-level rebuttal of a lemma or
conclusion, Cludia may need to expose it. If Concludia intentionally requires
all defeats to land on premises or inference edges, no new semantic primitive
should be invented in Cludia; the author should instead locate the appropriate
explicit defeat edges.

Forward this as a core-model/API question, not as a settled requirement for an
automatic "rebut all incoming arguments" command.

## 3. Active challenge presentation

### Minimal test result

A fresh graph contained:

```text
AND#J1(P1, P2) -> L1
CP1 -> inference J1:target L1
CP2 -> counterpoint CP1
```

After adding `CP1`, `top --challenged` displayed:

```text
L1! F · derived
```

After adding `CP2`, grounded evaluation correctly changed the result to:

```text
effective_truth: T
CP1 acceptance: out
CP2 acceptance: in
```

In the earlier build, `top` still displayed `L1! T · derived`. In the current
build, it correctly displays:

```text
L1  T
```

`top --challenged` is empty, and the JSON `ledger` row reports:

```json
{
  "challenged": false,
  "effective_truth": "T"
}
```

### Disposition

This request is resolved. Do not forward it. Truth propagation, the `!`
marker, the `challenged` JSON field, and the `top --challenged` filter now agree
on active accepted defeats.

This improvement partially helps suggestion 8 because a ledger now states the
current status correctly. It does not provide branch selection or show the
counterpoint subgraph, so the narrower suggestion 8 remains open.

## 6. More actionable diagnostics and defeat-authoring guidance

The recent truth work changes this recommendation substantially.

### A. Improve `truth_assignment_nonleaf`

Current result when assigning `F` to a sourced lemma:

```text
ERROR [truth_assignment_nonleaf] L1: truth can only be assigned to leaf premises and leaf counterpoints; L1 has incoming support
```

Suggested addition:

```text
Derived truth is calculated. To dispute this result, challenge an upstream
premise or undercut an incoming inference; use `evaluate` to inspect effective
truth. Use `normalize-truth` only to remove obsolete stored truth from sourced
statements.
```

This keeps the leaf-only invariant intact while teaching the correct action.

### B. Explain that a caveat is not automatically a defeat

The new evaluator exposed authoring mistakes in the mystery workspace. Several
statements such as:

> No witness saw the final push, although the converging evidence identifies
> Gideon.

had been entered as challenges. Grounded evaluation correctly treated those
counterpoints as active defeats, which made their target inferences false and
propagated falsity into the final conclusion.

The problem is the authored graph, not the evaluator. That sentence expresses
residual uncertainty about direct observation; it does not establish that the
inference is invalid.

Add guidance along these lines:

> Create a defeat only when accepting the counterpoint would make a premise
> false or out of scope, disable the inference from its sources to its target,
> or defeat another counterpoint. Mere absence of direct proof, a request for
> caution, or an acknowledged residual uncertainty is not automatically a
> defeat. Keep such qualifications in adjacent notes or state a separate
> truth-apt proposition without attaching a defeat edge.

This guidance becomes important now that defeat edges have effective semantic
consequences.

### C. Concrete modeling diagnosis from the Gideon graph

The final Gideon conclusion currently evaluates `F`, despite the prose solution
being convincing. Inspection shows that this is principally an authoring
problem rather than an evaluator problem.

Several counterpoints were true sentences but did not actually defeat the
inferences to which they were attached:

- `CP52` said no witness saw Gideon push Ruth, while also acknowledging that
  converging evidence identified him. It was attached as an undercut of the
  inference that Gideon pushed Ruth.
- `CP53` said no unique tool mark proved Gideon personally performed every
  binding step, while acknowledging that the evidence identified him as
  responsible. It was attached as an undercut of parcel responsibility.
- `CP27` said the unidentified freight-yard man could have learned the account
  theory from someone else. It was attached to the more limited inference that
  the man had confused Row 17 with an account, which Ruth's words already
  supported.
- `CP41` observed that bearer drafts did not identify the ultimate recipient.
  That is a legitimate limitation on proving personal receipt of proceeds, but
  the root made that stronger embezzlement claim indispensable to proving
  murder responsibility.

Because these counterpoints are accepted, the evaluator correctly disables
their target junctors. The final conclusion was also modeled as one large AND
inference requiring every detailed subclaim, including personal construction,
the exact push, and personal receipt of diverted funds. Defeating any one of
those details therefore defeated the entire accusation.

A cleaner current-understanding graph would:

1. Remove or relocate caveats that do not invalidate their target inference.
2. Counter genuinely superseded objections with later evidence.
3. Separate `Gideon is responsible for Aldous's murder` from the stronger and
   unnecessary claim that he personally performed every binding operation.
4. Separate `Escrow 17 was fraudulent under Gideon's exclusive administration`
   from the unproved detail that a particular bearer draft was personally
   cashed by Gideon.
5. Support the final accusation through a small number of intermediate
   responsibility, knowledge, motive, and method conclusions rather than one
   eight-source conjunction of every explanatory detail.

This is evidence that the new evaluator is useful: it exposed a mismatch
between the prose proof and the authored formal proof. The requested product
change is better guidance about defeat semantics, not weaker propagation.

## 7. Surface excessive junctor size earlier

### Recommendation

Forward this suggestion.

Concludia validation correctly warns when a junctor has more than three
sources and recommends introducing intermediate lemmas. In this play test,
those warnings appeared only after the graph already contained many junctors
with four to eight sources.

Cludia need not reject a large junctor, but it should surface the Concludia
warning at authoring time or in the ordinary workspace check:

- after `derive` creates an oversized junctor;
- after `add-source` crosses the preferred threshold; and/or
- through a concise structural-warning mode in `check --profile workspace`.

This would have changed authoring behavior while the graph was still easy to
factor.

## 8. Selected-derivation and effective-support ledger views

### Clarification

The request was not to return an arbitrary shallow or incomplete graph. A
`ledger` correctly returns the upstream support closure of its target.

The issue arises when the target has several independent incoming junctors.
For example, the poison hypothesis had three incoming derivations. Its ledger
returned the union of all three support closures, including derivations whose
effective truth was `U` as well as the derivation whose effective truth was
`T`.

Two potentially useful, semantically explicit queries are:

```text
cludia ledger FILE STATEMENT --inference JUNCTOR
```

Return the complete upstream closure of one selected incoming derivation.

```text
cludia ledger FILE STATEMENT --effective
```

Return support closure restricted to support edges currently participating in
the grounded effective result. The exact semantics need core-model review,
especially when multiple `T`, `U`, or defeated paths coexist.

This is better described as branch selection than as a "shortest proof." The
tool should not rank arguments by quality or infer probability.

An additional open question is whether `ledger` is intentionally support-only.
It currently annotates rows with `challenged` and `effective_truth` but does not
include the counterpoint subgraph that produced those values. If a full
argument ledger is desired, an explicit `--with-defeats` option may be more
appropriate than changing the default support ledger.

Forward this for coder analysis; it is useful but lower priority than batch
relations, shared counterpoints, and defeat-authoring guidance.

## Priority order

1. Add defeat-authoring guidance and actionable nonleaf-truth diagnostics.
2. Surface oversized-junctor warnings during ordinary authoring.
3. Investigate selected-derivation, effective-support, and optional
   defeat-inclusive ledger views.
4. Revisit shared counterpoints only after the portable `.arg` format and
   Concludia interoperability contract can represent multiple explicit targets.
