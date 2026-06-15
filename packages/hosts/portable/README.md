# Portable Host

Portable agents only need shell access to the `llm-wiki` binary.

Use `validate --json`, `lint --json`, `graph --json`, and `query-pack --json`
for machine-readable context. Use `query-pack` as bounded context only; answer
synthesis remains the host agent's responsibility.
