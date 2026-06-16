# Reasonix Host

Reasonix integration follows the same thin-wrapper model as Claude Code. The
host invokes `llm-wiki` through CLI or MCP and does not carry separate OKF core
logic.

## MCP

Prefer the shared setup command:

```bash
llm-wiki setup-hosts --apply --json
```

For manual setup, use `packages/hosts/reasonix/reasonix.example.toml` as a
project-level `reasonix.toml` fragment:

```toml
[[plugins]]
name = "llm-wiki"
type = "stdio"
command = "llm-wiki"
args = ["mcp"]
env = { LLM_WIKI_VAULT = "/path/to/llm-wiki-vault" }
```

Reasonix can also read the common project-root `.mcp.json` shape; use the
portable template when that is a better fit for the repository.
The setup command writes `LLM_WIKI_VAULT`; pass `--vault` to override the
default `$HOME/workspace/knowledge-base/llm-wiki`.
For the full smoke path, see `docs/host-mcp-smoke.md`.

Suggested hook command shape:

```bash
llm-wiki hook PromptSubmit --host reasonix --json
llm-wiki hook PostToolUse --host reasonix --json
llm-wiki hook Stop --host reasonix --json
```
