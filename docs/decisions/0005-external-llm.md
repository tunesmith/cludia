# ADR 0005: Keep the LLM outside the tool in v1

- Status: Accepted
- Date: 2026-08-23

## Context

Candidate discovery, missing-premise generation, and semantic auditing are
natural LLM tasks. The intended user already interacts with an LLM capable of
calling local tools.

Embedding those calls in the product would reverse the desired direction of
interaction and introduce provider, credential, cost, privacy, and prompt
contracts before the file/CLI workflow is proven.

## Decision

V1 is deterministic and model-agnostic. It exposes complete human and
versioned JSON operations. An external LLM uses the CLI, and later a thin MCP
adapter, during a conversation.

The tool performs structural validation. It does not claim to prove
natural-language entailment or silently persist model proposals.

## Consequences

- No API key or network access is required.
- Users choose their conversational model and client.
- Sensitive workspaces remain local unless the user shares them with an
  external model.
- Non-agent users do not receive built-in generative discovery in v1.
- Later model-powered features require a new decision and explicit boundary.

## Alternatives considered

- Built-in `discover` and `audit` calls: potentially valuable later, but
  premature.
- Embeddings from the start: could help retrieval at scale, but real corpus
  size and retrieval needs are not yet known.
- No LLM integration at all: unnecessarily restrictive; the CLI is explicitly
  designed for external agents.

