# Agent Instructions Index

Shared, tool-neutral agent guidance for WhoDB (Codex, Claude Code, and other
agents). Start with `../AGENTS.md`; open only what matches the task. If `ee/` is
present, `ee/.agents/README.md` indexes the EE overlay.

## Layout

| Path | Purpose |
|------|---------|
| `.agents/rules/` | Area rules with `paths:` frontmatter. Claude Code loads one when it works on matching files (through a symlink under `.claude/`); Codex users read the one for their area. |
| `.agents/skills/` | Procedures (`<name>/SKILL.md`) plus installed third-party skills. Codex reads this directory; Claude Code reads it through symlinks under `.claude/`. Both list skills by `description` and load the body when a task matches. |
| `.agents/docs/` | `sql-security.md` only (linked from the golangci lint message). |
| `.agents/hooks/` | Format, lint, codegen and build-check hooks wired into `.claude/settings.json` and `.codex/hooks.json`. |

## Rules

| Rule | Applies to |
|------|-----------|
| `go-backend.md` | Go in `core/` and `cli/`: SQL safety, plugin context, doc comments |
| `plugin-architecture.md` | Plugins, source contracts, catalogs: no database-type switches |
| `graphql.md` | Schema and resolvers: source-first API, operation authorization |
| `react-frontend.md` | Frontend: `@clidey/ux`, GraphQL hooks, JSDoc, test ids |
| `localization.md` | `t()` keys, `en_US` only; missing keys render silently |
| `e2e-tests.md` | Playwright conventions; adding an E2E database |
| `analytics.md` | PostHog contract: choke points, privacy, heartbeat |
| `cli.md` | CLI: plugins directly, identity, output, TUI gotchas |
| `desktop.md` | Wails build flow, `GetIsDesktopMode`, accelerators |
| `ci-cd.md` | Release triggers, SHA pins, arm/riscv64 stub rule |

## Procedure skills

| Task | Skill |
|------|-------|
| Add a database plugin or source-first connector | `.agents/skills/new-plugin/SKILL.md` |
| Add a GraphQL field end-to-end | `.agents/skills/new-graphql-field/SKILL.md` |
| Add a frontend page | `.agents/skills/new-frontend-page/SKILL.md` |
| Add translation keys | `.agents/skills/add-translation/SKILL.md` |
| Add CLI behavior | `.agents/skills/cli-feature/SKILL.md` |
| Add or modify desktop functionality | `.agents/skills/desktop-feature/SKILL.md` |
| Add a platform-constrained HTTP handler | `.agents/skills/platform-constrained-handler/SKILL.md` |
| Verify before finishing | `.agents/skills/review-checklist/SKILL.md` |
| Prove claims about external behavior | `.agents/skills/research-proof/SKILL.md` |
| Prepare or consume a handoff | `.agents/skills/task-handoff/SKILL.md` |

To add a procedure, create `.agents/skills/<name>/SKILL.md` with `name` and a
`description` that says when to use it, then link it for Claude Code:
`ln -s ../../.agents/skills/<name> .claude/skills/<name>` from the repo root.

## Usage

- Keep guidance where it is used: area gotchas in rules, procedures in skills,
  package facts in Go package comments. Don't restate what the code shows.
- Name files by their path from the repo root; the checker verifies they exist.
- After changing agent guidance, run `dev/check-agent-instructions.sh`.
