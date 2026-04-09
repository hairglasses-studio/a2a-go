# a2a-go Roadmap

Last updated: 2026-04-08.

## Current State

Forked A2A Go SDK used for protocol work and upstream contributions.

- Tier: `tier-1`
- Lifecycle: `active`
- Language profile: `Go`
- Visibility / sensitivity: `PUBLIC` / `public`
<!-- whiteclaw-rollout:start -->
## Whiteclaw-Derived Overhaul (2026-04-08)

This tranche applies the highest-value whiteclaw findings that fit this repo's real surface: engineer briefs, bounded skills/runbooks, searchable provenance, scoped MCP packaging, and explicit verification ladders.

### Strategic Focus
- Keep the whiteclaw backport thin here: this repo is a public upstream SDK fork, not a target for heavy autonomy scaffolding.
- Use whiteclaw patterns to improve maintainer clarity, contribution repeatability, and spec-drift tracking.
- Avoid forcing MCP-sidecar or `.ralph` patterns into a library repo that mainly needs docs, examples, and release hygiene.

### Recommended Work
- [ ] [Instructions] Fix the canonical-instructions contradiction between `AGENTS.md` and `CLAUDE.md` so maintainers see one source of truth.
- [ ] [Skills] Add the first repo skill for the maintainer loop: build, test, examples, upstream sync, and release validation.
- [ ] [Spec drift] Track A2A spec changes, upstream divergence, and downstream integration breakage in this roadmap instead of leaving it in session notes.
- [ ] [Verification] Add example-smoke verification for the stable server/client/example loops documented in the README.
- [ ] [Public docs] Document exactly which whiteclaw patterns are intentionally excluded here so the repo stays thin and contributor-friendly.

### Rationale Snapshot
- Tier / lifecycle: `tier-1` / `active`
- Language profile: `Go`
- Visibility / sensitivity: `PUBLIC` / `public`
- Surface baseline: AGENTS=yes, skills=no, codex=yes, mcp_manifest=missing, ralph=no, roadmap=yes
- Whiteclaw transfers in scope: maintainer skill, example smoke tests, spec-drift tracking, public maintainer docs
- Live repo notes: AGENTS, Codex config, 13 workflow(s), multi-module/workspace

<!-- whiteclaw-rollout:end -->
