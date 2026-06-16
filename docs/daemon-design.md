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
llm-wiki daemon replace --json
llm-wiki mcp
```

`status` and `doctor` exit `0` and report whether the daemon socket is
reachable. `start` starts the background daemon when it is not already running
and exits `0` with `running: true`. A running daemon may serve multiple MCP
proxy connections with different `LLM_WIKI_VAULT` values. The proxy sends the
resolved connection default to the daemon before MCP JSON-RPC bytes, and
path-optional MCP tools resolve omitted `path` from that connection default
before falling back to the daemon process environment. `stop` terminates the
current daemon if it is running and is idempotent when already stopped.

`replace` is the supported graceful handover command after replacing the
installed binary. It sends a private drain frame to the running daemon, waits
for `daemon.sock` to be released, then starts a new daemon. The drained daemon
stops accepting new sockets but keeps already accepted MCP streams alive until
their clients close. If the running daemon has no metadata or an old metadata
protocol, one hard restart is allowed because graceful control is unavailable.

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
- `daemon.meta.json`

The state directory is created with `0700`; socket, PID, lock, and log files are
created with user-only permissions where supported. `daemon.meta.json` records
the daemon protocol version, PID, executable path, executable size, and
executable mtime so a newer CLI can decide whether to reuse, drain-replace, or
hard-restart the running daemon.

The daemon socket has a private first-line frame before MCP JSON-RPC bytes:

```json
{"protocol":"llm-wiki-daemon/1","kind":"mcp","vault_path":"/abs/vault"}
```

The drain control frame is:

```json
{"protocol":"llm-wiki-daemon/1","kind":"drain"}
```

If the first socket line is not a daemon protocol frame, the daemon replays it
into the MCP stream for legacy compatibility.

## Responsibilities

The daemon owns long-lived runtime concerns:

- shared MCP backend for all host agents
- socket lifecycle and PID/status reporting
- start serialization through a lock file
- graceful replacement when executable metadata changes
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
- Manual smoke checks after rebuilding an installed binary should run
  `llm-wiki daemon replace --json` before probing host MCP behavior.
