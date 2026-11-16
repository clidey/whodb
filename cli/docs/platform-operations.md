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

## Metadata and run inspection

App reads include `offlineCapable`, `recoveryBindings` and source visibility.
`whodb_platform_app_view` accepts `page`; published views include their version.
Package details include teams/grants, lineage, media, reviews and changelog.
Installation reads and write results include source organization/project scope.

`whodb transforms get-run <transform-id> <run-id>` and
`whodb_platform_transform_run` look up an authorized run by id, including runs
outside recent history. Transform wait uses the same lookup. This requires a host
that advertises `TransformRun` in its manifest.

## Script previews

```sh
whodb scripts packages --language python
whodb scripts search --language python --query pandas
whodb scripts package python pandas
whodb scripts run --language python --file preview.py --dependencies pandas --sample-data '{"columns":["value"],"rows":[["42"]]}' --yes
```

These commands inspect the platform's curated catalog and run a script preview in
its configured isolated runtime. The package commands do not install dependencies.
Omitting sample data uses the platform's built-in sample rows; this is a preview,
not an implicit connection to a source. Runtime failures return `success: false`.
Check `success`, `runtime` and `error` before using the output.

MCP exposes `whodb_platform_script_packages`, `whodb_platform_script_package_search`
and `whodb_platform_script_package`. Execute through `whodb_platform_action` with
`resource: "script"`, `action: "execute"` and payload fields `language`, `script`,
`dependencies`, and optional JSON-string `sampleData`. Execution requires project
edit permission and the normal confirmation flow.

## Read-only plan inspection

```sh
whodb usage subscription
whodb usage plans
whodb usage plan-features free
whodb usage preview-plan pro monthly
```

MCP equivalents are `whodb_platform_subscription`, `whodb_platform_plans`,
`whodb_platform_plan_features` and `whodb_platform_subscription_preview`. They work
with organization scope and do not change subscriptions or create checkout sessions.
Unconfigured billing returns an explicit error rather than an empty catalog.

## Automation parity checks

The API root coverage check requires every operation to be implemented or explicitly
excluded. `TestPlatformAutomationSchemaContract` also checks argument signatures,
nested inputs, enums and response fields against the reviewed schema contract.
Schema changes require reviewing client selections, payloads, manifests and
registrations before the host-side contract is refreshed.

The read-field tests additionally require complete projections for app/package
metadata, datasets and transforms. CLI registration tests and real MCP `tools/list`
and `tools/call` tests verify the shared capability catalog.

Successful browser capture has its own integration test because the backend-only
CLI stack has no frontend or browser. Point at a disposable local stack serving the
current frontend and sandbox kernel, with Playwright Chromium installed:

```sh
cd cli
WHODB_PLATFORM_E2E_HOST=http://localhost:18080 \
WHODB_PLATFORM_E2E_KEYCLOAK_URL=http://localhost:14001 \
WHODB_CAPTURE_PLAYWRIGHT_PACKAGE="$PWD/../frontend/package.json" \
go test -tags 'e2e_platform e2e_browser' ./pkg/mcp -run '^TestPlatformAppCapture_RealPublishedApp$' -count=1
```

If your test environment replaces `HOME`, set `PLAYWRIGHT_BROWSERS_PATH` to the
existing browser cache. The test publishes a two-page fixture, checks MCP image
content, desktop interactions/tracing, mobile landscape capture, navigation,
failed-action diagnostics, published-vs-draft isolation and page metadata. It
rejects non-loopback hosts and cleans up its app.
