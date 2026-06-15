# Codex Host

Codex integration is a thin wrapper around the shared `llm-wiki` binary.
Do not duplicate validation, linting, indexing, graph, or query-pack logic in a
Codex-specific plugin.

## MCP

Copy `packages/hosts/codex/config.example.toml` into the relevant Codex
configuration file and keep the server process as `llm-wiki mcp`.

User-level Codex configuration lives in `~/.codex/config.toml`. Project-scoped
configuration can live in `.codex/config.toml` for trusted projects. If the
binary is not on `PATH`, replace `command = "llm-wiki"` with an absolute path.
For the full smoke path, see `docs/host-mcp-smoke.md`.

Suggested hook command shape:

```bash
llm-wiki hook UserPromptSubmit --host codex --json
llm-wiki hook PostToolUse --host codex --json
llm-wiki hook Stop --host codex --json
```
