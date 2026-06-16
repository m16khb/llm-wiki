# Turing Evidence

This directory contains evidence captured while verifying `llm-wiki` behavior
and the configured knowledge vault.

## Goals

- `G1`: verify that `llm-wiki` can validate OKF bundles and retrieve bounded
  context through CLI/service code and the active MCP server.
- `G2`: audit `/Users/habin/workspace/knowledge-base/llm-wiki` for OKF
  conformance and llm-wiki retrieval suitability.

## Evidence Map

- `goals.json`: criteria, status, evidence paths, and cleanup receipts.
- `ledger.jsonl`: append-only pass/completion records for each criterion.
- `evidence/G1-*.txt`: CLI, MCP, daemon, and final verification evidence for
  the llm-wiki information-collection check.
- `evidence/G2-*.txt`: vault validation, lint, graph, supplemental content
  scan, MCP retrieval, and final verification evidence.

The evidence is read-only audit output. It does not modify the knowledge vault.
