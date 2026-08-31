# Changelog

## 1.0.1 (2026-08-31)

### Changed

- Human CLI and TUI reads now present premises/counterpoints with literal
  `T/U/F` and lemmas/conclusions with Concludia-compatible `⊢/◇/⊬`
  provability. Fixed-width Top, Ledger, and Detail values are centered beneath
  a centered `∴` header rather than calling every value truth or status.
- The TUI applies warning color to the displayed truth/proof value under the
  exact same material-defeat condition as the durable `!` marker; selection
  styling takes precedence and `!` preserves the signal without color.
- Agent guidance now distinguishes accepted disconnected evidence from
  speculative hypotheses, reserves `U` for genuinely uncertain recorded
  propositions, and requires case-specific, exact-scope counterpoints rather
  than bare possibilities or unsupported rival explanations.

### Compatibility

- `.arg`, CLI JSON schema 2, batch schema 2, and evaluation schema 1 are
  unchanged. JSON continues exposing `effective_truth: "T"|"U"|"F"`; the
  role-aware proof glyphs are human presentation only.

## 1.0.0 (2026-08-30)

### Added

- A local, file-first reasoning workflow with scriptable CLI commands and a
  terminal navigator for Top, Statement Detail, and Derivation Ledger views.
- Permissive Cludia `.arg` workspaces with lossless reading of supported
  ordinary Concludia files and complete rooted Concludia export.
- Deterministic three-valued truth propagation with grounded undermines,
  undercuts, and recursive counterpoint acceptance.
- Atomic batch authoring and shared clone-returning mutation operations across
  CLI and TUI persistence paths.
- Monotonic role-consistent IDs, premise-to-lemma promotion, explicit material
  replacement, reviewed whole-document renumbering, durable statement order,
  and selected-inference Ledger reads.

### Compatibility

- `profile="cludia"` is the stable permissive profile. Legacy
  `profile="workspace"` files remain readable and migrate on their next
  successful durable save; reads, dry runs, and failed operations do not
  rewrite them.
- New Cludia files omit graph artifact `meta version`; imported authored values
  are preserved and remain distinct from Cludia's software and `.arg` syntax.
- Cludia 1.x guarantees reading prior Cludia `.arg` files and supported ordinary
  Concludia `.arg` files without silent loss.
- Human CLI and JSON contracts may evolve during 1.x. Incompatible JSON changes
  require a schema-version bump and release-note disclosure; older schemas need
  not remain implemented. Version 1.0.0 uses CLI schema 2, batch schema 2, and
  evaluation schema 1.
