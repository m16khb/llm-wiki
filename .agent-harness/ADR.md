---
name: ADR.md
description: Structural decisions, rationale, and rejected alternatives.
---

# Architecture Decision Records

## Purpose

Record structural choices, rejected alternatives, and decisions that affect long-term maintenance. This is not an implementation note; preserve why this structure was chosen and which alternatives should not be retried.

## When to read

- Before architecture changes, large refactors, or dependency/framework replacement
- When changing or bypassing existing structure
- When modifying code whose historical rationale is unclear

## When to append

- A new structure or boundary was chosen.
- Alternatives were considered and rejection reasons will reduce future re-analysis.
- Operations, performance, or security constraints shaped the design.

## Entry template

### YYYY-MM-DD: <decision title>

- Context: <problem and constraints>
- Decision: <chosen structure>
- Alternatives rejected:
  - <alternative>: <why rejected>
- Consequences: <tradeoffs and follow-up>
- Evidence: <files, commands, issues, docs>

## 2026-06-15 — Go host-neutral OKF core

- Kind: `adr`
- Source: codex implementation
- Summary: llm-wiki uses a single Go CLI/MCP core with host integrations as thin wrappers.
- Context: The user requested compatibility across Claude Code, Codex, Reasonix, and other agents while referencing agent-harness patterns.
- Decision: Keep OKF behavior in internal Go service packages and expose it through one Cobra CLI plus MCP catalog; host packages only contain settings/templates/docs.
- Consequences: New host support should add templates/adapters around the same binary rather than reimplementing validation, lint, graph, index, log, or query-pack behavior.
- Evidence:
  - go.mod
  - cmd/llm-wiki/main.go
  - internal/* packages
  - packages/hosts/*/README.md
  - agent-harness architecture evidence: common Go core plus thin host adapters
- Alternatives / rejected options:
  - Host-specific plugins implementing OKF behavior separately
  - Rust implementation from the older plan
  - Embedding LLM answer synthesis in core

## 2026-06-15 — Link upstream OKF spec instead of vendoring

- Kind: `adr`
- Source: codex implementation
- Summary: The repository documents OKF v0.1 compatibility and links to Google upstream sources instead of copying the full spec.
- Context: The user asked whether Google OKF docs should be copied into the repo.
- Decision: Do not vendor the full Google OKF spec initially; keep docs/okf-v0.1-compat.md with official links and implemented compatibility scope.
- Consequences: If upstream text or samples are copied later, add attribution and license files in the same change.
- Evidence:
  - docs/okf-v0.1-compat.md
  - GoogleCloudPlatform/knowledge-catalog repo license page
  - Google OKF SPEC.md upstream
  - Google Cloud OKF announcement
- Alternatives / rejected options:
  - Vendor third_party/google-okf/SPEC.md with attribution
  - Only keep machine-readable JSON schema and no prose compatibility note

## 2026-06-15 — Vendor Google OKF v0.1 spec after adopting Apache-2.0

- Kind: `adr`
- Source: codex implementation
- Summary: The project now vendors the upstream Google OKF v0.1 spec with license and source metadata, superseding the earlier link-only note.
- Context: After choosing Apache-2.0 for this repository, the user pointed out that copying the Apache-licensed OKF document is reasonable and useful for local reference.
- Decision: Vendor `okf/SPEC.md` and upstream `LICENSE.md` under `third_party/google-okf/`, add `SOURCE.md`, and update README/compat/architecture docs.
- Consequences: Future upstream refreshes should update SPEC.md, LICENSE.md if changed, SOURCE.md metadata, and any tests/docs that rely on spec wording.
- Evidence:
  - third_party/google-okf/SPEC.md
  - third_party/google-okf/LICENSE.md
  - third_party/google-okf/SOURCE.md
  - README.md
  - docs/okf-v0.1-compat.md
  - .agent-harness/ARCHITECTURE.md
- Alternatives / rejected options:
  - Keep only upstream links
  - Copy excerpts into compatibility docs
  - Track only JSON schema

## 2026-06-15 — SDK-backed MCP server and stricter OKF conformance

- Kind: `adr`
- Source: codex implementation
- Summary: MCP now uses the Go MCP SDK for real tools/list and tools/call behavior, and validate enforces OKF v0.1 hard conformance beyond missing type.
- Context: The user selected follow-up options 1 and 2: implement the actual MCP stdio JSON-RPC handler and strengthen OKF spec-based validation.
- Decision: Use the Go MCP SDK in `internal/mcp` with tools backed by existing service packages, and keep hard validation limited to OKF conformance while preserving lint for soft quality warnings.
- Consequences: Future CLI behavior changes should be reflected in MCP service tools, and future validation additions must distinguish OKF hard conformance from lint-only quality guidance.
- Evidence:
  - internal/mcp/mcp.go
  - internal/mcp/mcp_test.go
  - internal/validate/validate.go
  - internal/validate/validate_test.go
  - third_party/google-okf/SPEC.md
  - docs/okf-v0.1-compat.md
- Alternatives / rejected options:
  - Keep `llm-wiki mcp` as a ready JSON placeholder
  - Implement ad hoc JSON-RPC manually without the SDK
  - Move broken links into validation errors
