# Claude Code Host

Claude Code integration is a thin wrapper around the shared `llm-wiki` binary.
Keep host-specific files limited to settings, commands, and skill copy that call
the CLI or MCP server.

Suggested hook command shape:

```bash
llm-wiki hook UserPromptSubmit --host claude --json
llm-wiki hook PostToolUse --host claude --json
llm-wiki hook Stop --host claude --json
```
