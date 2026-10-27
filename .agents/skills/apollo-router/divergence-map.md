# Apollo Router v1 ↔ v2 Config Divergence Map

This document tracks every configuration section where v1 and v2 schemas differ
in ways that break parsing or change behavior. All templates in this skill branch
on these differences.

## CORS

| Aspect | v1 | v2 |
|--------|----|----|
| Origins | `cors.origins: [...]` (flat list) | `cors.policies: [{ origins: [...] }]` (array of policy objects) |
| Headers | `cors.allow_headers` | Same key, but inside each policy or at global level |
| Methods | `cors.methods` | Same, inheritable per-policy |
| Max Age | Duration string (for example: `cors.max_age: 24h`) | Same format (duration string) |
| Credentials | `cors.allow_credentials` | Same, inheritable per-policy |

See: [templates/v1/sections/cors.yaml](templates/v1/sections/cors.yaml) vs [templates/v2/sections/cors.yaml](templates/v2/sections/cors.yaml)

## JWT Authentication

| Aspect | v1 | v2 |
|--------|----|----|
| Issuer field | `issuer: <string>` (singular) | `issuers: [<string>, ...]` (plural, array) |
| Poll interval | `poll_interval` supported | Same |

> [!CAUTION]
> `issuer` is a v1 field. In v2, always use `issuers` (array).

See: [templates/v1/sections/auth.yaml](templates/v1/sections/auth.yaml) vs [templates/v2/sections/auth.yaml](templates/v2/sections/auth.yaml)

## Authorization Directives (`@authenticated` / `@requiresScopes` / `@policy`)

The enable key is the same in both versions — what diverges is availability and the claims context key read by coprocessors.

| Aspect | v1 | v2 |
|--------|----|----|
| Enable key | `authorization.directives.enabled` (default `true`) | Same |
| Minimum version | v1.29.1+ (GraphOS-connected) | All v2; Developer/Standard plans require v2.6.0+ |
| Claims context key | `apollo_authentication::JWT::claims` | `apollo::authentication::jwt_claims` |

> [!NOTE]
> Directives are **on by default** — config only disables them. They live in subgraph schemas, not `router.yaml`.

## Persisted Queries (Safelisting)

Distinct from APQ. The config key changed across v1 preview → GA:

| Aspect | v1.25.0–v1.32.0 (preview) | v1.32.0+ / all v2 (GA) |
|--------|---------------------------|------------------------|
| Config key | `preview_persisted_queries` | `persisted_queries` |
| `local_manifests` key | `experimental_local_manifests` (v1.50.0–v1.54) | `local_manifests` (v1.55.0+) |

> [!CAUTION]
> Enabling `persisted_queries.safelist` requires `apq.enabled: false` — APQ and safelisting are mutually exclusive.

## Operation Limits

| Aspect | v1 (early) | v1 (1.17+) / v2 |
|--------|-----------|------------------|
| Config key | `preview_operation_limits` | `limits` |

Both v1.17+ and v2 use `limits`. Only very old v1 installs use `preview_operation_limits`.

## Connectors (Router v2 only)

Connectors are a Router v2 feature. The relevant schema difference is early v2 preview vs current v2 GA:

| Aspect | v2.0.x (preview) | v2.1+ (GA) |
|--------|-------------------|------------|
| Config key | `preview_connectors` | `connectors` |
| Subgraph/source mapping | `subgraphs.<name>.sources.<source>` | `sources.<subgraph>.<source>` (dot notation) |

Only relevant if using Apollo Connectors for REST API integration.

## Telemetry

| Aspect | v1 | v2 |
|--------|----|----|
| `experimental_when_header` | Supported | Removed — use custom telemetry events |
| Metric names | Legacy names | OpenTelemetry naming conventions |
| Apollo reporting | FTV1 by default | OTLP by default (`otlp_tracing_sampler` defaults to 1.0) |
| Prometheus exporter | `telemetry.exporters.metrics.prometheus` | Same path |

Telemetry templates are largely compatible. v2 uses OTLP-first reporting.

## Traffic Shaping

Schema is identical between v1 and v2. Behavioral change: v2 returns **HTTP 429**
for rate-limited requests (restored in v2.1.0 after briefly returning 503 in v2.0.0).

## Context Keys (Coprocessors)

Multiple context keys were renamed in v2. Only relevant if configuring coprocessors.

| v1 Key | v2 Key |
|--------|--------|
| `apollo_authentication::JWT::claims` | `apollo::authentication::jwt_claims` |
| `apollo_operation_id` | `apollo::operation_id` |

> [!TIP]
> If using coprocessors during migration, set `context: deprecated` to preserve
> v1 key names while transitioning.

## Key Documentation References

| Topic | URL |
|-------|-----|
| v2 YAML Reference | https://www.apollographql.com/docs/graphos/routing/configuration/yaml |
| v1 → v2 Upgrade Guide | https://www.apollographql.com/docs/graphos/routing/upgrade/from-router-v1 |
| What's New in v2 | https://www.apollographql.com/docs/graphos/routing/about-v2 |
| CORS (v2) | https://www.apollographql.com/docs/graphos/routing/security/cors |
| JWT Auth | https://www.apollographql.com/docs/graphos/routing/security/jwt |
| Authorization Directives | https://www.apollographql.com/docs/graphos/routing/security/authorization |
| Persisted Query Safelisting | https://www.apollographql.com/docs/graphos/platform/security/persisted-queries |
| Request Limits | https://www.apollographql.com/docs/graphos/routing/security/request-limits |
| Traffic Shaping | https://www.apollographql.com/docs/graphos/routing/performance/traffic-shaping |
| v1 EOL Announcement | https://www.apollographql.com/blog/end-of-support-for-router-v1-x-long-term-support-lts-policy-update |
