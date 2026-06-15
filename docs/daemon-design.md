# Daemon Design

`llm-wiki` currently runs MCP through direct stdio:

```bash
llm-wiki mcp
```

This remains the default and supported integration path for Claude Code, Codex,
Reasonix, and portable MCP clients. The daemon CLI is a reserved contract for a
future runtime, not an active process supervisor.

## Current Contract

The visible daemon surface is:

```bash
llm-wiki daemon status --json
llm-wiki daemon doctor --json
llm-wiki daemon start --json
llm-wiki daemon stop --json
llm-wiki mcp --daemon
```

`status` and `doctor` exit `0` and report `implemented: false` and
`running: false`. `start` and `stop` emit the same structured DTO, set
`ok: false`, and exit `2` because the daemon runtime is unsupported.
`llm-wiki mcp --daemon` is reserved for future daemon-backed MCP and currently
returns an unsupported error. Plain `llm-wiki mcp` is unchanged.

## State And IPC Paths

Path resolution is deterministic and does not create directories or files:

1. `LLM_WIKI_STATE_DIR`
2. `$XDG_STATE_HOME/llm-wiki`
3. `~/.local/state/llm-wiki`

Reserved future files are:

- `daemon.sock`
- `daemon.pid`
- `daemon.lock`

## Future Responsibilities

A future daemon may own long-lived runtime concerns such as shared local state,
IPC, cache coordination, background indexing, host-neutral worker queues, and
MCP proxying. Those responsibilities should stay behind the same CLI/MCP
contract and must not duplicate OKF validation, linting, graphing, indexing, or
query-pack logic.

## Non-Goals

The current skeleton does not start processes, bind sockets, create lock or PID
files, spawn goroutines, run listeners, cache bundle data, watch files, or proxy
MCP requests.

## Rollout Trigger

Implement the runtime only when a concrete workflow needs shared process state
that direct stdio MCP cannot provide. Until then, host integrations must keep
using `llm-wiki mcp`.

## Safety Rules

- `start` and `stop` must remain side-effect-free until the daemon runtime is
  implemented intentionally.
- Runtime files must stay under the resolved state directory.
- Host templates must not opt into `mcp --daemon` before daemon-backed MCP is
  implemented.
- Any future runtime must preserve direct `llm-wiki mcp` as a smoke-tested
  fallback.
