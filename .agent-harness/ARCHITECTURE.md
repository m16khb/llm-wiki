---
name: ARCHITECTURE.md
description: System structure, component boundaries, and responsibilities.
---

# Architecture

## Purpose

`llm-wiki` is a Go, OKF-native local-first toolkit. It owns bundle mechanics such as validation, linting, indexing, logging, graph extraction, query-pack construction, hook event logging, and import/export planning. It does not synthesize LLM answers or call model APIs; host agents consume bounded context and decide what to write.

## Core Shape

- `cmd/llm-wiki`: Cobra CLI entrypoint, flag parsing, JSON output, and exit-code mapping.
- `internal/okf`: bundle scan model, reserved files, concept discovery, and root-safe write paths.
- `internal/frontmatter`: YAML frontmatter parse/write with unknown-field preservation.
- `internal/validate`: strict OKF conformance DTO for required fields and concept counts.
- `internal/lint`: soft quality warnings such as broken wiki links; validation stays OKF-only.
- `internal/index`, `internal/logstore`: deterministic index writing and locked append-only log writes.
- `internal/graph`, `internal/querypack`: deterministic graph and bounded context output; query-pack never answers.
- `internal/hooks`: host-shaped hook output plus redacted JSONL event logging with file locks.
- `internal/mcp`: stable MCP tool catalog surface for the same service semantics as CLI.
- `internal/importexport`: fixture-level NVK import/export planning and dry-run behavior.
- `packages/hosts/{claude,codex,reasonix,portable}`: host integration notes/templates only; no duplicated OKF core logic.

## OKF Boundary

OKF v0.1 compatibility is documented in `docs/okf-v0.1-compat.md`. The upstream Google OKF v0.1 spec is vendored for local reference under `third_party/google-okf/` with its Apache-2.0 license and source metadata. The executable hard contract remains the code and tests: non-reserved Markdown concepts require YAML frontmatter with `type`; `index.md` and `log.md` are reserved; unknown fields are tolerated and preserved; broken links are lint warnings.

## Host-Neutral Rule

Claude Code, Codex, Reasonix, and portable agents should invoke the same CLI/MCP behavior. Host-specific packages may provide settings, hooks, or skill wrapper text, but must not implement separate validation, linting, graph, index, log, or query-pack logic.
