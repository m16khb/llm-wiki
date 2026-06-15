# Claude Code Host

Claude Code integration is a thin wrapper around the shared `llm-wiki` binary.
Keep host-specific files limited to settings, commands, and skill copy that call
the CLI or MCP server.

## MCP

For a project-scoped MCP server, use the template in
`packages/hosts/claude/mcp.example.json` as the contents of a project-root
`.mcp.json`, or add the same server with:

```bash
claude mcp add --transport stdio --scope project llm-wiki -- llm-wiki mcp
```

Claude Code will ask for approval before using project-scoped MCP servers. Use
`/mcp` inside Claude Code or `claude mcp list` to inspect connection status.
For the full smoke path, see `docs/host-mcp-smoke.md`.

Suggested hook command shape:

```bash
llm-wiki hook UserPromptSubmit --host claude --json
llm-wiki hook PostToolUse --host claude --json
llm-wiki hook Stop --host claude --json
```
