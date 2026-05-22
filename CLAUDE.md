@AGENTS.md

## Claude Code

- Use plan mode for multi-file, risky, architectural, or ambiguous changes.
- Use subagents only for bounded sidecar work: exploration, review, test triage, or documentation lookup.
- Prefer separate worktrees for parallel implementation sessions.
- When a task touches `ee/`, follow `ee/AGENTS.md`; if running Claude Code inside `ee/`, `ee/CLAUDE.md` imports that overlay.
