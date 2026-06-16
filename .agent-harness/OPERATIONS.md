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

## Host Setup

First-time host setup should use the shared setup command:

```bash
llm-wiki setup-hosts --json
llm-wiki setup-hosts --apply --json
```

This writes Codex user MCP config plus project-local Claude Code and Reasonix
MCP config that all call `llm-wiki mcp`. Host configs pass `LLM_WIKI_VAULT`;
`--vault` overrides the path, and omitted values default to
`$HOME/workspace/knowledge-base/llm-wiki`. Non-JSON interactive apply prompts
for the vault path and accepts an empty input as that default. It intentionally
does not delete old plugins or caches; legacy cleanup is a separate explicit
maintenance step.

## OKF Bundle Workflow

- Initialize: `llm-wiki init <path> --profile obsidian --okf-version 0.1`
- Validate hard conformance: `llm-wiki validate <path> --json`
- Lint soft quality issues: `llm-wiki lint <path> --json`
- Apply safe index fixes: `llm-wiki lint <path> --fix` or `llm-wiki index <path> --write`
- Append history: `llm-wiki log <path> append --op <op> --message <text>`
- Build context: `llm-wiki graph <path> --json` and `llm-wiki query-pack <path> "<question>" --json`

Set `LLM_WIKI_VAULT` to define the default OKF bundle root for CLI commands and
MCP tools when the caller omits `path`:

```bash
export LLM_WIKI_VAULT="$HOME/workspace/knowledge-base/llm-wiki"
llm-wiki validate --json
llm-wiki query-pack "alpha" --json
```

Explicit path arguments still win over `LLM_WIKI_VAULT`.

## MCP

Run `llm-wiki mcp` as the stdio MCP proxy. It auto-starts or connects to the shared user-level daemon, then proxies MCP JSON-RPC bytes to the daemon socket. Multiple host agents can each run a proxy, but they should converge on one daemon for the same daemon state directory. Initial tools are `llm_wiki_validate`, `llm_wiki_lint`, `llm_wiki_index`, `llm_wiki_graph`, and `llm_wiki_query_pack`. These tools call the same internal service packages used by the CLI.

Baseline MCP smoke:

```bash
go test ./internal/mcp
tmp_state="$(mktemp -d)" && LLM_WIKI_STATE_DIR="$tmp_state" go run ./cmd/llm-wiki mcp < /dev/null; LLM_WIKI_STATE_DIR="$tmp_state" go run ./cmd/llm-wiki daemon stop --json; rm -rf "$tmp_state"
tmp_state="$(mktemp -d)" && LLM_WIKI_STATE_DIR="$tmp_state" go run ./cmd/llm-wiki daemon start --json && LLM_WIKI_STATE_DIR="$tmp_state" go run ./cmd/llm-wiki daemon status --json && LLM_WIKI_STATE_DIR="$tmp_state" go run ./cmd/llm-wiki daemon stop --json; rm -rf "$tmp_state"
```

Keep host configuration on plain `llm-wiki mcp`; that command is already daemon-backed.

## Host Integrations

Claude Code, Codex, Reasonix, and portable agents should use the same CLI/MCP surface. See `packages/hosts/*` for example hook/settings and MCP config files, and `docs/host-mcp-smoke.md` for host-specific smoke steps. Host adapters are allowed to format host settings but must not duplicate OKF core logic. Host smoke work should start with the non-mutating probes in `docs/host-mcp-smoke.md` before changing user-level host configuration.

## Runtime State

Hook event logging writes redacted JSONL under `.llm-wiki/hooks.jsonl` inside the current workspace. `.llm-wiki/` is ignored by git. Logs use file locks and payload caps; hooks should stay fast and should not perform model calls.

The daemon resolves runtime state in this order:

1. `LLM_WIKI_STATE_DIR`
2. `$XDG_STATE_HOME/llm-wiki`
3. `~/.local/state/llm-wiki`

Daemon files are `daemon.sock`, `daemon.pid`, `daemon.lock`, and `daemon.log`.
`daemon status` and `daemon doctor` are safe probes. `daemon start` creates the
state directory and starts the socket server when needed; it also restarts a
running state-dir daemon whose `LLM_WIKI_VAULT` differs from the caller's
environment, and best-effort stops stale sibling daemon processes that use the
same resolved state directory while leaving other state directories alone.
`daemon stop` terminates the state-dir daemon and is safe to run when already
stopped.

## Project Docs

`agent-harness project bootstrap --repo . --json` created this repo's AGENTS and `.agent-harness` docs. Future updates should read current docs first, then update through `project_docs_update` or append decisions through `project_docs_record`.
