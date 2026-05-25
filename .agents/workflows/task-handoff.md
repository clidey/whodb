---
name: task-handoff
description: Prepare or consume a concise handoff for multi-session or multi-agent work
---

# Task Handoff

Use this workflow when work will continue in another session, another worktree, or another agent. Keep the handoff short enough to read before coding.

## When to Create a Handoff

- The task is not complete and another session needs to continue it.
- A second agent or worktree needs enough context to avoid duplicating or reverting work.
- The implementation has unresolved risks, skipped verification, or important decisions that are not obvious from the diff.
- The user asks to pause, continue later, or hand work to another tool.

## Handoff Format

```markdown
## Goal
One sentence describing the intended end state.

## Current State
- What changed so far.
- What still needs to happen.
- Any user constraints that must be preserved.

## Files Touched
- `path/to/file`: why it changed.

## Decisions
- Decision made and why.

## Verification
- Command run: pass/fail and relevant result.
- Command not run: reason.

## Risks
- Known risk, blocker, or follow-up.

## Next Step
The first concrete action the next agent should take.
```

## Rules

- Be factual. Do not include speculation unless it is clearly marked as a risk or assumption.
- Mention unrelated dirty worktree changes separately so the next agent does not revert user work.
- Include exact commands, paths, and error text when they matter.
- Keep the next step actionable. Avoid vague instructions like "continue implementation."
- If the handoff is for a parallel agent, include explicit file ownership to avoid conflicts.

## Consuming a Handoff

1. Read the handoff before inspecting unrelated files.
2. Check `git status --short --untracked-files=all`.
3. Re-read the files listed under `Files Touched`.
4. Verify whether the stated next step is still valid.
5. Continue with the smallest change that moves the handoff goal forward.
