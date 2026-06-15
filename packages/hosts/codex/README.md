# Codex Host

Codex integration is a thin wrapper around the shared `llm-wiki` binary.
Do not duplicate validation, linting, indexing, graph, or query-pack logic in a
Codex-specific plugin.

Suggested hook command shape:

```bash
llm-wiki hook UserPromptSubmit --host codex --json
llm-wiki hook PostToolUse --host codex --json
llm-wiki hook Stop --host codex --json
```
