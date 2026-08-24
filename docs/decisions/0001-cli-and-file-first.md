# ADR 0001: Build the CLI and data file before a visual interface

- Status: Accepted
- Date: 2026-08-23

## Context

The central use case is conversational. The user wants to discuss a corpus with
an LLM while the agent reads and mutates durable state, much as Dagim can be
used without continuously watching its TUI.

A TUI, web application, or native Mac prototype would require interaction
decisions before the essential capture/combine/challenge loop has been tested.

## Decision

The first implementation will be a Go CLI over one local readable data file.
Human-readable output is the default and every scriptable operation has
versioned JSON.

A local web UI is planned after conversational dogfooding and must use the same
domain operations and persistence layer.

## Consequences

- The first usable prototype is also the agent integration surface.
- File and JSON contracts receive early design attention.
- Visual graph interaction is informed by real repeated workflows.
- Some users will find the first version less approachable without an agent.
- UI-only semantic operations are prohibited.

## Alternatives considered

- TUI first: attractive given Dagim, but premature for multi-selection and
  generative review.
- Hosted web first: closest to Concludia, but adds accounts, deployment, and
  privacy before the workflow is proven.
- Native Mac first: adds packaging and platform constraints without testing a
  distinct product hypothesis.

