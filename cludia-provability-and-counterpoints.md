# Cludia: Provability Semantics and Counterpoint Guidance

## Summary

Cludia is a premise-up proof record, not a classical truth engine or a
hypothesis workspace. It may contain many disconnected accepted facts and
values whose relevance is not yet known; those islands are not speculative
hypotheses. Premises may carry literal truth values, while lemmas and
conclusions are presented by provability. A proven lemma may be treated as true
downstream; a lemma whose proof is defeated is merely **not currently proven**,
not proven false.

This distinction should be visible in the interface and explicit in authoring guidance. It also suggests that speculative hypotheses and unsupported rival explanations should remain in external notes until the user believes the recorded evidence proves them.

## Semantic model

For premises, the existing truth values remain appropriate:

- `T`: asserted true.
- `F`: asserted false.
- `U`: truth not established.

For lemmas and conclusions, use proof status instead:

- `⊢ P`: `P` is currently proven by at least one undefeated inference whose required sources are available.
- `◇ P`: `P` is possibly proven because its current derivation depends on
  unknown status.
- `⊬ P`: `P` is not currently proven.

Critically:

> `⊬ P` does not mean `⊢ ¬P`.

An accepted undercut disables an inference route. If no other undefeated route proves the target, the target changes from `⊢` to `⊬`; it does not become semantically false. If proving a negation is ever needed, `¬P` should be represented and proven as its own proposition rather than inferred from failure to prove `P`.

This can be implemented using the roles Cludia already has; it does not require new statement types or arbitrary tags.

| Role | Appropriate status |
|---|---|
| Premise | `T`, `F`, or `U` |
| Lemma or conclusion | `⊢`, `◇`, or `⊬` |
| Counterpoint | Its effective truth and separate grounded acceptance status |

## Evaluation behavior

For a lemma or conclusion:

1. If at least one incoming inference is undefeated and its required sources are available, display `⊢`.
2. If the best available derivation depends on unknown sources, display `◇`.
3. If every incoming inference is defeated or false, display `⊬`.
4. Do not display literal `F` merely because a lemma has lost all current proof routes.
5. Defeating one inference must not affect independent incoming proofs.

Conceptually:

```text
accepted sources + undefeated inference  =>  ⊢ target
unknown-dependent best inference         =>  ◇ target
all proof routes unavailable/defeated    =>  ⊬ target
proof of an explicit contrary            =>  ⊢ ¬target
```

This preserves the important distinction between an argument being defeated and its conclusion being false. A conclusion may be true even when the graph no longer proves it.

## UI and API presentation

In the TUI, display `⊢`, `◇`, and `⊬` in the current value position for lemmas
and conclusions. Center every truth/proof value in a fixed-width column so
later columns align, and use a centered `∴` header: it suggests logical
consequence without calling every mixed value **Truth** or **Status**.

Recommended tooltip or help text:

- `⊢`: currently proven; may be used as an accepted source downstream.
- `◇`: possibly proven; the current derivation depends on unknown status.
- `⊬`: not currently proven; this does not mean proven false.

The TUI applies warning color to the value under exactly the same condition as
the `!` after the label: grounded defeat materially changed effective status
somewhere upstream. It does not color values merely because they are `◇` or
`⊬`; doing so would conflate uncertainty or lack of proof with contestation.
Selection styling takes precedence, while `!` preserves the signal without
color.

The current JSON API retains `effective_truth: "T"|"U"|"F"`. Statement role
supplies its interpretation: literal truth for premises/counterpoints and
provability for lemmas/conclusions. This preserves the existing schema and full
three-valued evaluation while human presentation follows Concludia's glyphs.

## Cludia is not a hypothesis workspace

Earlier guidance encouraged hypotheses as unknown propositions. That invited
users and agents to add every plausible explanation, then add speculative
counterpoints to prevent premature acceptance. The result was a large theory
graph rather than a concise proof record.

Recommended replacement guidance:

> Cludia is a proof record, not a workspace for speculative hypotheses. Keep conjectures, rival explanations, open questions, and brainstorming in external notes. Add a lemma when its cited premises are intended to establish it under the working proof standard. A lemma may later become unproven if a specific defect is found in its proof.

The `--truth U` facility can remain useful for genuinely unknown premises or imported claims, but documentation should not recommend it as the normal way to maintain a list of hypotheses.

## Intended role of counterpoints

A counterpoint is appropriate when a recorded premise or proof has a specific,
grounded defect. Examples include:

- A required premise is false or materially unreliable.
- A document or witness used by the proof has a demonstrated custody, identity, or reliability problem.
- A rule was applied outside its stated conditions.
- An actual exception applies in this case.
- A required knowledge transfer, access path, or chronological condition did not occur.
- The sources establish a weaker proposition than the target actually states.

A counterpoint is not appropriate merely because the conclusion might conceivably be false.

Recommended guidance:

> Add a counterpoint only when accepting it would give a reasonable evaluator cause to withdraw this particular proof under the working proof standard. The counterpoint should identify a case-specific defect in the cited premises or inference. Do not use bare possibilities, generic fallibility, absence of proof, residual uncertainty, or unsupported rival explanations as counterpoints.

## The bare-possibility problem

Suppose Cludia records:

```text
All humans are mortal.
Socrates is human.
Therefore, Socrates is mortal.  ⊢
```

“Socrates could be a vampire” is not a useful counterpoint. It supplies no grounded reason to reject either premise or the inference. It merely imagines an exceptional world. Ordinary proof standards do not require eliminating every logically possible exception.

If case-specific evidence later shows that Socrates has no pulse, has not aged in centuries, or was misclassified as human, that evidence may justify a counterpoint. The result would be:

```text
Socrates is mortal.  ⊬
```

It would not establish:

```text
Socrates is not mortal.  ⊢
```

## Counterpoint authoring checks

Before adding a counterpoint, ask:

1. Is this objection grounded in information currently present in the record?
2. Does it identify an actual defect, or only something that could hypothetically be different?
3. Does it attack the exact inference, rather than a stronger claim the inference never made?
4. Would a reasonable evaluator withdraw the proof if the counterpoint were accepted?
5. Is it more properly an external rival hypothesis that should remain outside Cludia for now?

Avoid counterpoints whose only substance is:

- “could,” “might,” “may,” or “possibly”;
- “another explanation exists”;
- “this does not prove the conclusion with certainty”;
- “there is no direct evidence”;
- generic warnings that witnesses, clocks, documents, or measurements can sometimes be wrong.

Those phrases need not be forbidden, and Cludia should not issue lexical
diagnostics merely because they occur. They are prompts for semantic review:

> Is this a live, evidence-backed defect in the proof, or merely residual
> possibility?

## Scope discipline

A counterpoint must target the proposition actually inferred.

For example:

```text
Both documents have the same distinctive typewriter defect.
Therefore, both documents came from the same machine.
```

“The same machine does not prove the same typist” does not undercut that conclusion. It attacks a stronger, separate inference about authorship. It should be attached only if the target is “the same person typed both documents.”

This scope check can prevent valid cautions from disabling proofs they do not actually address.

## Suggested guidance changes

The generated `cludia guidance` output should state explicitly:

- Premise truth and lemma provability use different semantics.
- `◇` means the best current proof depends on unknown status.
- `⊬` means not proven, not proven false.
- Hypotheses and competing theories belong in external notes until supported as intended proofs.
- Counterpoints audit accepted proofs; they are not a place to enumerate possibilities.
- Bare possibility, absence of direct proof, and residual uncertainty are not defeats.
- An undercut disables one inference route and leaves independent routes intact.
- Counterpoints must attack the exact authored inference.

The existing JSON field is retained for schema compatibility but now reports
`hypotheses_as_unknown_propositions: false`.

## Expected result

These changes should:

- Prevent users from reading defeated lemmas as semantically false.
- Reduce speculative and adversarial “anything is possible” counterpoints.
- Keep Cludia workspaces smaller and more proof-oriented.
- Preserve counterpoints for their intended purpose: identifying real defects in proofs previously regarded as adequate.
- Make the relationship between Cludia and Concludia conceptually consistent.
