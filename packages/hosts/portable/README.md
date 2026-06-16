# Portable Host

Portable agents only need shell access to the `llm-wiki` binary.

Use `validate --json`, `lint --json`, `graph --json`, and `query-pack --json`
for machine-readable context. Use `query-pack` as bounded context only; answer
synthesis remains the host agent's responsibility.

For MCP clients that understand the common `mcpServers` JSON shape, adapt
`packages/hosts/portable/mcp.example.json` and run the server as:

```bash
llm-wiki mcp
```

Set `LLM_WIKI_VAULT` in the MCP server env when tool calls should default to a
shared vault instead of requiring `path` every time.

For host-neutral smoke checks, see `docs/host-mcp-smoke.md`.
