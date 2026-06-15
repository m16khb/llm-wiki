---
name: CAUTIONS.md
description: Recurring mistakes, operational cautions, and avoidance guidance.
---

# Cautions

- Generated docs are drafts; directly verify weak evidence.
- Do not commit secrets, credentials, local state, or generated artifacts.
- CI workflows exist; compare local verification with CI behavior.
- `llm-wiki mcp` auto-starts a user-level daemon. After manually replacing the
  installed binary, stop any running daemon before MCP smoke checks if behavior
  appears stale.
