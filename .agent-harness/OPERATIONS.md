---
name: OPERATIONS.md
description: Operations quick-start, reference map, and runtime procedures.
---

# Operations

## Local Development

```bash
go test ./...
go run ./cmd/llm-wiki --version
go run ./cmd/llm-wiki validate fixtures/okf-minimal --json
```

Use `go run ./cmd/llm-wiki ...` during development and the installed `llm-wiki` binary in host integrations.

## OKF Bundle Workflow

- Initialize: `llm-wiki init <path> --profile obsidian --okf-version 0.1`
- Validate hard conformance: `llm-wiki validate <path> --json`
- Lint soft quality issues: `llm-wiki lint <path> --json`
- Apply safe index fixes: `llm-wiki lint <path> --fix` or `llm-wiki index <path> --write`
- Append history: `llm-wiki log <path> append --op <op> --message <text>`
- Build context: `llm-wiki graph <path> --json` and `llm-wiki query-pack <path> "<question>" --json`

## MCP

Run `llm-wiki mcp` as a stdio MCP server. Initial tools are `llm_wiki_validate`, `llm_wiki_lint`, `llm_wiki_index`, `llm_wiki_graph`, and `llm_wiki_query_pack`. These tools call the same internal service packages used by the CLI.

Baseline MCP smoke:

```bash
go test ./internal/mcp
go run ./cmd/llm-wiki mcp < /dev/null
go run ./cmd/llm-wiki daemon status --json
```

`llm-wiki mcp --daemon` is reserved for future daemon-backed MCP and currently
returns an unsupported error. Keep host configuration on plain `llm-wiki mcp`.

## Host Integrations

Claude Code, Codex, Reasonix, and portable agents should use the same CLI/MCP surface. See `packages/hosts/*` for example hook/settings and MCP config files, and `docs/host-mcp-smoke.md` for host-specific smoke steps. Host adapters are allowed to format host settings but must not duplicate OKF core logic. Host smoke work should start with the non-mutating probes in `docs/host-mcp-smoke.md` before changing user-level host configuration.

## Runtime State

Hook event logging writes redacted JSONL under `.llm-wiki/hooks.jsonl` inside the current workspace. `.llm-wiki/` is ignored by git. Logs use file locks and payload caps; hooks should stay fast and should not perform model calls.

The daemon skeleton resolves future runtime state in this order:

1. `LLM_WIKI_STATE_DIR`
2. `$XDG_STATE_HOME/llm-wiki`
3. `~/.local/state/llm-wiki`

Reserved daemon files are `daemon.sock`, `daemon.pid`, and `daemon.lock`.
`daemon status` and `daemon doctor` are safe probes. `daemon start` and
`daemon stop` are unsupported and must not create runtime files until the daemon
runtime is intentionally implemented.

## Project Docs

`agent-harness project bootstrap --repo . --json` created this repo's AGENTS and `.agent-harness` docs. Future updates should read current docs first, then update through `project_docs_update` or append decisions through `project_docs_record`.
