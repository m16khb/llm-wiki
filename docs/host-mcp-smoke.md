# Host MCP Smoke Tests

`llm-wiki` exposes one daemon-backed stdio MCP proxy for Claude Code, Codex,
Reasonix, and portable MCP clients:

```bash
llm-wiki mcp
```

The host packages under `packages/hosts/` are templates only. They must keep
calling the shared binary instead of reimplementing OKF validation, linting,
indexing, graphing, or query-pack logic.

Host templates should continue to use plain `llm-wiki mcp`; that command
auto-starts or connects to the shared daemon. `llm-wiki mcp --daemon` is only a
compatibility no-op.

## Baseline

Run these checks before debugging a host-specific configuration:

```bash
go test ./internal/mcp
tmp_state="$(mktemp -d)" && LLM_WIKI_STATE_DIR="$tmp_state" go run ./cmd/llm-wiki mcp < /dev/null; LLM_WIKI_STATE_DIR="$tmp_state" go run ./cmd/llm-wiki daemon stop --json; rm -rf "$tmp_state"
tmp_state="$(mktemp -d)" && LLM_WIKI_STATE_DIR="$tmp_state" go run ./cmd/llm-wiki daemon start --json && LLM_WIKI_STATE_DIR="$tmp_state" go run ./cmd/llm-wiki daemon status --json && LLM_WIKI_STATE_DIR="$tmp_state" go run ./cmd/llm-wiki daemon stop --json; rm -rf "$tmp_state"
go run ./cmd/llm-wiki validate fixtures/okf-minimal --json
go run ./cmd/llm-wiki query-pack fixtures/okf-minimal alpha --json
```

The `mcp < /dev/null` smoke should exit `0` and auto-start the daemon in the
isolated state directory. `go test ./internal/mcp` verifies tool listing and
representative calls through the Go MCP SDK in-memory transport.

## Host CLI Probe

These checks confirm that the host CLI can see an `llm-wiki` MCP entry without
modifying the user's normal host configuration. They do not approve tools or run
an LLM session.

## First-Time Setup

Use the shared setup command before host-specific debugging:

```bash
llm-wiki setup-hosts --json
llm-wiki setup-hosts --apply --json
```

The dry-run reports the files that would change. `--apply` writes:

- Codex user MCP config at `~/.codex/config.toml`
- Claude Code project MCP config at `.mcp.json`
- Reasonix project plugin config at `reasonix.toml`

All three entries call the same `llm-wiki mcp` binary. Host configs pass
`LLM_WIKI_VAULT` so MCP tool calls can omit `path`; `--vault` overrides the
path, and omitted values default to `$HOME/workspace/knowledge-base/llm-wiki`.
The command does not remove legacy plugins or caches; clean those up separately
if a host still loads an old integration.

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

[mcp_servers.llm-wiki.env]
LLM_WIKI_VAULT = "/path/to/llm-wiki-vault"
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

Prefer `llm-wiki setup-hosts --apply`. For manual setup, use
`packages/hosts/claude/mcp.example.json` as project-root `.mcp.json`, or run:

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

Prefer `llm-wiki setup-hosts --apply`. For manual setup, copy
`packages/hosts/codex/config.example.toml` into `~/.codex/config.toml` or a
trusted project `.codex/config.toml`:

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
5. If `LLM_WIKI_VAULT` is configured, ask Codex to call `llm_wiki_validate`
   without `path` and confirm the result uses the configured vault.
6. Ask Codex to call `llm_wiki_query_pack` with question `alpha`.
7. Confirm the result has `context_only: true` and no synthesized answer.

## Reasonix

Prefer `llm-wiki setup-hosts --apply`. For manual setup, use
`packages/hosts/reasonix/reasonix.example.toml` as a project `reasonix.toml`
fragment:

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
5. If `LLM_WIKI_VAULT` is configured, ask Reasonix to call
   `llm_wiki_validate` without `path` and confirm the result uses the
   configured vault.
6. Ask Reasonix to call `llm_wiki_graph` on `fixtures/okf-minimal`.
7. Confirm the graph contains the `alpha.md` node.

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
