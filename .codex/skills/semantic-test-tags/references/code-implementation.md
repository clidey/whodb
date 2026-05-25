# Code Implementation Notes

Use this reference when adding semantic tags to an existing frontend codebase.

## Existing Convention Scan

Before editing, quickly search the repo for local conventions:

```text
data-testid
data-qa-
getByTestId
testId
dataTestId
qa=
slotProps
inputProps
rootProps
```

Use the discovered naming and prop-forwarding style unless it is clearly unstable or tied to presentation.

## Placement

Put tags on the element that automation should operate on or assert:

- Buttons: the actual clickable button element, or the component that forwards DOM attributes to it.
- Inputs/selects/switches/uploads: the interactive control or stable wrapper used by the component library.
- Dialogs/drawers: the dialog root plus key action buttons and fields.
- Lists/tables/cards: the repeated row/card root, plus action buttons inside each item.
- Status/error/empty/loading: the visible state container that represents the state.
- Detail pages: the detail root or header that binds to the resource ID.

If a component library does not forward unknown `data-*` props to the desired DOM node, inspect the rendered DOM or component API and use the supported slot/prop such as `inputProps`, `slotProps`, `componentsProps`, `rootProps`, `popupProps`, `dropdownProps`, row/cell render props, option render props, or equivalent local wrapper conventions.

For popup components, decide which semantic surface automation needs:

- Trigger semantics belong on the visible button/input that opens the popup.
- Popup root semantics belong on the menu, dropdown, modal, drawer, popover, or tooltip content when tests must assert the opened state.
- Option/item semantics belong on rendered options or menu items when tests must choose specific business values.

## Shared Components

Do not hard-code feature-specific semantics inside a shared component unless the component always represents the same product concept.

Prefer:

```tsx
<ResourceCard
  resource={devbox}
  data-testid="devbox.list.item"
  data-qa-module="devbox"
  data-qa-object="instance"
  data-qa-resource-type="devbox"
  data-qa-resource-id={devbox.id}
/>
```

or a typed semantic prop:

```tsx
<ResourceCard
  resource={devbox}
  qa={{
    testId: "devbox.list.item",
    module: "devbox",
    object: "instance",
    resourceType: "devbox",
    resourceId: devbox.id,
  }}
/>
```

Avoid:

```tsx
function ResourceCard() {
  return <div data-testid="devbox.list.item" />;
}
```

unless `ResourceCard` is only used for DevBox list items.

## Conditional Attributes

Use `undefined` to omit attributes that do not apply in React:

```tsx
data-qa-disabled-reason={
  isLoading
    ? "loading"
    : !canCreate
      ? "permission_denied"
      : quotaExceeded
        ? "quota_exceeded"
        : undefined
}
```

Keep enums stable and machine-readable. Do not use user-visible text:

```tsx
data-qa-disabled-reason="permission_denied"
```

not:

```tsx
data-qa-disabled-reason="You do not have permission"
```

## Framework Adaptation

The semantic model is framework-agnostic; use the syntax and prop-forwarding conventions of the local stack.

React/TypeScript:

```tsx
<button data-testid="devbox.create.submit-button" data-qa-action="create" />
```

Vue:

```vue
<button
  data-testid="devbox.create.submit-button"
  data-qa-action="create"
  :data-qa-state="isSubmitting ? 'loading' : 'ready'"
/>
```

Svelte:

```svelte
<button
  data-testid="devbox.create.submit-button"
  data-qa-action="create"
  data-qa-state={isSubmitting ? 'loading' : 'ready'}
/>
```

Mobile:

- Prefer the platform's stable accessibility identifier or test ID mechanism.
- Keep the same module/object/action/state/resource naming model when naming identifiers.

## Sensitive Data

Resource binding should support evidence correlation without exposing secrets.

Allowed examples:

```tsx
data-qa-resource-id={devbox.id}
data-qa-workspace-id={workspaceId}
data-qa-namespace={namespace}
```

Avoid placing tokens, passwords, access keys, private URLs, or raw credentials in DOM attributes.

## Verification

After implementation, verify at least one of:

- Typecheck or lint passes.
- Component/unit tests pass.
- Existing Playwright tests can still locate the flow.
- Browser DOM inspection confirms the intended elements carry the expected tags.

For repeated rows or resource-bound cards, verify at least one rendered instance includes the expected resource type, resource ID, and state when those values are available.
