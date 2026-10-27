---
name: apollo-router
description: >
  Version-aware guide for configuring and running Apollo Router for federated GraphQL supergraphs.
  Generates correct YAML for both Router v1.x and v2.x. Use this skill when:
  (1) setting up Apollo Router to run a supergraph,
  (2) configuring routing, headers, or CORS,
  (3) implementing custom plugins (Rhai scripts or coprocessors),
  (4) configuring telemetry (tracing, metrics, logging),
  (5) troubleshooting Router performance or connectivity issues,
  (6) securing the graph with JWT, declarative field-level authorization directives, or persisted-query safelisting,
  (7) managing router.yaml as version-controlled config with CI/CD validation.
license: MIT
compatibility: Linux/macOS/Windows. Requires a composed supergraph schema from Rover or GraphOS.
metadata:
  author: apollographql
  version: "2.5.0"
allowed-tools: Bash(router:*) Bash(./router:*) Bash(rover:*) Bash(curl:*) Bash(docker:*) Read Write Edit Glob Grep
---

# Apollo Router Config Generator

Apollo Router is a high-performance graph router written in Rust for running Apollo Federation 2 supergraphs. It sits in front of your subgraphs and handles query planning, execution, and response composition.

**This skill generates version-correct configuration.** Router v1 and v2 have incompatible config schemas in several critical sections (CORS, JWT auth, connectors). Always determine the target version before generating any config.

## Step 1: Version Selection

Ask the user **before generating any config**:

```
Which Apollo Router version are you targeting?

  [1] Router v2.x (recommended — current LTS, required for Connectors)
  [2] Router v1.x (legacy — end-of-support announced, security patches only)
  [3] Not sure — help me decide
```

If the user picks **[3]**, display:

```
Quick guide:

  • Pick v2 if: you're starting fresh, using Apollo Connectors for REST APIs,
    or want backpressure-based overload protection.
  • Pick v1 if: you have an existing deployment and haven't migrated yet.
    Note: Apollo ended active support for v1.x. The v2.10 LTS (Dec 2025)
    is the current baseline. Migration is strongly recommended.

  Tip: If you have an existing router.yaml, you can auto-migrate it:
    router config upgrade router.yaml
```

Store the selection as `ROUTER_VERSION=v1|v2` to gate all subsequent template generation.

## Step 2: Environment Selection

Ask: **Production** or **Development**?

- **Production**: security-hardened defaults (introspection off, sandbox off, homepage off, subgraph errors hidden, auth required, health check on)
- **Development**: open defaults (introspection on, sandbox on, errors exposed, text logging)

Load the appropriate base template from:
- `templates/{version}/production.yaml`
- `templates/{version}/development.yaml`

## Step 3: Feature Selection

Ask which features to include:

- [ ] JWT Authentication
- [ ] Declarative Authorization (field-level `@authenticated` / `@requiresScopes` / `@policy` directives — requires GraphOS + request claims)
- [ ] CORS (almost always yes for browser clients)
- [ ] Operation Limits
- [ ] Traffic Shaping / Rate Limiting
- [ ] Telemetry (Prometheus, OTLP tracing, JSON logging)
- [ ] APQ (Automatic Persisted Queries — performance/bandwidth only, NOT a security control)
- [ ] Persisted Query Safelisting (GraphOS PQL operation allowlist — a security control; distinct from APQ)
- [ ] Connectors (REST API integration — Router v2 only; GA key is `connectors`, early v2 preview key was `preview_connectors`)
- [ ] Subscriptions
- [ ] Header Propagation
- [ ] Response Caching (entity + root field caching with Redis — Router v2 only, v2.6.0+)

## Step 4: Gather Parameters

For each selected feature, collect required values.

- Use section templates from `templates/{version}/sections/` for `auth`, `cors`, `headers`, `limits`, `telemetry`, and `traffic-shaping`.
- For Connectors in v2, use `templates/v2/sections/connectors.yaml` as the source.
- For APQ and subscriptions, copy the snippet from the selected base template (`templates/{version}/production.yaml` or `templates/{version}/development.yaml`) or from references.
- Only offer Connectors when `ROUTER_VERSION=v2`.

### CORS
- List of allowed origins (never use `"*"` for production)

### JWT Authentication
- JWKS URL
- Issuer(s) — note: v1 uses singular `issuer`, v2 uses plural `issuers` array

### Declarative Authorization (field-level)

> Field- and type-level access control enforced **in the router**, via the `@authenticated`, `@requiresScopes`, and `@policy` directives applied in subgraph schemas. This is the layer that the global `authorization.require_authentication` gate cannot express. It is a **GraphOS feature** (Enterprise; Developer/Standard plans require Router v2.6.0+) and requires a router connected to GraphOS. Directives are **enabled by default** — config only turns them *off*.

Confirm prerequisites before recommending these:

- **Router connected to GraphOS** (Router v1.29.1+; Developer/Standard plans need v2.6.0+).
- **A claims source.** Directives evaluate the claims at the `apollo::authentication::jwt_claims` context key. Populate it via JWT authentication (configure that feature too) **or** a coprocessor that injects claims.
- **`@policy` additionally requires a Supergraph plugin** (Rhai script or coprocessor) to evaluate each policy — the router extracts required policies into `apollo::authorization::required_policies` but does not decide them itself.

Ask:
- **Which fields/types need protection, and at what level?** (`@authenticated` = any valid identity; `@requiresScopes` = specific scopes; `@policy` = custom logic.)
- **Where do scopes/claims come from?** (JWT claims vs. coprocessor-injected.)

The directives live in the **subgraph schemas**, not in `router.yaml`. The router config only enables/disables the feature and (for `@policy`) wires the evaluating plugin. See `references/configuration.md` → Authorization.

### Persisted Query Safelisting (GraphOS PQL)

> **Not the same as APQ.** APQ (`apq`) is a runtime bandwidth optimization that caches *any* operation a client sends — it provides **no** security. Safelisting uses a GraphOS-managed **Persisted Query List (PQL)** that clients register at build time; the router then **rejects operations not on the list**. This is the "persisted query safelisting" security control. It is a **GraphOS feature** requiring a router connected to GraphOS (`APOLLO_KEY` + `APOLLO_GRAPH_REF`).

Pick a **security level** (increasing restrictiveness):

| Level | Config | Behavior |
|-------|--------|----------|
| Audit (recommended first) | `persisted_queries.log_unknown: true` | Logs unregistered operations; rejects nothing. Use to confirm all clients are registered before enforcing. |
| Safelist | `safelist.enabled: true` | Rejects operations not in the PQL. IDs *and* full strings both accepted if registered. |
| Safelist, IDs only | `safelist.enabled: true` + `require_id: true` | Rejects unregistered operations **and** any freeform operation string, even if the string is registered. |

Then gather:
- **Is the router GraphOS-connected?** Safelisting needs the PQL fetched from GraphOS (or `local_manifests` for offline licenses).
- **Have clients published their operations to the PQL** (via `rover persisted-queries publish` in their CI/CD)? If not, start in audit mode.
- When enabling `safelist`, **APQ must be disabled** (`apq.enabled: false`) — they are mutually exclusive.

Config key history: GA `persisted_queries` since v1.32.0 (was `preview_persisted_queries` in v1.25.0–v1.32.0); GA in all v2. See `references/configuration.md` → Persisted Query Safelisting.

### Connectors (v2 only)
- Subgraph name and source name (used as `connectors.sources.<subgraph>.<source>`)
- Optional `$config` values for connector runtime configuration
- If migrating old v2 preview config, rename `preview_connectors` to `connectors`

### Operation Limits
Present the tuning guidance:

```
Operation depth limit controls how deeply nested a query can be.

  Router default: 100 (permissive — allows very deep queries)
  Recommended starting point: 50

  Lower values (15–25) are more secure but will reject legitimate queries
  in schemas with deep entity relationships or nested fragments.
  Higher values (75–100) are safer for compatibility but offer less
  protection against depth-based abuse.

  Tip: Run your router in warn_only mode first to see what depths your
  real traffic actually uses, then tighten:
    limits:
      warn_only: true

What max_depth would you like? [default: 50]
```

The same principle applies to `max_height`, `max_aliases`, and `max_root_fields`.

### Telemetry
- OTEL collector endpoint (default: `http://otel-collector:4317`)
- Prometheus listen port (default: `9090`)
- Trace sampling rate (default: `0.1` = 10%)

### Traffic Shaping
- Client-facing rate limit capacity (default: 1000 req/s)
- Router timeout (default: 60s)
- Subgraph timeout (default: 30s)

### Response Caching (v2 only, v2.6.0+)

> **Security: data leakage risk.** Before generating any response cache config, you MUST ask the user which types and fields return user-specific data.  Cached data defaults to shared — subgraph responses without `Cache-Control: private` are visible to all users.  User-specific subgraphs must return `Cache-Control: private` and have `private_id` configured on the router.

- Ask: **Which subgraphs serve user-specific data?** (e.g., accounts, profiles, carts)
- Ask: **How do you identify users?** (JWT `sub` claim, session token, API key)
- Redis URL (default: `redis://localhost:6379`)
- Default TTL (default: `5m`)
- Enable active invalidation? If yes: invalidation listen address and shared key
- Use section template: `templates/v2/sections/response-caching.yaml`
- For security requirements, schema directives, and advanced config: `references/response-caching.md` (start with the Security section)

## Step 5: Generate Config

1. Load the correct version template from `templates/{version}/`
2. Assemble section templates for supported sectioned features, then merge base-template snippets for APQ/subscriptions as needed
3. Inject user-provided parameters
4. Add a comment block at the top stating the target version

## Step 6: Validate

Run the [post-generation checklist](validation/checklist.md):

- [ ] All env vars referenced in config are documented
- [ ] CORS origins don't include wildcards (production)
- [ ] Rate limiting is on `router:` (client-facing), not only `all:` (subgraph)
- [ ] JWT uses `issuers` (v2) not `issuer` (v1), or vice versa
- [ ] If production: introspection=false, sandbox=false, subgraph_errors=false
- [ ] Health check is enabled
- [ ] Homepage is disabled (production)
- [ ] Run: `router config validate <file>` if Router binary is available

## Required Validation Gate (always run)

After generating or editing any `router.yaml`, you MUST:

1. Run `validation/checklist.md` and report pass/fail for each checklist item.
2. Run `router config validate <path-to-router.yaml>` if Router CLI is available.
3. If Router CLI is unavailable, state that explicitly and still complete the checklist.
4. Do not present the configuration as final until validation is completed.

## Configuration as Code (git + CI/CD)

`router.yaml` is the router's contract with every request — treat it like application code, not an ops afterthought. Whenever you generate or edit config, steer the user toward this workflow:

- **Commit `router.yaml` to version control.** It should live in git alongside the service, with changes reviewed via pull request. This gives you history, blame, and rollback for the most safety-critical file in the API layer.
- **Never commit secrets.** Keep `APOLLO_KEY`, JWKS URLs, Redis URLs, and invalidation keys out of the file — reference them with `${env.*}` expansion and inject at deploy time. The committed file should be safe to read by anyone with repo access.
- **Validate in CI.** Run `router config validate router.yaml` on every PR so a malformed or version-mismatched config fails the build before it ships. Pin the Router version used in CI to the version you deploy.
- **Pair config changes with schema checks.** Schema changes flow through `rover subgraph check` / `rover subgraph publish` (the `rover` skill); config changes flow through this validate-in-CI gate. Both gate the same deploy.
- **Promote the same file across environments.** Differences between dev and prod should be expressed through env vars, not divergent committed files, so what you reviewed is what runs.

A minimal CI step (provide actual commands only if asked):

```yaml
# Validate router config on every pull request
- run: router config validate router.yaml
```

## Step 7: Conditional Next Steps Handoff

After answering any Apollo Router request (config generation, edits, validation, or general Router guidance), decide whether the user already has runnable prerequisites:

- GraphOS-managed path: `APOLLO_KEY` + `APOLLO_GRAPH_REF`, or
- Local path: a composed `supergraph.graphql` plus reachable subgraphs

If prerequisites are already present, do not add extra handoff text.

If prerequisites are missing or unknown, end with a concise **Next steps** handoff (1-3 lines max) that is skill-first and command-free:

1. Suggest the `rover` skill to compose or fetch the supergraph schema.
2. Suggest continuing with `apollo-router` once the supergraph is ready to validate and run with the generated config.
3. If subgraphs are missing, suggest `apollo-server`, `graphql-schema`, and `graphql-operations` skills to scaffold and test.

Do not include raw shell commands in this handoff unless the user explicitly asks for commands.

## Quick Start (skill-first)

1. Use this `apollo-router` skill to generate or refine `router.yaml` for your environment.
2. Choose a runtime path:
   - GraphOS-managed path: provide `APOLLO_KEY` and `APOLLO_GRAPH_REF` (no local supergraph composition required).
   - Local supergraph path: use `graphql-schema` + `apollo-server` to define/run subgraphs, then use `graphql-operations` for smoke tests, then use the `rover` skill to compose or fetch `supergraph.graphql`.
3. Use this `apollo-router` skill to validate readiness (`validation/checklist.md`) and walk through runtime startup inputs.

Default endpoint remains `http://localhost:4000` when using standard Router listen defaults.

If the user asks for executable shell commands, provide them on request. Otherwise keep Quick Start guidance skill-oriented.

## Running Modes

| Mode | Command | Use Case |
|------|---------|----------|
| Local schema | `router --supergraph ./schema.graphql` | Development, CI/CD |
| GraphOS managed | `APOLLO_KEY=... APOLLO_GRAPH_REF=my-graph@prod router` | Production with auto-updates |
| Development | `router --dev --supergraph ./schema.graphql` | Local development |
| Hot reload | `router --hot-reload --supergraph ./schema.graphql` | Schema changes without restart |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `APOLLO_KEY` | API key for GraphOS |
| `APOLLO_GRAPH_REF` | Graph reference (`graph-id@variant`) |
| `APOLLO_ROUTER_CONFIG_PATH` | Path to `router.yaml` |
| `APOLLO_ROUTER_SUPERGRAPH_PATH` | Path to supergraph schema |
| `APOLLO_ROUTER_LOG` | Log level (off, error, warn, info, debug, trace) |
| `APOLLO_ROUTER_LISTEN_ADDRESS` | Override listen address |

## Reference Files

- [Configuration](references/configuration.md) — YAML configuration reference
- [Headers](references/headers.md) — Header propagation and manipulation
- [Plugins](references/plugins.md) — Rhai scripts and coprocessors
- [Telemetry](references/telemetry.md) — Tracing, metrics, and logging
- [Connectors](references/connectors.md) — Router v2 connectors configuration
- [Response Caching](references/response-caching.md) — Entity/root-field caching, invalidation, and observability (v2 only)
- [Troubleshooting](references/troubleshooting.md) — Common issues and solutions
- [Divergence Map](divergence-map.md) — v1 ↔ v2 config differences
- [Validation Checklist](validation/checklist.md) — Post-generation checks

## CLI Reference

```
router [OPTIONS]

Options:
  -s, --supergraph <PATH>    Path to supergraph schema file
  -c, --config <PATH>        Path to router.yaml configuration
      --dev                  Enable development mode
      --hot-reload           Watch for schema changes
      --log <LEVEL>          Log level (default: info)
      --listen <ADDRESS>     Override listen address
  -V, --version              Print version
  -h, --help                 Print help
```

## Ground Rules

- ALWAYS determine the target Router version (v1 or v2) before generating config
- DEFAULT to v2 for new projects
- ALWAYS include a comment block at top of generated config stating the target version
- ALWAYS use `--dev` mode for local development (enables introspection and sandbox)
- ALWAYS disable introspection, sandbox, and homepage in production
- PREFER GraphOS managed mode for production (automatic updates, metrics)
- USE `--hot-reload` for local development with file-based schemas
- NEVER expose `APOLLO_KEY` in logs or version control
- USE environment variables (`${env.VAR}`) for all secrets and sensitive config
- PREFER YAML configuration over command-line arguments for complex setups
- TEST configuration changes locally before deploying to production
- WARN if user enables `allow_any_origin` or wildcard CORS in production
- RECOMMEND `router config upgrade router.yaml` for v1 → v2 migration instead of regenerating from scratch
- MUST run `validation/checklist.md` after every router config generation or edit
- MUST run `router config validate <file>` when Router CLI is available
- MUST report when CLI validation could not run (for example, Router binary missing)
- MUST append a brief conditional handoff when runtime prerequisites are missing or unknown
- MUST make this handoff skill-first and avoid raw shell commands unless the user explicitly requests commands
- MUST keep Quick Start guidance skill-first and command-free unless the user explicitly requests commands
- MUST state that Rover is required only for the local supergraph path; GraphOS-managed runtime does not require local Rover composition
- USE `max_depth: 50` as the default starting point, not 15 (too aggressive) or 100 (too permissive)
- RECOMMEND `warn_only: true` for initial limits rollout to observe real traffic before enforcing
- ONLY offer Response Caching when `ROUTER_VERSION=v2` (requires v2.6.0+)
- ALWAYS use `${env.*}` for Redis URLs, passwords, and invalidation shared keys
- NEVER enable `response_cache.debug: true` in production config
- RECOMMEND combining Cache-Control headers (passive TTL) with @cacheTag (active invalidation) for production
- ALWAYS ask which fields return user-specific data before generating response cache config — never assume all data is safe to cache as shared
- ALWAYS configure `private_id` for subgraphs that serve user-specific data, and ensure those subgraphs return `Cache-Control: private` (via `@cacheControl(scope: PRIVATE)` in Apollo Server, or by setting the header directly in other frameworks)
- NEVER generate response cache config without addressing private data — if the user says "no user-specific data", confirm explicitly before proceeding
- ALWAYS bind the invalidation endpoint to `127.0.0.1`, NEVER `0.0.0.0` in production
- NEVER conflate APQ with persisted-query safelisting — APQ (`apq`) is a bandwidth optimization with no security value; safelisting (`persisted_queries.safelist`) is the operation allowlist. If a user asks to "lock down which queries can run", point them to safelisting, not APQ
- ALWAYS disable APQ (`apq.enabled: false`) when enabling `persisted_queries.safelist` — they are mutually exclusive
- RECOMMEND starting persisted queries in audit mode (`log_unknown: true`) to confirm all clients are registered before turning on `safelist.enabled`
- STATE that persisted-query safelisting requires a GraphOS-connected router (PQL fetched via `APOLLO_KEY` + `APOLLO_GRAPH_REF`, or `local_manifests` for offline licenses)
- USE `persisted_queries` (GA, v1.32.0+ and all v2), NOT `preview_persisted_queries` (v1.25.0–v1.32.0)
- TREAT global `authorization.require_authentication` and declarative directives as different layers: the former gates the whole request, the latter (`@authenticated` / `@requiresScopes` / `@policy`) does field- and type-level filtering
- STATE that declarative authorization directives require a GraphOS-connected router (v1.29.1+; Developer/Standard plans need v2.6.0+) and a claims source (JWT auth or a coprocessor populating `apollo::authentication::jwt_claims`)
- NOTE that authorization directives are ENABLED BY DEFAULT — `authorization.directives.enabled: false` only turns them off; never imply config is required to "turn them on"
- STATE that `@policy` additionally requires a Rhai script or coprocessor at the Supergraph stage to evaluate `apollo::authorization::required_policies`
- PLACE authorization directives in subgraph schemas, NEVER in `router.yaml` — router config only enables/disables the feature
- RECOMMEND committing `router.yaml` to version control and running `router config validate` in CI on every PR, with all secrets referenced via `${env.*}` and injected at deploy time
- NEVER commit secrets (`APOLLO_KEY`, JWKS/Redis URLs, invalidation keys) to the config file; the committed `router.yaml` must be safe to share with anyone holding repo access
