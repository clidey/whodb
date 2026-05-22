# Agent Instructions Index

This directory contains shared agent guidance for WhoDB. It is intentionally tool-neutral so Codex, Claude Code, opencode, Pi, and other coding agents can use the same source of truth.

Start with `../AGENTS.md`. Use this index only to find the one or two deeper references that match the task.

## Directory Layout

| Path | Purpose |
|------|---------|
| `.agents/docs/` | Detailed runbooks and reference material for specific systems or policies. |
| `.agents/rules/` | Domain-specific rules for backend, frontend, GraphQL, localization, and E2E work. |
| `.agents/workflows/` | Step-by-step procedures for recurring implementation, review, handoff, and research tasks. |

If `ee/` is present, also read `ee/AGENTS.md` for Enterprise Edition boundaries and use `ee/.agents/` for EE-specific rules, workflows, and docs.

## Common Workflows

| Task | Read |
|------|------|
| Add a database plugin | `.agents/workflows/new-plugin.md` |
| Add a GraphQL field end-to-end | `.agents/workflows/new-graphql-field.md` |
| Add a frontend page | `.agents/workflows/new-frontend-page.md` |
| Add translation keys | `.agents/workflows/add-translation.md` |
| Add CLI behavior | `.agents/workflows/cli-feature.md` |
| Prepare or consume a handoff | `.agents/workflows/task-handoff.md` |
| Prove claims about external behavior | `.agents/workflows/research-proof.md` |
| Verify before finishing | `.agents/workflows/review-checklist.md` |

## Usage Rules

- Do not read the whole `.agents` tree by default. Start with `AGENTS.md`, then open only the relevant rule, workflow, or doc.
- Keep tool-specific instructions in adapter files such as `CLAUDE.md`; keep shared behavior here.
- Do not move user-facing product documentation into this directory. Public docs belong under `docs/`.
- When adding new agent guidance, prefer a focused workflow or rule over expanding always-loaded instructions.
- After changing agent instructions, run `dev/check-agent-instructions.sh`.
