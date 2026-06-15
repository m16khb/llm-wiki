---
name: TESTING.md
description: Verification standards, test practices, and required checks.
---

# Testing

## Required Local Checks

Run these before claiming completion for code changes:

```bash
gofmt -w cmd internal
go vet ./...
go test ./...
go test ./internal/snapshots
go run ./cmd/llm-wiki --version
go run ./cmd/llm-wiki setup-hosts --json
go run ./cmd/llm-wiki daemon status --json
go run ./cmd/llm-wiki validate fixtures/okf-minimal --json
go run ./cmd/llm-wiki validate fixtures/okf-invalid-missing-type --json
go run ./cmd/llm-wiki mcp < /dev/null
```

The invalid fixture is expected to return exit code 1 while emitting the validation DTO with `ok: false`.

## Behavior Coverage

Current tests cover:

- frontmatter parse/write and unknown-field preservation
- missing frontmatter and missing `type` validation
- invalid UTF-8 rejection for concept and reserved Markdown files
- `index.md`/`log.md` exclusion from concept count at every hierarchy level
- root `index.md` `okf_version` allowance and non-root index frontmatter rejection
- `log.md` date heading validation for `YYYY-MM-DD`
- nested concept path stability
- broken link warning in lint without validation failure
- safe write path rejection for traversal and symlink escape
- locked append behavior for log writes
- hook JSONL redaction, payload cap, and 50-writer concurrency
- host hook output shapes for Codex, Claude Code, and Reasonix
- deterministic graph edges from wiki links
- query-pack bounded context and no synthesized answer
- MCP SDK in-memory server tool listing and `llm_wiki_validate`/`llm_wiki_query_pack` calls
- normalized CLI JSON golden snapshots for validation and query-pack DTOs
- daemon state-path resolution, unsupported start/stop behavior, and daemon CLI JSON snapshots
- fixture-level NVK dry-run planning

## Test Style

- Add tests before production behavior for new features or bug fixes.
- Prefer fixture-level tests for CLI-visible OKF behavior.
- Use MCP SDK in-memory transports for MCP handler behavior; do not require installed host agents.
- Normalize dynamic fields in golden tests, especially timestamps, temp paths, and generated absolute roots. CLI snapshots live under `testdata/snapshots/`.
- Do not depend on network, real user home configuration, or host-specific installed agents.
- Host end-to-end checks are documented as optional smoke procedures. Normal tests must stay runnable without Claude Code, Codex, or Reasonix installed.

## CI

`.github/workflows/ci.yml` uses `actions/checkout@v6` and `actions/setup-go@v6`, then runs formatting, vet, tests, and CLI smoke checks.
