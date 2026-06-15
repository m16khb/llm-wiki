# Reasonix Host

Reasonix integration follows the same thin-wrapper model as Claude Code. The
host invokes `llm-wiki` through CLI or MCP and does not carry separate OKF core
logic.

Suggested hook command shape:

```bash
llm-wiki hook PromptSubmit --host reasonix --json
llm-wiki hook PostToolUse --host reasonix --json
llm-wiki hook Stop --host reasonix --json
```
