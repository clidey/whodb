# Focused MCP and platform operations

## Task catalogs

```sh
whodb mcp serve --tool-profile data
whodb mcp serve --tool-profile functions
whodb mcp serve --tool-profile workspace
```

The default `all` catalog is unchanged. Profiles narrow discovery; permissions,
write confirmation, read-only mode and explicit disabled tools still apply.
`--tools` intersects the profile. Use `all` for advanced operations. Persist a
profile through `whodb setup` or `whodb_mcp_setup` with
`{"tool_profile":"functions"}`. Setup's preview/approval and restart rules apply.
The `whodb://mcp/configuration` resource lists the active and available profiles.

`whodb_platform_runtime_readiness` now reports observed configuration separately
from runtime reachability and verification. Metadata alone leaves runtime states
`unknown`. Supply `function_id` to inspect the latest persisted run; its completion
timestamp is historical evidence, not a probe of current capacity. Unavailable
inventories produce warnings and unknown state rather than a zero-resource claim.
Workspace maps include apps and package installations.

## Unattended CLI jobs

Supply a service-account API key through your secret runner's environment:

```sh
export WHODB_PLATFORM_SESSION_HOST=https://your-whodb.example
export WHODB_PLATFORM_SESSION_ORG="organization-id"
export WHODB_PLATFORM_SESSION_PROJECT="project-id"
# Inject WHODB_PLATFORM_API_KEY from your job's secret store.
whodb sources list --format json --quiet
```

Use canonical workspace IDs. The key is bound to the explicit session host;
a conflicting `--host` is rejected. HTTP redirects are not followed. An empty,
invalid, expired or revoked key never falls back to a personal login or refresh
token. The key is not written to configuration or the keyring. Backend grants
still govern reads and writes. This credential path is for CLI platform commands;
hosted MCP continues to use its existing OAuth flow.

## Security ledger

Organization managers can use:

```sh
whodb security-audit --limit 50 --request-id <request-id>
whodb security-audit --run-id <run-id>
whodb security-audit --outcome DENIED
whodb security-audit --cursor <nextCursor>
whodb security-audit --mode exports
whodb security-audit --mode verify
```

Output is JSON. Optional event filters include project, actor, action, outcome,
and RFC3339 `--since`/`--until`. MCP exposes the same operations through
`whodb_platform_security_audit`. This is the platform security ledger;
the database `audit` command remains a data-quality inspection.

Exports report organization-scoped receipt backlog and oldest pending event.
Exporter attempt timestamps and failure stage describe the responding process;
missing observations are unknown. Verification checks retained chained rows
against the stored tip and reports excluded unchained events. It is not an
independent external integrity anchor. Disabled auditing is reported explicitly.
