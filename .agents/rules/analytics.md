---
paths:
  - "core/src/analytics/**"
  - "cli/pkg/analytics/**"
  - "frontend/src/config/posthog.tsx"
  - "frontend/src/config/analytics-*.ts"
  - "frontend/src/config/frontend-analytics.ts"
---

# Analytics Rules (PostHog)

All senders (web, backend, CLI, desktop) share one PostHog project, so events must stay consistent across them.

## Choke points
- Go captures go through `buildProperties()` (`core/src/analytics/posthog.go`); TS events go through `sanitizeAnalyticsProperties()` (`frontend/src/config/analytics-sanitize.ts`). They stamp `source`, `build_edition`, `build_environment`, `app_version` and (web only) `platform`. Never set these at call sites or bypass the choke points. Go callers must set `Config.Edition`.
- Use the `build_`-prefixed names. Don't add `environment`, `edition` or a second meaning of `platform`; desktop OS goes in `os`. `client_type` is the calling surface (browser, desktop, mobile, cli, mcp), distinct from the emitting `source`.

## Ownership
- The backend owns "it happened" events (resources, runs, auth, query outcomes); server capture can't be ad-blocked and is the source of truth. The frontend owns UX-only events (screen views, `*_opened`/`*_abandoned` funnels, client failures). Never emit one concept from both sides.
- `cli.query.execute` (success) and `cli.query.error` (failure, older events lack `success`) must be combined for a success rate. A local MCP response with `outcome: confirmation_required` is not an executed write. Failed local MCP tool calls carry `error_code` from the tool's fixed `error_type` (unknown ones become `internal_error`), never the raw message; security-policy and disallowed-connection responses also set `denied: true` and `outcome: denied`.

## Privacy
- Never send raw error messages; map them through `analytics.ErrorCode` (Go) or `frontendAnalyticsErrorCode` (TS). Canceled work is `error_code: canceled`, distinct from timeouts.
- No SQL, object paths, column names, values, hostnames, emails, credentials or free text. New properties must pass `SAFE_ANALYTICS_PROPERTY_KEYS` (`frontend/src/config/analytics-events.ts`).
- Counts, sizes, durations and lengths go through `analytics.DurationBucket` / `analytics.CountBucket`; unknown negative sizes are omitted, not reported as empty.
- Frontend exceptions may carry `error_frame` and `error_component` only, never the message, stack or URL.

## Identity and consent
- Backend distinct ID: resolver from `analytics.SetDistinctIDResolver`, then the `X-WhoDB-Analytics-Id` header, then `anonymous`. Never invent synthetic IDs.
- Consent lives in `frontend/src/config/posthog.tsx`; use its exports and never read or write the consent localStorage key directly.

## Install heartbeat
- `telemetry.heartbeat` (`core/src/analytics/heartbeat.go`) deliberately bypasses `Enabled()` and `buildProperties()`. Its payload is fixed (`build_edition`, `app_version`, `dev_build`, `os`, `arch`, `source`, no IP, no person profile); add nothing to it. It is skipped when `CI=true` and disabled by `WHODB_HEARTBEAT_DISABLED=true` (documented in `docs/installation.mdx`).
- Its install UUID is also used by CLI and MCP analytics, so it is linkable; don't describe it as unlinkable.

## Verify
- Go tests in `core/src/analytics/` and `pnpm test:analytics` in `frontend/`. Locally, events echo to `window.__WHODB_ANALYTICS_DEBUG__` when remote analytics is off.
