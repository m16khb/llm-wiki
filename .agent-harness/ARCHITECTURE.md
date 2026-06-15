---
name: ARCHITECTURE.md
description: System structure, component boundaries, and responsibilities.
---

# Architecture

## Purpose

`llm-wiki` is a Go, OKF-native local-first toolkit. It owns bundle mechanics such as validation, linting, indexing, logging, graph extraction, query-pack construction, hook event logging, MCP serving, and import/export planning. It does not synthesize LLM answers or call model APIs; host agents consume bounded context and decide what to write.

## Core Shape

- `cmd/llm-wiki`: Cobra CLI entrypoint, flag parsing, JSON output, exit-code mapping, and `mcp` stdio command wiring.
- `internal/okf`: bundle scan model, reserved files at every hierarchy level, concept discovery, and root-safe write paths.
- `internal/frontmatter`: YAML frontmatter parse/write with unknown-field preservation.
- `internal/validate`: strict OKF conformance DTO for UTF-8, required concept frontmatter/type, reserved file structure, and concept counts.
- `internal/lint`: soft quality warnings such as broken wiki links; validation stays OKF-only.
- `internal/index`, `internal/logstore`: deterministic index writing and locked append-only log writes.
- `internal/graph`, `internal/querypack`: deterministic graph and bounded context output; query-pack never answers.
- `internal/hooks`: host-shaped hook output plus redacted JSONL event logging with file locks.
- `internal/mcp`: Go MCP SDK server that exposes the same service semantics as CLI through `llm_wiki_validate`, `llm_wiki_lint`, `llm_wiki_index`, `llm_wiki_graph`, and `llm_wiki_query_pack`.
- `internal/importexport`: fixture-level NVK import/export planning and dry-run behavior.
- `packages/hosts/{claude,codex,reasonix,portable}`: host integration notes/templates only; no duplicated OKF core logic.

## OKF Boundary

OKF v0.1 compatibility is documented in `docs/okf-v0.1-compat.md`. The upstream Google OKF v0.1 spec is vendored for local reference under `third_party/google-okf/` with its Apache-2.0 license and source metadata. The executable hard contract remains the code and tests: non-reserved Markdown concepts require valid UTF-8, parseable YAML frontmatter, and `type`; `index.md` and `log.md` are reserved at every level; root `index.md` may declare only `okf_version`; `log.md` date headings must use `YYYY-MM-DD`; unknown fields are tolerated and preserved; broken links are lint warnings.

## Host-Neutral Rule

Claude Code, Codex, Reasonix, and portable agents should invoke the same CLI/MCP behavior. Host-specific packages may provide settings, hooks, or skill wrapper text, but must not implement separate validation, linting, graph, index, log, or query-pack logic.
