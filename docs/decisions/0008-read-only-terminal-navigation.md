# ADR 0008: Add read-only terminal navigation before graphical UI work

- Status: Accepted
- Date: 2026-08-26
- Partial supersession: ADR 0010 later adds focused durable Top reordering while
  retaining this ADR's navigation model and deferral of general TUI authoring.

## Context

ADR 0001 deliberately placed the CLI and data file before any visual interface
because the core capture, inference, challenge, and repair loop had not yet been
dogfooded. Subsequent Codex and Claude investigations validated that loop and
repeatedly reconstructed two exact textual views: statements with no outgoing
support target, and a topologically ordered derivation to a selected statement.

Generated graphical views were useful for visual-language exploration but
either invented edge topology or required fragile manual layout. Full-text
terminal lists and derivation ledgers communicated the same graph structure
reliably. Dagim also demonstrates that a terminal list/detail navigator can
remain useful while an external agent edits the underlying file.

The connected argument supporting this decision is
[0008-read-only-terminal-navigation.arg](0008-read-only-terminal-navigation.arg).

## Decision

- `cludia FILE` opens a read-only terminal navigator; named subcommands retain
  precedence.
- The initial TUI contains only Top, Statement Detail, and Derivation Ledger
  views.
- Top lists non-counterpoint statements with no outgoing support in document
  order, with longest support depth and a compact challenge marker.
- Statement text is always complete and wraps; list and ledger views do not
  summarize or truncate it.
- Statement Detail exposes distinct justifications, their undercuts, direct
  statement challenges, recursive counterpoints, and downstream uses.
- The ledger is rooted at one statement and displays stable topological order
  with label, full statement text, and compact `AND(...)`, `OR(...)`, or direct
  derivation notation. Human output omits junctor IDs.
- `cludia top` and `cludia ledger` expose the same deterministic read models in
  human and versioned JSON form for agent and interface parity.
- The TUI automatically follows valid external file changes, preserves
  selection by durable statement ID when possible, and retains the last valid
  in-memory document when an external change is invalid.
- The first TUI is strictly read-only and introduces no mutation semantics,
  autosave, undo history, graphical map, or embedded LLM.

## Consequences

- ADR 0001's CLI-first sequencing remains satisfied; dogfooding now supplies
  the interaction evidence that was previously missing.
- Exact textual navigation precedes graphical web work without replacing the
  CLI or JSON integration surface.
- Top and ledger semantics live in the domain query layer rather than Bubble
  Tea presentation code.
- Full challenge adjudication remains out of scope: any attached undermine or
  incoming-inference undercut marks a statement as challenged even when a
  counterpoint itself has been countered.
- Search, isolated-only browsing, challenge-only browsing, copy/export, and all
  TUI mutations remain deferred until the three-view navigator is dogfooded.

## Alternatives considered

- Continue with guidance-only agent summaries: workable, but repeated manual
  reconstruction lacks a stable human navigation surface and exact query
  contract.
- Build a graphical renderer first: offers spatial overview but introduces
  layout and topology-presentation risk before simpler textual navigation is
  tested.
- Add TUI mutation authoring immediately: rejected because read-only navigation
  can be evaluated independently and all durable mutations already exist in the
  CLI.
