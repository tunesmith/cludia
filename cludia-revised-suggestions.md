# Remaining Cludia Suggestions After Mystery Play Test

## Context

These remaining suggestions come from using Cludia to track a six-installment
fair-play mystery containing hundreds of statements, dozens of junctors, and
many defeats. Completed, withdrawn, or resolved suggestions are removed rather
than retained as project history; this file tracks only possible future work.

## Remaining work

| # | Suggestion | Current disposition |
|---|---|---|
| 2 | Reuse one counterpoint at multiple explicit defeat locations | Requires a coordinated Concludia and `.arg` format decision |
| 8 | Defeat-inclusive explanation views | Effective-support semantics remain open |

## 2. Reusing one counterpoint at multiple explicit defeat locations

### User need

One true counterpoint may be relevant to more than one explicit defeat
location. The current workflow must create nearly identical counterpoint
statements for each target.

The desired operation would remain explicit. It must not automatically attack
every inference sharing a target merely because one counterpoint is relevant to
several of them.

### Format blocker

Concludia's internal defeat registry can hold multiple defeat edges originating
from one counterpoint. The portable `.arg` syntax cannot currently express that
state: a counterpoint declaration has one trailing target, and Cludia therefore
rejects a second target with `defeat_source_multiple`.

Do not add an `attach-counterpoint` operation in Cludia alone. First decide how
multiple explicit targets are represented and round-tripped in `.arg`, update
Concludia's parser/serializer and interoperability contract, and then expose the
shared capability in Cludia.

### Proposition-level rebuttal remains separate

Concludia currently models premise defeats, inference defeats, and
counterpoints of counterpoints. It does not define a proposition-level rebuttal
of a derived lemma or conclusion independently of its incoming inference
paths. Cludia should not invent that semantic primitive independently.

## 8. Defeat-inclusive explanation views

### Defeat-inclusive explanation

The ledger is intentionally support-only. It annotates statements with
effective truth and propagated contestation but does not show the counterpoint
subgraph that produced those values.

A separate explanation view, or an explicit defeat-inclusive ledger mode,
could show the accepted defeat chain responsible for changed truth while
leaving the default ledger simple. Statement Detail already supplies local
defeat inspection; the open need is a coherent rooted explanation.

### Effective-support view remains undefined

Do not implement `ledger --effective` until its policy is explicit. There is no
unique structural answer today:

- for a true OR, it could include one true justification or every true one;
- for a false AND, it could include only decisive false sources or every source;
- for unknown truth, several different unknown paths may explain the result;
- active but outcome-neutral justifications may or may not belong.

Any future design must declare those choices rather than silently presenting
one path as the argument.

## Priority order

1. Design a defeat-inclusive rooted explanation view.
2. Revisit shared counterpoints only after the `.arg` format decision.
