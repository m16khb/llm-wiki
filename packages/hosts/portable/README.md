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

For host-neutral smoke checks, see `docs/host-mcp-smoke.md`.
