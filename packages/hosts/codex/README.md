# Codex Host

Codex integration is a thin wrapper around the shared `llm-wiki` binary.
Do not duplicate validation, linting, indexing, graph, or query-pack logic in a
Codex-specific plugin.

## MCP

Prefer the shared setup command:

```bash
llm-wiki setup-hosts --apply --vault "$HOME/workspace/knowledge-base/llm-wiki" --json
```

For manual setup, copy `packages/hosts/codex/config.example.toml` into the
relevant Codex configuration file and keep the server process as `llm-wiki mcp`.
Set `LLM_WIKI_VAULT` in the MCP server env when tool calls should default to a
shared vault instead of requiring `path` every time.

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
