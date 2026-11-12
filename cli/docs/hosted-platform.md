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
