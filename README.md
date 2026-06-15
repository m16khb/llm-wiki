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

## Verification

```bash
gofmt -w .
go vet ./...
go test ./...
go run ./cmd/llm-wiki --version
go run ./cmd/llm-wiki validate fixtures/okf-minimal --json
go run ./cmd/llm-wiki validate fixtures/okf-invalid-missing-type --json
```
