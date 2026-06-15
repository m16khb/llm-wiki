# OKF v0.1 Compatibility

`llm-wiki` targets Open Knowledge Format (OKF) v0.1 as a local-first,
host-neutral toolkit. This repository vendors the upstream Google OKF v0.1 spec
as local reference material under `third_party/google-okf/`.

## Upstream References

- Specification: https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md
- Announcement: https://cloud.google.com/blog/products/data-analytics/how-the-open-knowledge-format-can-improve-data-sharing
- Repository license: https://github.com/GoogleCloudPlatform/knowledge-catalog
- Vendored copy: `third_party/google-okf/SPEC.md`

The upstream repository states that its solutions are Apache-2.0 licensed. This
repo includes `third_party/google-okf/LICENSE.md` and `SOURCE.md` alongside the
vendored spec.

## Implemented Contract

- A bundle is a directory tree of UTF-8 Markdown files.
- Non-reserved `.md` files are concept documents.
- Concept documents require parseable YAML frontmatter with a non-empty `type`
  field.
- `index.md` and `log.md` are reserved filenames at every hierarchy level and
  are excluded from concept counts.
- `index.md` files must not contain frontmatter, except the bundle-root
  `index.md` may declare only `okf_version`.
- `log.md` date headings must use ISO 8601 `YYYY-MM-DD`.
- Optional and unknown frontmatter fields are tolerated and preserved by the
  frontmatter round-trip path.
- Broken links are lint warnings, not validation errors.
- `query-pack` returns bounded context only; it does not synthesize answers.
- `llm-wiki mcp` exposes MCP stdio tools backed by the same CLI service layer.

## Current Scope

This initial implementation intentionally avoids direct LLM API calls, remote
storage, vector search, desktop UI, and Obsidian plugin behavior. Claude Code,
Codex, Reasonix, and portable hosts should all call the same CLI/MCP surface.
