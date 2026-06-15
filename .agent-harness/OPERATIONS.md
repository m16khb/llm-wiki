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

## Host Integrations

Claude Code, Codex, Reasonix, and portable agents should use the same CLI/MCP surface. See `packages/hosts/*` for example hook/settings files. Host adapters are allowed to format host settings but must not duplicate OKF core logic.

## Runtime State

Hook event logging writes redacted JSONL under `.llm-wiki/hooks.jsonl` inside the current workspace. `.llm-wiki/` is ignored by git. Logs use file locks and payload caps; hooks should stay fast and should not perform model calls.

## Project Docs

`agent-harness project bootstrap --repo . --json` created this repo's AGENTS and `.agent-harness` docs. Future updates should read current docs first, then update through `project_docs_update` or append decisions through `project_docs_record`.
