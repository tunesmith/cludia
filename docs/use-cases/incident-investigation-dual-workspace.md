# Dual-workspace incident investigation

This is one optional way to use Cludia during a software incident. It is not a
required workspace mode or `.arg` profile.

## Artifacts

An incident may use:

```text
incident-current.arg
incident-history.arg
incident-response.dagim
```

`incident-current.arg` contains the current best causal and justificatory model.
Materially superseded propositions can be replaced through explicit graph
operations and removed from this file; Git preserves earlier tracked revisions.

`incident-history.arg` contains explicitly time-scoped observations, beliefs,
tests, and changes in investigative direction. A statement such as “At 10:14,
responders believed the database was unavailable” remains true even after that
hypothesis is rejected.

`incident-response.dagim` contains dependency-ordered mitigation, recovery, and
corrective work.

## Suggested conversational transaction

For a material change in the current model, an external agent can:

1. Add the replacement proposition with a new ID.
2. Audit and repair each affected support or defeat relation.
3. Delete the obsolete current-model statement when appropriate.
4. Add a time-scoped account of the transition to the history workspace.
5. Validate both `.arg` files and the Dagim plan.
6. Commit the tracked artifacts together at a meaningful checkpoint.

Cludia does not currently provide a multi-file atomic transaction. If one
mutation fails, the valid but uncommitted files can be repaired or reverted
before the shared Git checkpoint.

## Variations

A purely chronological discovery record may be clearer as Markdown or another
timeline-oriented artifact. A smaller inquiry may need only one `.arg`. The
identity rules in ADR 0007 do not depend on this dual-workspace arrangement.
