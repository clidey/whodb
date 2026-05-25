---
name: semantic-test-tags
description: Use this skill for frontend code tasks whose goal is to add, complete, audit, or review machine-readable UI semantics for automation or GUI agents. Trigger on Chinese or English requests such as 补充语义标签, 给前端加 data-testid, 补 data-qa-state/resource-id/error-code, 让 Playwright/自动化测试/GUI Agent 读懂页面, 按 PRD/需求改服务里的语义标签, review 语义标签覆盖, or check unstable test IDs. It covers reading requirements plus routes/pages/components/hooks/API/state/permissions, then implementing or reviewing stable data-testid and data-qa contracts. Do not use for writing tests only, aria/accessibility-only fixes, CSS/style-only changes, backend API contracts, or ordinary component refactors that do not change semantic tag coverage.
metadata:
  source: "Feishu wiki AdT2whQogiGVySk3m3ccNzRFnrb, revision 5"
---

# Semantic Test Tags

## Purpose

Use this skill when the task is to make frontend code readable to automated tests and GUI agents by adding stable semantic tags based on real product requirements and actual implementation structure.

The agent should not invent tags from a screenshot or isolated UI text alone. It must read the requirement, inspect the relevant frontend code, identify the business objects and states represented by the UI, then add tags that expose stable product semantics.

When the user asks to add, supplement, or land semantic tags in a service, treat it as an implementation task by default: modify the relevant code, produce the semantic tag contract document, and verify the result. Stop at analysis only when the user explicitly asks for review/planning, or when the requirement and code do not provide enough signal to infer safe business semantics.

## When To Use

Use this skill for:

- Reading a requirement or PRD and then supplementing frontend code with semantic tags.
- Adding `data-testid` and `data-qa-*` attributes to existing pages, components, forms, lists, dialogs, status panels, and error states.
- Making a service easier for Playwright, black-box tests, GUI agents, or evidence collectors to inspect.
- Inferring UI semantics from routes, component names, data models, API calls, state machines, stores, hooks, query keys, and permission logic.
- Connecting UI evidence to API, database, Kubernetes, runtime, logs, workspace, tenant, namespace, or resource IDs.
- Creating or updating `semantic-test-contract.md` after tags are added.
- Reviewing whether a frontend change has enough semantic surface for automation.

For API services, data pipelines, CLIs, and backend jobs, prefer their native contracts instead: OpenAPI or API contracts, schema or lineage contracts, command/output contracts, job/event/metric contracts.

## Implementation Workflow

Follow this workflow when the user asks to supplement tags in a service:

1. Understand the requirement.
   - Identify the user journey, target module, business object, allowed actions, important states, error paths, permissions, and quality evidence needs.
   - If the requirement is vague, infer from code and existing product patterns before asking follow-up questions.

2. Locate the frontend surface.
   - Search routes, pages, component directories, menu config, feature flags, i18n keys, API clients, stores, hooks, and tests.
   - Prefer existing local conventions for test IDs or QA attributes if the repo already has them.
   - Identify repeated components carefully: add generic semantics in shared components only when the semantics are truly shared; otherwise pass semantic props from the feature layer.

3. Inventory existing project conventions.
   - Search for existing patterns before designing new ones: `data-testid`, `data-qa-`, `getByTestId`, `testId`, `dataTestId`, `qa`, `slotProps`, `inputProps`, `rootProps`, and local test helper utilities.
   - Identify naming style, component prop-forwarding patterns, test locator preferences, and existing docs or PR checklist expectations.
   - Reuse the local convention unless it is unstable, presentation-bound, or conflicts with the semantic contract.

4. Prepare a tag plan before editing.
   - Build a concise element-to-tag plan that maps requirement items to code locations and semantic attributes.
   - Use this shape, adapting column names as needed:

```text
Element | Code location | Required | data-testid | data-qa-* | Evidence source
```

   - Include only elements that participate in operation, assertion, evidence, risk, state, error handling, or resource binding.
   - Use the plan to avoid missed requirement elements and to make shared-component changes explicit before patching.

5. Map UI elements to product semantics.
   - For each relevant element, decide its module, object, purpose, action, field, state, risk, and resource binding.
   - Prioritize elements that participate in operation, assertion, evidence, or risk.
   - Avoid adding tags to pure layout, decoration, or text that automation will never operate on or assert.

6. Implement tags in code.
   - Add stable `data-testid` to automation-facing elements.
   - Add `data-qa-*` attributes when business semantics, state, disabled reason, error code, risk, or resource binding matters.
   - Preserve existing `aria-*`, roles, labels, keyboard behavior, and component props.
   - Do not break styling, component composition, forwarded props, or TypeScript types.
   - Read [references/code-implementation.md](references/code-implementation.md) when working with shared components, component libraries, conditional attributes, or resource-bound rows.

7. Produce the semantic tag contract.
   - Search for existing contract or testing docs before creating new files: `semantic-test-contract.md`, `docs/testing`, `tests/README`, `qa`, `e2e`, and local PR templates are common locations.
   - Always produce a semantic tag document for the touched module or workflow. Prefer updating an existing `semantic-test-contract.md`; otherwise create a focused module/workflow contract near the local docs or test documentation convention.
   - If the repository has no obvious docs location, create a small `semantic-test-contract.md` beside the relevant feature docs, test docs, or module directory rather than skipping the document.
   - Record tag IDs, element types, business semantics, states, disabled reasons, error codes, resource bindings, evidence source, and change rules.
   - Read [references/contract-template.md](references/contract-template.md) when drafting the contract.

8. Verify.
   - Run the narrowest relevant typecheck, lint, unit test, component test, or Playwright test available.
   - For browser-visible changes, inspect the rendered DOM when practical and confirm tags appear on the intended elements.
   - If tests are not available or cannot run, report that explicitly with the best static checks performed.

## Done Criteria

The work is complete only when:

- Requirement-mentioned action, field, state, error, disabled, and resource elements are accounted for in the tag plan.
- Core user actions and automation-facing fields have stable `data-testid`.
- Key states expose `data-qa-state` or `data-qa-loading` when automation must assert them.
- Disabled states and errors use stable machine-readable enums such as `data-qa-disabled-reason` and `data-qa-error-code`.
- Resource rows, cards, or detail roots expose safe resource binding such as `data-qa-resource-type`, `data-qa-resource-id`, workspace, tenant, or namespace when available.
- Shared components receive feature-specific semantics from the feature layer unless the shared component truly represents one business concept.
- A semantic tag contract document has been created or updated.
- Verification has run, or the final response explains why it could not run and what static checks were completed.

## Tagging Decision

Add stable semantics to elements involved in operation, assertion, evidence, or risk:

- Core user action buttons.
- Form inputs, selectors, switches, upload controls.
- Create, delete, confirm, submit, restart, transfer, authorization, and other high-risk actions.
- Key business object cards, list rows, and detail roots.
- Key status display areas.
- Loading, empty, success, failed, and disabled states.
- Error, warning, permission, and quota messages.
- Result regions that automation must assert.
- Elements that bind to backend objects, platform objects, workspace IDs, tenants, namespaces, or business IDs.

Do not tag pure layout containers, decorative icons, style-only wrappers, meaningless `div` or `span` nodes, or ordinary explanatory copy that automation will not operate on or assert.

## Attribute Layers

Use these layers together when the element is important enough:

1. Stable locator: `data-testid`
2. Business semantics: `data-qa-module`, `data-qa-object`, `data-qa-action`, `data-qa-field`, `data-qa-risk`
3. State semantics: `data-qa-state`, `data-qa-loading`, `data-qa-disabled-reason`, `data-qa-error-code`
4. Resource binding: `data-qa-resource-type`, `data-qa-resource-id`, `data-qa-workspace-id`, `data-qa-tenant-id`, `data-qa-namespace`

Read [references/qa-attributes.md](references/qa-attributes.md) when choosing exact fields, naming examples, or stable enum values.

## Code Reading Heuristics

When choosing tag names and `data-qa-*` values, derive semantics from stable product concepts, not incidental UI text.

Useful signals:

- Route path and page file name for module and page context.
- Component name for object or panel semantics.
- API endpoint, query key, store slice, model type, or resource DTO for `data-qa-object` and resource binding.
- Form schema, validation schema, or field name for `data-qa-field`.
- Button handler names and mutation hooks for `data-qa-action`.
- Permission checks and disabled logic for `data-qa-disabled-reason`.
- Status enum, phase enum, condition, or backend state field for `data-qa-state`.
- Error code, toast payload, API error mapping, or alert branch for `data-qa-error-code`.

If a component displays a concrete backend or platform resource, bind the rendered row/card/detail root to the relevant stable ID when available. Use resource IDs that automation can correlate with API or runtime evidence; do not expose secrets or sensitive tokens.

## Naming Rules

Always provide a stable `data-testid` for automation-facing elements.

Format:

```text
<module>.<object>.<purpose>
```

For large systems, include a domain:

```text
<domain>.<module>.<object>.<purpose>
```

Good examples:

```text
devbox.list.create-button
devbox.create.name-input
devbox.create.runtime-select
devbox.create.submit-button
devbox.detail.status-badge
workspace.member.role-select
object-storage.bucket.delete-confirm-button
```

For repeated rows, cards, and options, keep `data-testid` stable across instances and use resource binding or row scope for uniqueness:

```text
data-testid="devbox.list.item"
data-qa-resource-id="devbox-xxx"
```

Do not append dynamic resource IDs or array indexes to `data-testid` by default. A stable shared test ID plus `data-qa-resource-id`, accessible row text, or scoped locators gives automation a durable contract without turning runtime data into locator names.

Avoid names tied to presentation, DOM shape, array position, or implementation details:

```text
button-1
card-3
primary-button
ant-btn-submit
div-list-item
random-8f3a
```

## Stability Contract

Treat semantic tags as a test contract, not ordinary implementation detail.

- Do not change `data-testid` because visible text changed.
- Do not change `data-testid` because CSS, component hierarchy, or layout changed.
- Do not use random IDs, array indexes, style names, or component library class names as test semantics.
- Do not reuse the same `data-testid` for different business elements.
- When business semantics change, update the semantic contract document.
- When deleting or renaming tags used by automation, document impact and migration in the PR.

## Accessibility

Preserve real accessibility semantics first, then add test semantics. `aria-*` is for users and assistive technology; `data-testid` is for stable locating; `data-qa-*` is for business meaning, state, risk, and evidence binding.

Do not pollute accessibility labels for test-only needs, and do not rely on accessibility semantics as the complete testing contract when business state, risk, or resource binding must be asserted.

## React Patterns

Prefer passing semantics from the feature layer into reusable components when the same shared component can represent different business objects. These examples use React/TypeScript because that is common in service frontends; adapt the same semantics to Vue, Svelte, native templates, or mobile accessibility identifiers using the local framework conventions.

Submit button:

```tsx
<Button
  aria-label="创建 DevBox"
  data-testid="devbox.create.submit-button"
  data-qa-module="devbox"
  data-qa-object="instance"
  data-qa-action="create"
  data-qa-state={isSubmitting ? "loading" : "ready"}
  data-qa-disabled-reason={
    quotaExceeded
      ? "quota_exceeded"
      : !canCreate
        ? "permission_denied"
        : undefined
  }
  disabled={isSubmitting || quotaExceeded || !canCreate}
  onClick={handleCreate}
>
  创建
</Button>
```

Business object list item:

```tsx
<div
  data-testid="devbox.list.item"
  data-qa-module="devbox"
  data-qa-object="instance"
  data-qa-resource-type="devbox"
  data-qa-resource-id={devbox.id}
  data-qa-workspace-id={workspaceId}
  data-qa-state={devbox.status}
>
  ...
</div>
```

Error message:

```tsx
{quotaExceeded && (
  <Alert
    data-testid="devbox.error.quota-exceeded"
    data-qa-module="devbox"
    data-qa-object="quota"
    data-qa-state="error"
    data-qa-error-code="quota_exceeded"
  >
    当前工作空间资源不足
  </Alert>
)}
```

## Contract Document

For each UI module or workflow touched by semantic tag work, maintain a semantic tag contract document, preferably named `semantic-test-contract.md`, that lists page entries, semantic tags, state enums, disabled reasons, error codes, resource bindings, evidence sources, and change rules.

Read [references/contract-template.md](references/contract-template.md) when creating or updating the module contract.

## PR Review Checklist

When reviewing a frontend PR that touches user-operable pages, verify:

- Core action buttons have stable `data-testid`.
- Form fields have stable `data-testid`.
- Key state areas have assertable tags.
- Error and warning messages have assertable tags.
- Disabled states expose stable reasons such as `data-qa-disabled-reason`.
- Business object rows or cards expose resource IDs such as `data-qa-resource-id`.
- Tests do not rely on CSS class, DOM hierarchy, or visible button text as the only locator.
- Added, deleted, or renamed semantic tags are reflected in `semantic-test-contract.md`.

## Final Response Expectations

When this skill is used to change code, summarize:

- Which requirement or user journey was tagged.
- Which files were changed.
- The main `data-testid` or `data-qa-*` surfaces added.
- Where the semantic tag contract document was created or updated.
- Which elements were intentionally skipped as layout, decoration, or low-value semantic targets.
- Which verification commands or browser checks were run, or why they were not run.
