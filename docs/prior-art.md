# Prior Art and Lessons

This is not an exhaustive literature review. It records systems examined during
the initial product discussion and the specific lessons relevant to Cludia.

## Dagim

[Dagim](https://github.com/tunesmith/dagim) is a local, single-file DAG editor with stable IDs,
atomic mutation, a TUI, and a public JSON CLI.

Borrow:

- conversation-friendly CLI operations;
- human-readable, versioned local files;
- disconnected nodes and components;
- stable identity despite text edits;
- structural validation, dry runs, and atomic saves;
- JSON as a public agent interface.

Do not borrow:

- completion state;
- blocking-edge semantics;
- the `# dagim v1` format.

## Concludia

[Concludia](https://concludia.org/) is an asynchronous, evolvable modality for
communicating arguments through statements, junctors, and counterpoints.

Borrow:

- entailment-style sufficiency rather than relevance-only support;
- linked multi-premise junctors;
- fact/value distinctions and truth tokens;
- premise undermines and inference undercuts;
- counterpoints that can themselves be counterpointed;
- the `.arg` syntax and rooted publication target.

Relax:

- one connected argument;
- a root or conclusion during early authoring;
- rejection of isolated statements and unfinished components.

## ARG-tech and OVA

[ARG-tech](https://www.arg.tech/) and its
[Online Visualisation of Argument](https://www.arg.tech/index.php/ova/) tools
provide academic argument analysis, scheme nodes, linked premises, attacks, and
annotation of who said what.

Useful lesson: a first-class inference/scheme node is the correct representation
for premises that work jointly.

Mismatch: the experience emphasizes reconstructing and depicting existing
discourse more than maintaining an open corpus to discover new conclusions.

## Carneades

The [Carneades Argumentation System](https://carneades.github.io/carneades/) is
an open-source academic argumentation research project named for the Greek
Academic Skeptic. It explored structured arguments, argument schemes,
counterarguments, proof standards, and automatic argument construction.

Useful lessons:

- argument invention can be treated separately from argument evaluation;
- schemes and critical questions may inform future LLM audits;
- richer argument semantics require more than an ordinary graph edge.

Mismatch:

- research-prototype UX and maintenance status;
- substantial machinery around schemes, weights, proof standards, and
  cumulative arguments that v1 does not want;
- no reason to replace Concludia's more capable `.arg` representation.

## Analysis of Competing Hypotheses

Richards Heuer's Analysis of Competing Hypotheses compares alternative
explanations against evidence and emphasizes disconfirmation. See the CIA's
[Psychology of Intelligence Analysis](https://www.cia.gov/resources/csi/books-monographs/psychology-of-intelligence-analysis-2/).

Useful lesson: ask which alternatives and disconfirming observations have been
missed.

Mismatch: the matrix and evidential-consistency workflow is not the persistent
data model. Cludia remains focused on explicit statements and sufficiency
claims, without weights.

## i2 Analyst's Notebook and FBI Sentinel

[i2 Analyst's Notebook](https://i2group.com/solutions/i2-analysts-notebook/plans)
is a large commercial Windows product for entity, link, temporal, geospatial,
and social-network analysis. It is difficult for an ordinary individual Mac
user to evaluate or purchase casually.

[FBI Sentinel](https://www.fbi.gov/how-we-can-help-you/more-fbi-services-and-information/freedom-of-information-privacy-act/department-of-justice-fbi-privacy-impact-assessments/sentinel)
is an internal case-management and analysis environment rather than publicly
downloadable software.

Useful lessons:

- investigation tools need navigation, filtering, search, and timelines at
  scale;
- link analysis and case management are established categories.

Mismatch: both are much larger in scope and center entities, records, and cases
rather than natural-language entailment.

General-purpose civilian link analysis could be valuable in its own right, but
vibe-coding an accessible i2-like application is explicitly outside this
project's initial scope.

## Casefleet and CaseMap

[Casefleet](https://www.casefleet.com/use-cases/investigations-software) and
[CaseMap](https://www.lexisnexis.com/en-us/products/casemap.page) organize legal
facts, evidence, people, issues, and chronologies. Casefleet notably connects a
fact to exact supporting source passages and lets AI propose facts for human
approval.

Useful lesson: if structured provenance is added later, distinguish the source
artifact, cited passage, and asserted statement.

Mismatch: v1 is not evidence ingestion, chronology management, or litigation
software. Adding provenance now would obscure the inference experiment.

## Maltego

[Maltego Community Edition](https://www.maltego.com/pricing/) provides publicly
accessible entity and link analysis for OSINT.

Useful lesson: graph expansion, grouping, selection, filtering, and navigation
may inform a later visual interface.

Mismatch: the graph represents entities and discovered relationships, not
premise conjunctions and conclusions.

## CaseBoard and true-crime workflows

[CaseBoard](https://caseboardapp.org/) targets amateur investigators,
researchers, and true-crime podcasters with private case files, clues, evidence,
people, and theories.

Public true-crime research guidance commonly recommends a document vault,
master timeline, entity list, source citations, and claim tracker. These are
adjacent needs, but the initial Cludia workflow is narrower: turn captured
statements into explicit reasoning.

## What appears distinctive

None of the ingredients is individually new. The promising product combination
is:

1. frictionless capture of atomic statements;
2. a disconnected local workspace;
3. conversational discovery of candidate statement combinations;
4. explicit missing-premise reconstruction;
5. strict multi-premise sufficiency for persisted authoring;
6. first-class challenges without scoring;
7. promotion of a complete rooted structure into Concludia.

The innovation claim should be about this workflow and modality, not about
inventing a new logic or graph format.
