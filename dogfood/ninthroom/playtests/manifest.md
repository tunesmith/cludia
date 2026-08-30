# Ninth Room playtests

These files preserve Cludia workspaces produced while solving different
revisions of *The Ninth Room*. They are dogfood evidence, not canonical mystery
sources or golden-output fixtures.

Keep blind/raw captures distinct from graphs extended after an accusation or
solution. Do not continue editing a raw capture; copy it to a clearly named
`post-solution` or `curated` workspace first.

| ID | File | Solver | Story revision | Delivery | Capture state | First accusation | Notes |
|---|---|---|---|---|---|---|---|
| 01 | `01-human-original-post-solution.cludia` | Human, with agent-mediated CLI mutations | Initial conversational version; first committed in `072b488` | Six incremental sections | Began blind, then extended substantially after accusation and epilogue | Gideon, after Part VI | Hybrid reasoning record rather than a pristine accusation-time snapshot |
| 02 | `02-agent-first-revision-blind.cludia` | Agent; model not recorded | First six-part revision, preserved by the split at `2ff1d06` | Six incremental sections | Reported blind solve through its completed accusation; no later curation reported | Gideon, after Part VI | Playtest was broadly successful but exposed procedural and evidentiary gaps addressed by the following story revision |
| 03 | `03-agent-procedural-revision-premature-convergence.cludia` | Agent; model not recorded | Procedural-detail revision committed in `219a5ca` | Six incremental sections | Reported blind solve through its completed accusation; no later curation reported | Strong Gideon hypothesis after Part IV; final accusation after Part VI | Detailed non-standard Cludia usage; demonstrates premature culprit convergence caused by accumulated procedural records |

## Playtest 01 details

- Format metadata: workspace profile, version `0.1.0`.
- Exact Cludia CLI build was not recorded (`cludia version` currently reports
  `dev`).
- The committed file contains post-solution bridging premises, forensic facts,
  intermediate lemmas, and a completed proof rooted at `L36`.
- Because the file was first committed after that curation, Git does not contain
  its earlier state at the moment of accusation.
- SHA-256 at relocation:
  `a9f1cba44e5b8e7b63381e1c65ee0b4b81afeaaf563395ab8751984376cb9d60`.

## Playtest 02 details

- Format metadata: workspace profile, version `0.1.0`.
- Workspace size: 296 statements, 75 junctors, and eight defeat relations.
- The completed solution is rooted at `L74`; its upstream closure contains 96
  statements and 23 junctors.
- The rooted closure is exportable but reports proof-local warnings for several
  junctors with more than three sources. Preserve those as observed agent
  behavior rather than retroactively restructuring this raw playtest.
- The exact point of unique culprit convergence was not recorded. The playtest
  was reported as reasonably successful, with holes that motivated the next
  revision's stronger procedural evidence.
- SHA-256 at relocation:
  `2d20fff7c338cb8cf21c0f2365553f7bf375b49d129ba3ee8a70d330d8897a20`.

## Playtest 03 details

- Format metadata: workspace profile, version `0.1.0`.
- Workspace size: 215 statements, 72 junctors, and six defeat relations.
- The solver used bracketed epistemic categories inside statement text, including
  `OBSERVED`, `TESTIMONY`, provisional and strong inferences, suspect rankings,
  assessments, open questions, and final conclusions.
- Inference and conclusion statements retained `P` identities while their roles
  changed; the next-ID metadata still reports `L=1`. This is valid but unusual
  usage and should be preserved rather than normalized.
- Gideon was the provisional leading suspect after Part II (`P74`), remained
  the leading single-person fit after Part III (`P104`), and became a strong
  culprit and overall hypothesis after Part IV (`P132` and `P142`). This makes
  the graph a useful record of premature convergence.
- The final solution is rooted at `P210`; its upstream closure contains 49
  statements and 12 junctors. It is exportable but reports seven proof-local
  warnings for junctors with more than three sources, including a ten-source
  final junctor.
- SHA-256 at relocation:
  `868edf40637f860dcb540c96288aff80b9f9580f97064586aad9baeae0d6307c`.
