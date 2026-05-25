# QA Attribute Reference

## Stable Locator

Required for automation-facing elements:

```html
data-testid="devbox.create.submit-button"
```

Purpose:

```text
Provide stable locators for Playwright, automation scripts, and GUI agents.
```

Naming:

```text
<module>.<object>.<purpose>
<domain>.<module>.<object>.<purpose>
```

Examples:

```text
devbox.list.create-button
devbox.create.name-input
devbox.create.runtime-select
devbox.create.cpu-input
devbox.create.memory-input
devbox.create.submit-button
devbox.detail.status-badge
devbox.detail.terminal-button
workspace.member.role-select
object-storage.bucket.create-submit
```

## Business Semantics

Recommended attributes:

```html
data-qa-module="devbox"
data-qa-object="instance"
data-qa-action="create"
data-qa-field="runtime"
```

Common fields:

| Attribute | Meaning | Example |
| --- | --- | --- |
| `data-qa-module` | Product module | `devbox` |
| `data-qa-object` | Business object | `instance` |
| `data-qa-action` | User action | `create` |
| `data-qa-field` | Form field | `runtime` |
| `data-qa-risk` | Risk type | `resource_mutation` |

## State Semantics

Recommended attributes:

```html
data-qa-state="running"
data-qa-loading="false"
data-qa-disabled-reason="quota_exceeded"
```

Common fields:

| Attribute | Meaning | Example |
| --- | --- | --- |
| `data-qa-state` | Business state | `creating` / `running` / `failed` |
| `data-qa-loading` | Loading flag | `true` / `false` |
| `data-qa-disabled-reason` | Disabled reason | `quota_exceeded` |
| `data-qa-error-code` | Error code | `permission_denied` |

Use stable enums for disabled reasons, not visible text.

Suggested disabled reason enums:

```text
quota_exceeded
permission_denied
invalid_config
loading
not_ready
dependency_unavailable
unsupported_version
unknown
```

## Resource Binding

Recommended attributes:

```html
data-qa-resource-type="devbox"
data-qa-resource-id="devbox-xxx"
data-qa-workspace-id="ws-xxx"
```

Common fields:

| Attribute | Meaning |
| --- | --- |
| `data-qa-resource-type` | Resource type |
| `data-qa-resource-id` | Resource ID |
| `data-qa-workspace-id` | Workspace ID |
| `data-qa-tenant-id` | Tenant ID |
| `data-qa-namespace` | Namespace |

Use resource binding to connect UI evidence to API, database, Kubernetes, runtime, or log evidence.
