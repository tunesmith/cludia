# CLI process and JSON contract

## Scope

This document defines the process-level contract for scripts and external
agents invoking `cludia`. It complements the response-family field contracts
tested with each command.

## Exit status and streams

Successful commands exit with status `0`. Failed commands exit nonzero.

For a recognized command with valid command-line syntax and `--json`:

- success writes exactly one indented JSON object followed by a newline to
  standard output;
- structural, validation, domain, stale-plan, and other expected operation
  failures write exactly one structured JSON failure object to standard output
  and exit nonzero;
- warnings on successful commands appear only in the JSON `diagnostics` array;
- standard error is empty for those structured successes and expected
  structured failures.

Command-line usage failures are different because the process may not have a
valid response request to execute. Missing arguments, unknown flags, invalid
flag combinations, and unknown commands write human-readable usage or error
text to standard error, write nothing to standard output, and exit nonzero even
when `--json` appeared among the incomplete arguments.

Without `--json`, normal results and validation diagnostics are written to
standard output. Usage and dispatch errors are written to standard error.

Unexpected process failures that cannot be represented as an operation result
may write a human-readable error to standard error and exit nonzero. Callers
must always inspect the exit status before interpreting either stream.

## JSON versions and response families

Every structured top-level response contains integer `schema_version`. Version
`2` is the current contract. Scripts must read this field and reject versions
they do not support.

Batch input carries its own independent schema version. The current and only
supported version is 2, the atomic statement, derivation, and defeat
transaction. Grounded evaluation objects also carry an independent evaluation
schema version, which is currently 1.

`effective_truth` remains `T`, `U`, or `F` in JSON. Human presentation is
role-aware: premises and counterpoints show those literal values, while lemmas
and conclusions render them as `⊢`, `◇`, and `⊬`. No `proof_status` field is
added in schema 2; callers can combine statement role with effective truth when
they need the human interpretation.

Within one schema version, field names, JSON types, meanings, and required
collection behavior are stable for each response family. An incompatible
change to those shapes or meanings requires a schema-version change. Stable
diagnostic codes may be added as new failure cases are recognized without
changing the response schema.

Cludia 1.x does not retain old JSON schema implementations indefinitely.
Incompatible changes increment the applicable schema version and are disclosed
in release notes; callers upgrade with the installed client. The durable 1.x
compatibility promise applies to `.arg` files rather than old CLI requests.

Each command's tests assert its exact top-level key set. Nested public mappings
whose fields are part of a mutation protocol, such as derive role changes and
renumber mappings, are likewise contract-tested.

Selected ledger reads add `selected_inference`, containing the selected
junctor, its `effective_truth`, `disabled_by_undercut`, whether other root
justifications were omitted, and whether those omitted routes change the
root's overall truth. Junctor effective truth remains the calculated result of
its sources; disabling its target edge does not rewrite that truth. Normal
complete ledger reads omit this object.

## Collections and diagnostics

Public collection fields are encoded as arrays. An empty collection is `[]`,
not `null` and not an omitted field, unless that field is explicitly documented
as optional for its response family.

Diagnostics contain stable `code`, `message`, `severity`, and element context;
line is included when known. Human-readable messages may improve without a
schema change. Scripts should branch on `code`, not parse `message` text.

Warnings do not change a successful exit status. Any error-severity diagnostic
makes the operation unsuccessful.

## Mutation timing and atomicity

An applied mutation validates the complete resulting document and saves it
atomically before writing its success response. A dry run validates and reports
the proposed result without writing or consuming identifiers.

Expected mutation failure, including a stale two-phase token, writes no
workspace change. File-creation and export commands also refuse partial output.

The response is emitted after persistence. Therefore, an output transport
failure such as a closed stdout pipe can occur after a mutation has committed.
When delivery is uncertain, the workspace file is authoritative and the caller
must re-read it before retrying. State-bound two-phase operations remain safe to
retry only by obtaining a fresh plan when the prior outcome is unknown.

## Representative cases

| Case | Exit | stdout | stderr | Write |
|---|---:|---|---|---|
| Successful `--json` read | 0 | One success object | Empty | None |
| Successful `--json` mutation | 0 | One success object | Empty | Atomic |
| Successful `--json --dry-run` | 0 | One dry-run object | Empty | None |
| JSON validation/domain failure | Nonzero | One failure object | Empty | None |
| Stale JSON apply token | Nonzero | One failure object | Empty | None |
| Usage or flag failure | Nonzero | Empty | Human usage/error | None |
