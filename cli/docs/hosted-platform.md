# Hosted WhoDB Platform CLI

The hosted platform commands let an existing WhoDB account use the CLI against
a hosted or self-hosted WhoDB platform. The default host is
`https://app.whodb.com`; pass `--host` for local, staging or self-hosted
environments.

```bash
whodb login
whodb login --host http://localhost:8080
whodb status
whodb manifest
```

## Authentication

`login` opens the browser and uses the platform's sign-in flow. The refresh
token goes in the OS keyring; only non-secret account metadata is written to
the CLI config.

The CLI keeps one login per host. Signing in to another host leaves existing
sessions alone. Signing in again to the same host asks before revoking and
replacing that host's session (`--yes` skips the prompt).

```bash
whodb whoami
whodb logout
whodb logout --host http://localhost:8080 --local   # remove local credentials only
```

## Workspace selection

Hosted commands need an organization and project. If the account has one
organization, and that organization has one project, the CLI selects them and
says so. Otherwise save a default once:

```bash
whodb orgs list
whodb projects list --org <org-id-or-slug>
whodb use --org <org-id-or-slug> --project <project-id-or-slug>
```

Commands also accept `--org` and `--project`. When neither flag is given, the
saved default is used directly, without listing organizations first; a default
that no longer exists surfaces as the error from the command's own request.

For independent terminals, set a process-local scope instead. Values resolve as
IDs, slugs or names and never overwrite the default saved by `whodb use`:

```bash
export WHODB_PLATFORM_SESSION_HOST=https://your-whodb.example   # optional
export WHODB_PLATFORM_SESSION_ORG=acme
export WHODB_PLATFORM_SESSION_PROJECT=analysis
whodb ontologies list
```

`WHODB_PLATFORM_SESSION_ORG` and `WHODB_PLATFORM_SESSION_PROJECT` must be set
together.

## Platform manifest

The host publishes an authenticated manifest of the operations and fields it
supports. It only gates compatibility and command UX; the backend still
authorizes every request.

The CLI caches the manifest with the platform version and checks the version at
most every ten minutes. When the version changes, or the server rejects an
operation it does not know, the CLI refreshes the manifest and retries once.
Operations an older host does not advertise fail with "this WhoDB host does not
support X yet" until the host is upgraded.

```bash
whodb manifest --refresh --format json
```

## Sources

```bash
whodb sources types
whodb sources fields Postgres

printf "%s\n" "$PGPASSWORD" | whodb sources create Postgres \
  --name local-postgres --hostname localhost --port 5432 \
  --username postgres --database postgres --password-stdin

whodb sources list
whodb sources get local-postgres
whodb sources config local-postgres        # secret-looking values are redacted
whodb sources update local-postgres --database analytics
whodb sources test local-postgres
whodb sources objects local-postgres
whodb sources columns local-postgres --ref table:public.users
whodb sources rows local-postgres --ref table:public.users --limit 25
whodb sources delete local-postgres        # prompts; --yes skips
```

`sources test` also accepts `--type` plus connection flags to test a source
before saving it.

## OAuth Secrets

List the reviewed providers and get the state, expiry and last renewal of a
managed Secret by ID:

```sh
whodb secrets oauth-providers --format json
whodb secrets oauth-status SECRET_ID --format json
```

Set `WHODB_OAUTH_CLIENT_SECRET` in the local CLI or stdio MCP process environment
using your local secret manager. The client secret is read when the approved
write executes. Keep it out of command arguments, tool payloads and chat.

```sh
whodb resources create oauth_secret --host app.whodb.com --org clidey --project social-media \
  --payload-json '{"name":"SOCIAL_LINKEDIN_ACCESS_TOKEN","provider":"linkedin","clientId":"YOUR_CLIENT_ID","scopes":["w_organization_social"]}'
```

In local stdio MCP, use `whodb_platform_create` with resource `oauth_secret` and
the same metadata, then review and confirm the write in the default confirmation
mode. The read tools are `whodb_platform_oauth_secret_providers` and
`whodb_platform_oauth_secret_status`. Remote MCP supports the reads. Use the
hosted Secrets page for creation when using remote MCP.

Start provider consent from a terminal:

```sh
whodb secrets authorize SOCIAL_LINKEDIN_ACCESS_TOKEN --host app.whodb.com --org clidey --project social-media
```

After confirmation, WhoDB opens your default browser. Sign in as the same WhoDB
user if prompted, then approve access at the provider. Use `--no-browser` or
`--format json` to copy the one-time link for use on another machine. The link
expires after 10 minutes.

In MCP, call `whodb_platform_action` with `resource: "oauth_secret"`, `action:
"authorize"`, and the secret `id`. After the user approves the preview, pass its
token to `whodb_platform_confirm` and open the returned `authorization_url`.
Treat the link as private. Use `oauth-status` to check that consent completed.

On a server without browser handoffs, open the secret in **Secrets** and select
**Authorize / reconnect**.

## Automation output

Every hosted command accepts `--format json`. Lifecycle and mutating commands
return an envelope; read commands emit the resource directly. `--quiet`
suppresses informational text in human-readable output.

```json
{"command": "sources.create", "success": true, "data": {}}
```

Service-account jobs are described in [platform operations](platform-operations.md).

## MCP

`whodb mcp serve` exposes the hosted platform tools, using the saved logins.
Policies, tool selection and per-call workspace targeting are covered in the
[CLI README](../README.md#platform-access-policy-and-tool-selection). In short:

- A call's explicit `workspace` wins. Otherwise the server uses the session
  default set by `whodb_platform_session_use`, then the process scope
  (`--platform-host`, `--platform-org`, `--platform-project` or the
  `WHODB_PLATFORM_SESSION_*` variables), then the saved CLI default.
- The session default lives in memory for that MCP session only and is never
  written to disk; `whodb_platform_session_clear` drops it. Don't run `whodb use`
  to switch hosts for one chat; it changes the default for every session.
- Per process, the server caches each host's manifest for five minutes and each
  account's organization and project lists for one minute. A lookup miss
  refetches once, and listing tools always fetch live data.
- Hosted writes return a preview bound to its host, account, organization and
  project, and run only after `whodb_platform_confirm`; `whodb_platform_pending`
  recovers open confirmations. See
  [upload review and confirmation limitations](../README.md#upload-review-and-confirmation-limitations).
- Read `whodb://platform/schema` for the enabled tools and payload shapes, and
  `whodb://platform/tool-guide` for read/write behavior and row limits.
- Requests carry an `X-WhoDB-Client` header (`whodb-cli/<version>` from commands,
  `whodb-cli-mcp/<version>` from MCP) so the host can tell CLI traffic apart.

## Security model

- The CLI acts as the signed-in user; source permissions are enforced by the
  hosted backend.
- Refresh tokens are stored in the OS keyring. Source secrets are never written
  to the CLI config.
- `sources config` and `whodb_platform_source_config` redact secret-looking
  values in human and JSON output.

## Edge execution and operational visibility

Function reads include execution mode, live behavior, assigned runner and current
run status when the host advertises those fields. Set `--execution-mode ASYNC`
(or `LIVE` with `--live-behavior SERVICE`) when creating a Python edge function;
assign it with `functions update <id> --edge-runner-id <runner-id>`. Clearing the
assignment uses `--edge-runner-id ""`. Promote the reviewed function before running it.
The device owner must separately enroll/pair the device and grant execution access.
Device credentials, consent and runner downloads remain in the browser/device flow.

```bash
whodb edge runners --format json
whodb edge start <function-id> --runner-id <runner-id> --input '{"task":"refresh"}'
whodb edge runs <function-id>
whodb edge run <run-id> --format json
whodb edge schedule --function-id <function-id> --runner-id <runner-id> --interval-seconds 60
whodb edge schedules <function-id>
whodb edge stop-schedule <schedule-id>
whodb apps access <app-id> --search alice --limit 20 --offset 0
whodb transforms history --per-transform 5 --limit 20
whodb agent-sessions list
whodb agent-sessions get <session-id>
whodb agent-sessions cancel <session-id>
whodb analysis get <artifact-id>
whodb analysis rows <dataset-artifact-id> --limit 20
whodb analysis rows <dataset-artifact-id> --limit 20 --cursor '<nextCursor>'
whodb usage current --org acme --format json
whodb usage entitlements --org acme
whodb usage feature <feature-key> --org acme
```

Writes retain the existing confirmation prompt (`--yes` explicitly approves the
CLI action). Edge intervals range from 10 to 3600 seconds and are pinned to a
promoted function version. Inspect the same run after a timeout rather than
starting a duplicate. App access requires sharing permission and returns effective
users and direct team grants; follow offsets until the returned total is reached.
AI sessions belong to the signed-in user; inspection does not resume AI generation.
Analysis artifacts preserve warnings and provenance. Follow `nextCursor` until it
is absent and inspect execution metadata before treating results as complete.
Usage and entitlements require only an organization and never modify billing.

MCP exposes the corresponding `whodb_platform_edge_*`, `app_access`,
`project_transform_runs`, `agent_sessions`, `agent_session`, `analysis_artifact`,
`analysis_artifact_page`, `usage`, `entitlements` and `feature_access` tools.
MCP writes reuse the confirmed generic tools: `action` with resource
`edge_function`/action `run`, `create` with resource `edge_schedule`, and `action`
with resource `edge_schedule`/action `stop` or `agent_session`/action `cancel`.
Read `whodb://platform/schema` for payloads. Preserve `idempotency_key` across
MCP retries. All tools accept per-call workspace targeting; task profiles filter
discovery without granting permissions.
