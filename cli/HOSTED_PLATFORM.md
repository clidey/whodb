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

If the account has exactly one organization, the CLI selects it automatically.
If that selected organization has exactly one project, the CLI selects that
project automatically and prints what it selected.

If a host is no longer reachable and you only need to remove local credentials:

```bash
whodb-cli logout --host http://localhost:8080 --local
```

## Workspace Selection

Hosted source commands need an organization and project. When there is only one
possible organization or project, the CLI can select it automatically. Otherwise,
select defaults once:

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

## MCP Platform Tools

Hosted platform MCP mode is opt-in:

```bash
whodb-cli mcp serve --platform
```

The platform tools use the existing hosted login and selected workspace:

```bash
whodb-cli login
whodb-cli use --org <org-id-or-slug> --project <project-id-or-slug>
```

For single-workspace accounts, `login` or `status` can select the workspace
automatically and report what was selected.

Platform mode also exposes MCP resources and prompts that agents should read
before choosing tools:

- `whodb://platform/schema` — exact enabled tool list, resource metadata,
  prompt metadata, generic write specs, and payload shapes for the current
  server mode.
- `whodb://platform/workspace` — current host, signed-in user, selected
  organization, selected project, and readiness state.
- `whodb://platform/tool-guide` — tool categories, read/write behavior, row
  limits, field projection guidance, and confirmation behavior.

Platform prompts:

- `whodb_platform_overview`
- `whodb_platform_read_workflow`
- `whodb_platform_write_safety`
- `whodb_platform_source_workflow`

Hosted MCP tool groups:

- Workspace and readiness: `whodb_platform_status`,
  `whodb_platform_orgs`, `whodb_platform_projects`
- Sources: `whodb_platform_sources`, `whodb_platform_source_types`,
  `whodb_platform_source_fields`, `whodb_platform_source_objects`,
  `whodb_platform_source_columns`, `whodb_platform_source_rows`,
  `whodb_platform_source_constraints`, `whodb_platform_source_content`,
  `whodb_platform_source_config`, `whodb_platform_source_test`,
  `whodb_platform_source_create`, `whodb_platform_source_update`,
  `whodb_platform_source_delete`
- Project resources: `whodb_platform_secrets`,
  `whodb_platform_ai_providers`, `whodb_platform_ai_provider_models`,
  `whodb_platform_datasets`, `whodb_platform_dataset`,
  `whodb_platform_dataset_rows`, `whodb_platform_ontologies`,
  `whodb_platform_ontology`, `whodb_platform_ontology_fast_lookups`,
  `whodb_platform_ontology_fast_lookup_suggestions`,
  `whodb_platform_ontology_rows`, `whodb_platform_ontology_follow_link`
- Lineage and transforms: `whodb_platform_project_lineage`,
  `whodb_platform_lineage`, `whodb_platform_lineage_neighbors`,
  `whodb_platform_transforms`, `whodb_platform_transform_runs`
- Functions and files: `whodb_platform_functions`,
  `whodb_platform_function`, `whodb_platform_files`,
  `whodb_platform_file_preview`, `whodb_platform_file_search`,
  `whodb_platform_tabular_files`, `whodb_platform_storage_usage`
- Generic writes and confirmations: `whodb_platform_create`,
  `whodb_platform_update`, `whodb_platform_delete`,
  `whodb_platform_action`, `whodb_platform_pending`,
  `whodb_platform_confirm`

Read `whodb://platform/schema` at runtime for the authoritative list. Some
write and confirmation tools are hidden in read-only or safe modes.

`whodb_platform_source_config` returns redacted configuration only. Secret-looking
values such as passwords, tokens, client secrets, and private keys are masked.

Hosted create, update, delete, and action tools do not execute immediately in
the default mode. They return a confirmation token, and the write runs only
after approval through `whodb_platform_confirm`. Use `whodb_platform_pending`
to recover active confirmation tokens.

Generic write tools are capability-backed. Before using
`whodb_platform_create`, `whodb_platform_update`, `whodb_platform_delete`, or
`whodb_platform_action`, agents should read `whodb://platform/schema` and use
the `write_specs` and `payload_shapes` entries instead of guessing GraphQL
mutation names or payload fields.

For hosted source creation, agents should call `whodb_platform_source_types` and
`whodb_platform_source_fields` first instead of guessing source type ids or
connection field names.

For read tools that accept a `fields` parameter, agents should request only the
fields needed for the current answer, then call the tool again with more fields
only if needed. Avoid broad details such as source content, file previews,
function files, row previews, and large lineage graphs unless the user asks for
them or they are required for the task.

If no workspace is selected yet, agents should call `whodb_platform_orgs` and
`whodb_platform_projects`, then ask the user to run:

```bash
whodb-cli use --org <org-id-or-slug> --project <project-id-or-slug>
```

When `--platform` is set, the MCP server exposes only hosted platform tools.
Local database MCP tools such as `whodb_query` and `whodb_connections` are not
registered.

Example hosted platform MCP config:

```json
{
  "mcpServers": {
    "whodb-platform": {
      "command": "whodb-cli",
      "args": ["mcp", "serve", "--platform"]
    }
  }
}
```

Local smoke test with the MCP inspector:

```bash
whodb-cli login --host http://localhost:8080
whodb-cli use --host http://localhost:8080 --org <org-id-or-slug> --project <project-id-or-slug>
npx @modelcontextprotocol/inspector whodb-cli mcp serve --platform
```

In the inspector, call:

1. `whodb_platform_status`
2. Read `whodb://platform/workspace`
3. Read `whodb://platform/schema`
4. Read `whodb://platform/tool-guide`
5. `whodb_platform_orgs`
6. `whodb_platform_projects`
7. `whodb_platform_sources`
8. `whodb_platform_source_types`
9. `whodb_platform_source_fields`
10. `whodb_platform_datasets` with narrow `fields`
11. `whodb_platform_files` with narrow `fields`
12. `whodb_platform_create`, then `whodb_platform_pending`, then `whodb_platform_confirm`
13. `whodb_platform_update`, then `whodb_platform_pending`, then `whodb_platform_confirm`
14. `whodb_platform_delete`, then `whodb_platform_pending`, then `whodb_platform_confirm`

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
