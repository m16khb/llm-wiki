# llm-wiki

`llm-wiki` is a Go, OKF-native toolkit for local-first LLM wiki bundles.
It provides one CLI/MCP contract that Claude Code, Codex, Reasonix, and
portable shell-based agents can all call.

## Status

This is an initial scaffold targeting OKF v0.1. It intentionally keeps LLM
synthesis outside the core: host agents may read query packs and decide what to
write, while `llm-wiki` validates, lints, indexes, logs, graphs, and packages
bounded context.

## Install From Source

```bash
go install github.com/m16khb/llm-wiki/cmd/llm-wiki@latest
```

For local development:

```bash
go run ./cmd/llm-wiki --version
go test ./...
```

## CLI Contract

```bash
llm-wiki init <path> --profile obsidian --okf-version 0.1
llm-wiki validate <path> --json
llm-wiki lint <path> --json
llm-wiki lint <path> --fix
llm-wiki index <path> --write
llm-wiki log <path> append --op <op> --message <text>
llm-wiki graph <path> --json
llm-wiki query-pack <path> "<question>" --json
llm-wiki import nvk <source> <dest> --dry-run
llm-wiki export nvk <source> <dest> --dry-run
llm-wiki hook <event> --host <claude|codex|reasonix> --json
llm-wiki setup-hosts --apply --json
llm-wiki daemon status --json
llm-wiki mcp
```

The stable validation DTO starts as:

```json
{
  "ok": true,
  "okf_version": "0.1",
  "bundle_root": "/abs/path",
  "concept_count": 1,
  "reserved_files": ["index.md", "log.md"],
  "errors": [],
  "warnings": []
}
```

## OKF Compatibility

See [docs/okf-v0.1-compat.md](docs/okf-v0.1-compat.md). A vendored copy of the
upstream Google OKF v0.1 spec is kept under
[third_party/google-okf](third_party/google-okf) with its Apache-2.0 license and
source metadata.

## Host Integrations

Host packages in `packages/hosts/` are intentionally thin. They document how a
host should invoke the same `llm-wiki` binary without duplicating core logic.
See [docs/host-mcp-smoke.md](docs/host-mcp-smoke.md) for Claude Code, Codex,
Reasonix, and portable MCP smoke-test steps.

For first-time setup from an installed binary, run a dry-run first and then
apply:

```bash
llm-wiki setup-hosts --json
llm-wiki setup-hosts --apply --json
```

This writes Codex user MCP config plus project-local Claude Code and Reasonix
MCP config that all call the same `llm-wiki mcp` binary. It does not remove old
plugins or caches; delete legacy host integrations separately when needed.

## Runtime Strategy

The supported runtime path is daemon-backed stdio MCP:

```bash
llm-wiki mcp
```

`llm-wiki mcp` auto-starts or connects to the shared user-level daemon, then
proxies MCP JSON-RPC bytes to the daemon socket. `llm-wiki daemon start`,
`status`, `doctor`, and `stop` manage that backend process. `llm-wiki
mcp --daemon` is accepted as a compatibility no-op because daemon-backed MCP is
now the default.

See [docs/daemon-design.md](docs/daemon-design.md) for the daemon contract.

## MCP Tools

`llm-wiki mcp` exposes an MCP stdio proxy backed by the same service layer as
the CLI. Initial tools:

- `llm_wiki_validate`
- `llm_wiki_lint`
- `llm_wiki_index`
- `llm_wiki_graph`
- `llm_wiki_query_pack`

## Verification

```bash
gofmt -w cmd internal
go vet ./...
go test ./...
go test ./internal/snapshots
go run ./cmd/llm-wiki --version
go run ./cmd/llm-wiki daemon status --json
go run ./cmd/llm-wiki validate fixtures/okf-minimal --json
go run ./cmd/llm-wiki validate fixtures/okf-invalid-missing-type --json
```
