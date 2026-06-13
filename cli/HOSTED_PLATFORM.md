# Hosted WhoDB Platform CLI

The hosted platform commands let an existing WhoDB account use the public CLI
against a hosted or self-hosted WhoDB platform host.

The default host is `https://app.whodb.com`. Use `--host` for local, staging,
or self-hosted environments.

```bash
whodb-cli login
whodb-cli login --host http://localhost:8080
whodb-cli status
whodb-cli manifest
```

## Authentication

`login` opens the browser and uses the hosted platform auth flow. The CLI stores
the refresh token in the OS keyring and stores only non-secret account metadata
in the CLI config.

Only one hosted login is active at a time. If you run `login` while another
host/account is active, the CLI asks before revoking the old session and
replacing the local entry.

```bash
whodb-cli login --host https://app.whodb.com
whodb-cli whoami
whodb-cli logout
```

If a host is no longer reachable and you only need to remove local credentials:

```bash
whodb-cli logout --host http://localhost:8080 --local
```

## Workspace Selection

Hosted source commands need an organization and project. Select defaults once:

```bash
whodb-cli orgs list
whodb-cli projects list --org <org-id-or-slug>
whodb-cli use --org <org-id-or-slug> --project <project-id-or-slug>
```

Power users can pass `--org` and `--project` directly on `sources` commands.

`status` shows the current login, workspace selection, manifest, and source
management capability state.

```bash
whodb-cli status
whodb-cli status --format json
```

## Platform Manifest

The hosted platform publishes a small authenticated manifest for CLI
compatibility. The manifest tells the CLI which hosted operations and fields are
available. It is not a permission system; the backend still enforces access for
every request.

The CLI caches the manifest in the local config with the platform version. When
the platform version changes, or a GraphQL validation error indicates schema
drift, the CLI refreshes the manifest and retries the failed request once.

```bash
whodb-cli manifest
whodb-cli manifest --refresh --format json
```

## Source Management

Discover available source types and required fields:

```bash
whodb-cli sources types
whodb-cli sources fields Postgres
```

Create a source:

```bash
printf "%s\n" "$PGPASSWORD" | whodb-cli sources create Postgres \
  --name local-postgres \
  --hostname localhost \
  --port 5432 \
  --username postgres \
  --database postgres \
  --password-stdin
```

List, inspect, and update sources:

```bash
whodb-cli sources list
whodb-cli sources get local-postgres
whodb-cli sources config local-postgres
whodb-cli sources update local-postgres --database analytics
```

Test connections:

```bash
whodb-cli sources test local-postgres

printf "%s\n" "$PGPASSWORD" | whodb-cli sources test \
  --type Postgres \
  --hostname localhost \
  --port 5432 \
  --username postgres \
  --database postgres \
  --password-stdin
```

Browse source metadata and preview rows:

```bash
whodb-cli sources objects local-postgres
whodb-cli sources columns local-postgres --ref table:public.users
whodb-cli sources rows local-postgres --ref table:public.users --limit 25
```

Delete is destructive and prompts by default:

```bash
whodb-cli sources delete local-postgres
whodb-cli sources delete local-postgres --yes
```

## Automation Output

All hosted commands accept `--format json`. Lifecycle and mutating commands use
an automation envelope:

```json
{
  "command": "sources.create",
  "success": true,
  "data": {}
}
```

Read commands emit the requested resource directly:

```bash
whodb-cli status --format json
whodb-cli manifest --format json
whodb-cli sources types --format json
whodb-cli sources list --format json
```

Use `--quiet` to suppress informational text in human-readable output.

## Security Model

- The CLI acts as the logged-in user.
- Source permissions are enforced by the hosted backend.
- The manifest only gates compatibility and command UX.
- Refresh tokens are stored in the OS keyring.
- Source secrets are never written to the CLI config.
- `sources config` redacts secret-looking values in human and JSON output.
