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
| `app_version` | semver | Go config; TS not applicable |
| `platform` | `browser`, `wails` | Web events only (TS runtime context) |
| `$host` | request/browser host | Go: from request metadata; TS: auto-captured by posthog-js |

Choke points:
- Go: `buildProperties()` in `core/src/analytics/posthog.go` — every capture
  from every Go sender flows through it.
- TS: `sanitizeAnalyticsProperties()` in
  `frontend/src/config/analytics-sanitize.ts`.

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

## Identity

- Backend distinct IDs resolve in order: registered resolver
  (`analytics.SetDistinctIDResolver`, set at boot by edition entry points) →
  `X-WhoDB-Analytics-Id` header from the frontend → `anonymous`.
- Never invent synthetic distinct IDs (random, per-process, hashed timestamps).
  Unattributable events belong to `anonymous`.

## Privacy

- Never send raw error messages. Map errors through `analytics.ErrorCode`
  (Go) or the frontend error-code helpers to a fixed taxonomy.
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

## Install heartbeat (separate from all of the above)

`telemetry.heartbeat` (`core/src/analytics/heartbeat.go`) is an anonymous
install counter, sent once at startup and every 24h from Go backends and
desktop. It deliberately bypasses `Enabled()` and `buildProperties()`:

- Distinct ID is a **stable random install UUID** persisted in the unified
  config file — this makes unique installs, retention, and version adoption
  measurable. It has no relation to the user or machine, and deleting the
  config file resets it.
- `$process_person_profile: false` (no person created) and `$ip: ""` (IP
  discarded at ingestion).
- Payload is fixed: `build_edition`, `app_version`, `dev_build`, `os`, `arch`,
  `source`. Never add request context, usage data, or new properties to it —
  the install ID must never become linkable to behavioral analytics or a
  person.
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
