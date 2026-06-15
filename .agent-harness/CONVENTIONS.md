---
name: CONVENTIONS.md
description: Coding conventions, package structure, and layer boundaries.
---

# Conventions

## Go Style

- Use `gofmt` for all Go files.
- Keep package names short, lowercase, and domain-specific: `okf`, `lint`, `graph`, `querypack`.
- Tests live beside the package as `*_test.go` and should verify public behavior, not private implementation details.
- JSON DTO fields use snake_case and should stay stable once exposed through CLI/MCP.

## Layer Boundaries

- CLI code in `cmd/llm-wiki` may parse flags, map exit codes, and print JSON/text. It should call internal service packages instead of duplicating behavior.
- `internal/okf` owns bundle traversal, reserved filename rules, and safe path resolution.
- `internal/frontmatter` owns YAML node preservation; callers should not rewrite frontmatter with ad hoc strings.
- `internal/validate` is OKF conformance only. Quality and graph completeness belong in `internal/lint` or other soft-warning packages.
- `internal/hooks` must remain small, deterministic, redacted, and file-lock based. Hooks must not perform expensive reads, network calls, or answer synthesis.
- `packages/hosts/*` are templates and docs only. Host-specific packages must not contain alternate OKF validators or query engines.

## Dependency Rules

- Prefer the Go standard library unless a dependency directly supports a planned contract.
- Current intentional dependencies: Cobra for CLI, yaml.v3 for node-preserving frontmatter, flock for append-safe writes, goldmark for Markdown parsing surface, and the Go MCP SDK for MCP compatibility.
- New dependencies require a concrete use, tests, and README or project-doc updates when they affect users.

## OKF Rules

- Preserve unknown frontmatter fields when round-tripping.
- Treat `index.md` and `log.md` as reserved at every hierarchy level.
- Do not reject broken links in validation; warn in lint.
- `query-pack` must return bounded context only and leave answer synthesis to the host agent.
