# Host MCP Smoke Tests

`llm-wiki` exposes one stdio MCP server for Claude Code, Codex, Reasonix, and
portable MCP clients:

```bash
llm-wiki mcp
```

The host packages under `packages/hosts/` are templates only. They must keep
calling the shared binary instead of reimplementing OKF validation, linting,
indexing, graphing, or query-pack logic.

## Baseline

Run these checks before debugging a host-specific configuration:

```bash
go test ./internal/mcp
go run ./cmd/llm-wiki mcp < /dev/null
go run ./cmd/llm-wiki validate fixtures/okf-minimal --json
go run ./cmd/llm-wiki query-pack fixtures/okf-minimal alpha --json
```

`go run ./cmd/llm-wiki mcp < /dev/null` should exit `0`. It only proves that the
stdio server starts and handles EOF; `go test ./internal/mcp` verifies tool
listing and representative calls through the Go MCP SDK in-memory transport.

## Host CLI Probe

These checks confirm that the host CLI can see an `llm-wiki` MCP entry without
modifying the user's normal host configuration. They do not approve tools or run
an LLM session.

Claude Code:

```bash
claude --version
claude mcp --help
```

Codex, using a temporary `CODEX_HOME`:

```bash
tmp="$(mktemp -d)"
cat >"$tmp/config.toml" <<'EOF'
[mcp_servers.llm-wiki]
command = "llm-wiki"
args = ["mcp"]
startup_timeout_sec = 10
tool_timeout_sec = 60
EOF
CODEX_HOME="$tmp" codex mcp list
rm -rf "$tmp"
```

Expected Codex list fields include `llm-wiki`, command `llm-wiki`, args `mcp`,
and status `enabled`.

Reasonix, using a temporary project directory:

```bash
tmp="$(mktemp -d)"
cat >"$tmp/reasonix.toml" <<'EOF'
[[plugins]]
name = "llm-wiki"
type = "stdio"
command = "llm-wiki"
args = ["mcp"]
EOF
(cd "$tmp" && reasonix mcp list)
rm -rf "$tmp"
```

Expected Reasonix list output includes `llm-wiki (stdio) llm-wiki mcp`.

## Claude Code

Use `packages/hosts/claude/mcp.example.json` as project-root `.mcp.json`, or run:

```bash
claude mcp add --transport stdio --scope project llm-wiki -- llm-wiki mcp
claude mcp list
```

Then start Claude Code from the project and inspect `/mcp`. Project-scoped
servers require explicit approval before tools are available.

End-to-end confirmation:

1. Start Claude Code from the repository root.
2. Approve the project-scoped `llm-wiki` MCP server when prompted.
3. Open `/mcp` and confirm the `llm-wiki` server is connected.
4. Ask Claude Code to call `llm_wiki_validate` on `fixtures/okf-minimal`.
5. Confirm the result has `ok: true` and `concept_count: 1`.

## Codex

Copy `packages/hosts/codex/config.example.toml` into `~/.codex/config.toml` or
a trusted project `.codex/config.toml`:

```toml
[mcp_servers.llm-wiki]
command = "llm-wiki"
args = ["mcp"]
startup_timeout_sec = 10
tool_timeout_sec = 60
```

Restart Codex after changing MCP config. If `llm-wiki` is not on `PATH`, use an
absolute `command`.

End-to-end confirmation:

1. Start Codex from the repository root after the MCP config is in place.
2. Inspect the MCP servers panel or run `codex mcp list` and confirm
   `llm-wiki` is enabled.
3. Ask Codex to call `llm_wiki_validate` on `fixtures/okf-minimal`.
4. Confirm the result has `ok: true` and `concept_count: 1`.
5. Ask Codex to call `llm_wiki_query_pack` with question `alpha`.
6. Confirm the result has `context_only: true` and no synthesized answer.

## Reasonix

Use `packages/hosts/reasonix/reasonix.example.toml` as a project
`reasonix.toml` fragment:

```toml
[[plugins]]
name = "llm-wiki"
type = "stdio"
command = "llm-wiki"
args = ["mcp"]
```

Start Reasonix and inspect `/mcp`. Reasonix can also consume the common
project-root `.mcp.json` shape; use `packages/hosts/portable/mcp.example.json`
for that path.

End-to-end confirmation:

1. Start `reasonix chat` from the project directory.
2. Inspect `/mcp` and confirm the `llm-wiki` server is connected.
3. Ask Reasonix to call `llm_wiki_validate` on `fixtures/okf-minimal`.
4. Confirm the result has `ok: true` and `concept_count: 1`.
5. Ask Reasonix to call `llm_wiki_graph` on `fixtures/okf-minimal`.
6. Confirm the graph contains the `alpha.md` node.

## Expected Tools

All hosts should expose the same tool names:

- `llm_wiki_validate`
- `llm_wiki_lint`
- `llm_wiki_index`
- `llm_wiki_graph`
- `llm_wiki_query_pack`

The query-pack tool returns bounded context only. Host agents remain
responsible for synthesis.

## References

- Claude Code MCP: https://docs.anthropic.com/en/docs/claude-code/mcp
- Codex config reference: https://developers.openai.com/codex/config-reference
- Reasonix MCP guide: https://github.com/esengine/DeepSeek-Reasonix/blob/main-v2/docs/GUIDE.md
