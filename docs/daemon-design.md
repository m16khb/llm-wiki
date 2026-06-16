# Daemon Design

`llm-wiki` runs MCP through a shared user-level daemon:

```bash
llm-wiki mcp
```

The `mcp` command is a stdio proxy. It ensures the daemon is running, connects
to the daemon Unix socket, and forwards MCP JSON-RPC bytes between the host and
the daemon. Claude Code, Codex, Reasonix, and portable MCP clients therefore
use the same backend process instead of each embedding separate OKF behavior.

## Runtime Contract

The visible daemon surface is:

```bash
llm-wiki daemon status --json
llm-wiki daemon doctor --json
llm-wiki daemon start --json
llm-wiki daemon stop --json
llm-wiki mcp
```

`status` and `doctor` exit `0` and report whether the daemon socket is
reachable. `start` starts the background daemon when it is not already running
and exits `0` with `running: true`. If the running daemon's `LLM_WIKI_VAULT`
does not match the caller's environment, `start` restarts it so path-optional
MCP tools resolve the same default vault as the host proxy. `stop` terminates
the daemon if it is running and is idempotent when already stopped.

`llm-wiki mcp --daemon` is accepted as a compatibility no-op; daemon-backed MCP
is now the default behavior of `llm-wiki mcp`.

## State And IPC Paths

Path resolution is deterministic:

1. `LLM_WIKI_STATE_DIR`
2. `$XDG_STATE_HOME/llm-wiki`
3. `~/.local/state/llm-wiki`

Runtime files are:

- `daemon.sock`
- `daemon.pid`
- `daemon.lock`
- `daemon.log`

The state directory is created with `0700`; socket, PID, lock, and log files are
created with user-only permissions where supported.

## Responsibilities

The daemon owns long-lived runtime concerns:

- shared MCP backend for all host agents
- socket lifecycle and PID/status reporting
- start serialization through a lock file
- MCP stream serving through the same internal service layer as the CLI

It must not duplicate OKF validation, linting, graphing, indexing, or
query-pack logic. Those remain in the internal service packages used by both
CLI commands and MCP tools.

## Safety Rules

- Host templates should keep calling plain `llm-wiki mcp`; that command is the
  daemon-backed proxy.
- Tests that start a daemon must set `LLM_WIKI_STATE_DIR` to an isolated temp
  directory and stop the daemon during cleanup.
- Runtime files must stay under the resolved state directory.
- Manual smoke checks after rebuilding an installed binary should stop any old
  daemon first if behavior looks stale.
