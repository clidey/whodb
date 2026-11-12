---
name: research-proof
description: Prove claims about external tools, APIs, libraries, services, standards, or current behavior
---

# Research Proof

Use this workflow when making claims about behavior outside this repository, especially third-party APIs, dependencies, hosted services, standards, current pricing, product capabilities, or agent tooling.

## When Proof Is Required

- The claim affects implementation, security, billing, auth, deployments, compatibility, or user-facing behavior.
- The claim is about a third-party library, framework, CLI, cloud service, database, browser API, or protocol.
- The claim may have changed recently.
- The user asks whether something is true, current, supported, recommended, or safe.
- A repo rule explicitly says to show proof.

## Preferred Sources

1. Official documentation from the vendor or standards body.
2. Actual source code from the dependency, SDK, or CLI implementation.
3. Release notes, changelogs, or migration guides from the project owner.
4. Reproducible local evidence from this repository or a minimal command.

Avoid relying on blogs, forum answers, AI summaries, or stale issue comments when an official source or source code is available.

## Workflow

1. State the specific external claim that needs proof.
2. Find an official source or actual implementation that answers that claim.
3. Prefer the narrowest source that proves the exact behavior.
4. Record the source link, version, date, or command used.
5. Apply the finding to the repo with the smallest necessary change.
6. In the final response, cite the proof or summarize the exact local evidence.

## Local Evidence

Use local evidence when the behavior can be verified directly:

```bash
# Examples only. Choose commands relevant to the task.
go doc package.Symbol
pnpm why package-name
pnpm exec tool --help
rg -n "functionName|exportName" node_modules package-lock.json pnpm-lock.yaml
```

Do not vendor-copy large external snippets into the repo. Quote only the minimum needed to justify the implementation.

## Output Expectations

- Name the source of truth.
- Include the exact version when version affects behavior.
- Explain how the evidence changes the implementation decision.
- Say when proof could not be found and what assumption remains.
- Do not present guesses as verified facts.
