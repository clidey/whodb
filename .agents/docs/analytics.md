# Analytics Contract (PostHog)

All senders (browser frontend, Go backend, CLI, desktop) share one PostHog
project. This contract keeps their events consistent and queryable. Follow it
whenever adding or changing analytics events or properties.

## Required properties on every event

Stamped automatically at the two choke points — do not set these manually at
call sites, and do not bypass the choke points:

| Property | Values | Stamped by |
|---|---|---|
| `source` | `web`, `backend`, `cli`, `desktop` | Go: `buildProperties()` from `Config.Source`; TS: `sanitizeAnalyticsProperties()` |
| `build_edition` | `ce`, `ee` | Go: `Config.Edition` (required — never leave unset); TS: runtime context |
| `build_environment` | e.g. `production`, `cli` | Go: `Config.Environment`; TS: runtime context |
| `app_version` | release version, or `dev`/`development` for unstamped builds | Go config; TS: compiled `__APP_VERSION__` |
| `platform` | `browser`, `wails` | Web events only (TS runtime context) |
| `$host` | request/browser host | Go: from request metadata; TS: auto-captured by posthog-js |

Choke points:
- Go: `buildProperties()` in `core/src/analytics/posthog.go` — every capture
  from every Go sender flows through it.
- TS: `sanitizeAnalyticsProperties()` in
  `frontend/src/config/analytics-sanitize.ts`.

Frontend event properties cannot override the runtime source, edition,
environment, platform, caller, version or deployment. Frontend events carry
`client_type: browser` or `desktop`; local CLI/MCP events use `cli` or `mcp`.
These are calling surfaces, distinct from the emitting `source`. Server calls
without a known caller remain unclassified rather than guessing from an ID.

Naming: use the `build_`-prefixed names above. Do not introduce `environment`,
`edition`, or a second `platform` meaning. Desktop OS goes in `os` (GOOS), not
`platform`.

## Event ownership: one owner per concept

- **Backend owns "it happened" events** — resource created/updated/deleted,
  runs, auth outcomes, billing. Server-side capture cannot be ad-blocked and is
  the source of truth for counts and funnels.
- **Frontend owns UX-only events** — screen views, form
  opened/submitted/abandoned, option toggles, client-side failures
  (`ui.*` and `*_opened`/`*_abandoned`-style funnel events).

Never emit the same concept from both sides: that double-counts every action.
If the backend already records it, the frontend must not.

## Source query outcomes

The source resolvers emit `source.rows_queried` and `source.query_executed`
once per resolver attempt. Capture requires server analytics to be enabled and
a non-empty, non-disabled analytics identity, matching login capture controls.
Requests rejected before the resolver runs are outside these metrics.

Properties include the resolved connector, success/failure, coarse error code,
latency, `duration_bucket` and `slow`. Result metadata uses `row_count_bucket`,
`column_count_bucket`, `total_row_count_bucket` and `empty_result`; returned
rows stay separate from the connector's reported total. Unknown negative totals
are omitted. Row browsing also records filter/sort presence and pagination
buckets/presence. Raw queries include only the fixed `statement_type`,
`multi_statement` and a query-length bucket. Never include SQL or source object
paths, column names or values.

## CLI and local MCP outcomes

CLI events include `execution_mode: command`, `tui` or `mcp`, in addition to
`client_type`. Terminal UI editor, history and explicitly confirmed chat SQL
queries use the same query events as the query command, with a fixed
`origin_type` for their entry point. This does not cover every terminal UI action
or queries executed internally by AI tools.

Query successes use `cli.query.execute`; failures use `cli.query.error` with
`success: false`. Older failure events may omit `success`; dashboards must still
count them and combine the two event names before calculating a query success
rate. Query-command connections emit attempt and terminal success/error events;
this is not complete connection coverage for every command or terminal UI flow.

Timing and result counts use the shared `analytics.DurationBucket` and
`analytics.CountBucket` helpers. Unknown negative sizes must not become empty
results. Canceled work has `error_code: canceled`, distinct from timeouts and
execution errors. Client-observed cancellation is not proof that the database
stopped executing.

A local MCP response awaiting write approval has
`outcome: confirmation_required`. Its successful tool response is not an
executed write or a completed user task.

## Identity

- Backend distinct IDs resolve in order: registered resolver
  (`analytics.SetDistinctIDResolver`, set at boot by edition entry points) →
  `X-WhoDB-Analytics-Id` header from the frontend → `anonymous`.
- Never invent synthetic distinct IDs (random, per-process, hashed timestamps).
  Unattributable events belong to `anonymous`.
- CLI command and MCP analytics use the persisted install UUID. These measure
  installations, not people. Exclude the shared `anonymous` ID from identity
  retention and report unattributed event volume separately.

## Privacy

- Never send raw error messages. Map errors through `analytics.ErrorCode`
  (Go) or the frontend error-code helpers to a fixed taxonomy.
- The Go capture choke point drops raw `path`, `referer` and `user_agent`
  properties and does not attach those values from request metadata. Request
  metadata remains available internally for audit and SDK classification.
  Application-host context (`$host`) is distinct from source connection hosts.
- Frontend exception captures may include `error_frame` (bundle file name with
  line and column of the first same-origin stack frame) and `error_component`
  (innermost React component name). Both are build identifiers; never add the
  message, full stack, or URL.
- New properties must pass the sanitizer allowlists
  (`SAFE_ANALYTICS_PROPERTY_KEYS` in `frontend/src/config/analytics-events.ts`;
  backend detail allowlists in the edition analytics wrappers). No hostnames,
  emails, SQL, credentials, file paths, or free text.
- High-cardinality values (counts, sizes, durations, text lengths) go through
  the bucket helpers, not raw.

## Consent

- Single source of truth: `frontend/src/config/posthog.tsx` — use its exports
  (`CONSENT_STORAGE_KEY`, `remoteAnalyticsAllowed`, `grantStoredConsentIfUnset`,
  `getStoredConsentState`). Never read or write the consent localStorage key
  directly, and never duplicate the host-gating logic.
- Consent-unknown users are not captured (`opt_out_capturing_by_default`).

## Install heartbeat (independent capture control)

`telemetry.heartbeat` (`core/src/analytics/heartbeat.go`) is a pseudonymous
install counter, sent once at startup and every 24h from Go backends and
desktop. It deliberately bypasses `Enabled()` and `buildProperties()`:

- Distinct ID is a **stable random install UUID** persisted in the unified
  config file — this makes unique installs, retention, and version adoption
  measurable. It contains no direct user or machine identifier, and deleting
  the config file resets it. The CLI currently reuses this UUID for command and
  MCP analytics, so it is linkable to that behavior. Keep installation reporting
  separate from behavioral retention; do not describe this ID as unlinkable.
- `$process_person_profile: false` (no person created) and `$ip: ""` (IP
  discarded at ingestion).
- Payload is fixed: `build_edition`, `app_version`, `dev_build`, `os`, `arch`,
  `source`. Never add request context, usage data, person identifiers, or new
  properties to it.
- Source builds (no ldflags-stamped version) are tagged `dev_build: true` and
  `app_version: "dev"` so real installs are filterable, not skipped. CI runs
  (`CI=true`) are skipped entirely.
- Governed solely by `WHODB_HEARTBEAT_DISABLED=true`, independent of the
  consent system. It is publicly documented in the README; keep code, README,
  and this doc in sync if the payload ever changes.

## Verifying changes

- A PostHog "data quality" insight counts events with null `build_edition` or
  `source`, broken down by `$lib`. If your change makes it nonzero, you broke
  the contract.
- Locally, frontend events echo to the debug sink
  (`window.__WHODB_ANALYTICS_DEBUG__` and console) when remote analytics is
  disallowed — check required properties there before shipping.
