# Vision

## The experience

People regularly encounter isolated facts that feel suggestive before they
know what those facts establish. A datum, quotation, observation, or remembered
detail may matter, but it may be weeks before another statement gives it shape.
Ordinary notes preserve the fragments but do little to help them become
reasoning.

Cludia is a notebook for that interval between noticing and concluding.

It begins as a corpus of short, accepted factual and value statements. Most are
disconnected. Over time, some can be assembled into partial findings, lemmas,
and eventually conclusions. Other true statements remain unused or turn out to
be red herrings; disconnectedness means only that their relevance is not yet
known, not that they are speculative hypotheses.

The desired experience is not primarily drawing a graph. It is being able to
talk through the corpus with an LLM while the tool quietly maintains a durable,
inspectable structure:

> Show me the unexplained observations. Which of these statements can work
> together? Does that conclusion actually follow? What premise did we smuggle
> in? What challenges this clue or this inference? Preserve the repaired
> argument.

## Premise-up discovery

Concludia is naturally conclusion-oriented. A participant encounters a claim
and asks, "Why?" The graph expands downward toward its premises.

Cludia reverses the initial authoring pressure. It starts with premises and asks
what, if anything, follows. No global conclusion is required, and the corpus
may contain many disconnected reasoning islands.

The resulting mature structure should nevertheless be representable as a
Concludia graph. Cludia is therefore not a competing logic. It is a generative
workspace that can produce Concludia arguments.

## Logical force, not additive force

Cludia does not treat statements as weighted points in favor of a conclusion.
It does not sum evidence, calculate confidence, or turn repeated suggestions
into a percentage.

An authored inference is a linked unit:

```text
A AND B -> C
```

The author asserts that if every source is true, the target follows. If it does
not, the structure needs another premise, a more modest target, a repair, or a
recorded challenge.

The system focuses on true statements and explicit logical moves. It does not
claim that unrestricted natural-language entailment can be mechanically
proved; semantic sufficiency remains subject to human and LLM audit.

## Abduction as discovery, deduction as artifact

The search for a promising conclusion is often abductive: a person or model
invents an explanation that might organize the observations. That is valuable,
but the invented hypothesis is not made true merely by explaining the clues.

Cludia therefore keeps abduction outside the persisted edge semantics. A human
or LLM may use any reasoning process to invent a candidate. Before persistence,
the candidate is reconstructed as an explicit sufficiency claim. Hidden bridge
premises must become statements. A speculative theory is not captured as an
unknown premise merely because it may or may not be true.

This distinction lets the experience feel Sherlockian without pretending that
suggestiveness is proof.

## Challenges are part of inquiry

An investigation needs to remember not only successful arguments but material
problems:

- An **undermine** challenges the truth or scope of a statement.
- An **undercut** grants the stated sources but challenges whether the target
  follows from that particular inference.
- A **counterpoint of a counterpoint** challenges the prior counterpoint. The
  same relationship may continue recursively; no special "rejoinder" concept
  is needed.

Challenges are not scores. They are statements with explicit targets. They
should encourage repair, qualification, or further investigation.

The workspace is a living record of inquiry. A challenged or obsolete
inference may remain alongside its undercut while the author evaluates or
repairs it. That coexistence records disagreement with an authored sufficiency
claim; it does not erase the claim or make it structurally invalid. Known-bad
inferences should eventually be repaired or removed as reasoning hygiene, but
the tool does not require the workspace to be semantically settled before it
can be saved or exported with its contestation intact.

## Conversation is the first interface

Dagim demonstrated that a useful graphical domain can be manipulated
productively through CLI calls embedded in a normal conversation. The user does
not need to stare at the TUI while an agent lists nodes, discovers structure,
proposes mutations, and applies approved changes.

Cludia treats that interaction as the primary prototype. The initial interface
is a readable file plus a complete JSON CLI. A visual interface comes later and
must not create semantic capabilities unavailable to the CLI.

## Product hypothesis

The project succeeds if a person and an LLM can maintain a workspace of dozens
or hundreds of observations, discover non-obvious linked inferences, expose
missing premises, preserve meaningful challenges, and promote mature rooted
structures into Concludia without manually managing the file.

The hypothesis is about a modality of reasoning, not a graph renderer:

> A durable corpus plus a conversational, premise-up construction loop can
> help people reach wiser conclusions than either unstructured notes or a
> conclusion-first argument editor alone.
