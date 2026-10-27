# Response Caching (Router v2.6.0+)

Entity-level and root-field caching for subgraph responses, backed by Redis. Requires Router v2.6.0 or later.

## Overview

Response caching enables the router to cache origin responses and reuse them across queries:

- **Entity representations**: Cached independently per origin — each origin's contribution to an entity is cached separately and reusable across different queries
- **Root query fields**: Cached as complete units (the entire response for that root field)

Response caching applies to **query operations only** — mutations and subscriptions are never cached.

### Scope

Scope is determined by the `Cache-Control` header the subgraph returns.  (The router docs call these "PUBLIC" and "PRIVATE" — we use "shared" and "private" here because they describe the actual behavior more clearly.)

- **Shared** (default — no `private` directive in the header): Data is identical for all users and stored in a single shared cache entry
- **Private** (`Cache-Control: private`): Data is user-specific; requires `private_id` configuration to cache per-user in separate entries

### Mixed TTLs

When an origin response contains multiple entity representations, the router uses the minimum TTL across all representations. Client responses never claim to be fresher than their least-fresh component.

## Security

### Data leakage risk: cached data defaults to shared

> **Security: cross-user data leakage.** Caching requires opt-in — subgraphs must return a `Cache-Control` header for the router to cache anything.  But once a response IS cached, it defaults to PUBLIC scope (shared across all users).  If a subgraph returns `Cache-Control: max-age=300` for user-specific data without also including `private`, that data is shared across all users.

The risk is not "everything gets cached" — it's "data that IS cached defaults to shared."  If a subgraph opts into caching but forgets to mark user-specific responses as `private`, those responses are silently served to other users.

### Before enabling response caching, you MUST:

1. **Identify every field that returns user-specific data** — ask the user which types and fields vary per user. Do not guess. Common examples: user profiles, preferences, bookmarks, cart contents, order history, notifications, permissions.

2. **Ensure user-specific responses include `Cache-Control: private`** — how you do this depends on your subgraph framework:
   - **Apollo Server**: use `@cacheControl(scope: PRIVATE)` in your schema
   - **Other servers**: set the header directly, e.g., `Cache-Control: private, max-age=60`

   Apollo Server example:
   ```graphql
   type User @key(fields: "id") {
     id: ID!
     name: String!
     email: String! @cacheControl(maxAge: 60, scope: PRIVATE)
     preferences: UserPreferences! @cacheControl(maxAge: 300, scope: PRIVATE)
   }
   ```

3. **Configure `private_id`** for every subgraph that serves private-scoped data:
   ```yaml
   response_cache:
     enabled: true
     subgraph:
       subgraphs:
         accounts:
           private_id: "user_id"  # REQUIRED for private scope to work
   ```

4. **Extract the user identifier** from the request (e.g., JWT `sub` claim) via a Rhai script:
   ```rhai
   fn supergraph_service(service) {
     let request_callback = |request| {
       let claims = request.context[Router.APOLLO_AUTHENTICATION_JWT_CLAIMS];
       if claims != () {
         request.context["user_id"] = claims["sub"];
       }
     };
     service.map_request(request_callback);
   }
   ```

**What happens if `private_id` is missing:** When a subgraph returns `Cache-Control: private` but no `private_id` is configured, the router cannot identify the user.  The cache is bypassed entirely for those requests — no data leakage occurs, but you get no caching benefit.  However, if a response *should* be private but the subgraph doesn't include `private` in the `Cache-Control` header, it will be cached as shared and served to all users.

### Additional security requirements

- **Debug mode**: NEVER enable `response_cache.debug: true` in production. It exposes internal cache keys, tags, and entry metadata to Apollo Sandbox.
- **Invalidation endpoint**: ALWAYS bind to loopback (`127.0.0.1`), NEVER `0.0.0.0`. Exposing the invalidation endpoint publicly allows anyone to flush your cache.
- **Shared key**: ALWAYS use `${env.INVALIDATION_SHARED_KEY}` for the invalidation shared key. Never hardcode it.
- **Redis credentials**: ALWAYS use `${env.*}` expansion for Redis URLs, usernames, and passwords.

### When NOT to cache

Not all data benefits from caching. Avoid response caching when:

- Data contains highly sensitive information (financial records, health data, PII) where even a misconfiguration risk is unacceptable
- User identification is unreliable across requests (anonymous users, inconsistent auth)
- Data changes too frequently to benefit from caching (real-time feeds, live scores)

## Setup

### Minimal configuration

```yaml
response_cache:
  enabled: true
  subgraph:
    all:
      enabled: true
      ttl: 5m  # Required: used when Cache-Control header lacks max-age
      redis:
        urls: ["${env.CACHE_REDIS_URL:-redis://localhost:6379}"]
```

### Full annotated example

```yaml
response_cache:
  enabled: true
  debug: false  # Set true only in dev — exposes cache data to Apollo Sandbox
  invalidation:
    listen: 127.0.0.1:4000  # Bind to loopback — never expose publicly in production
    path: /invalidation
  subgraph:
    all:
      enabled: true
      ttl: "${env.CACHE_DEFAULT_TTL:-5m}"
      redis:
        urls: ["${env.CACHE_REDIS_URL:-redis://localhost:6379}"]
        fetch_timeout: 250ms   # Default: 150ms
        insert_timeout: 750ms  # Default: 500ms
        invalidate_timeout: 750ms  # Default: 1s
        pool_size: 5           # Default: 5; increase for high traffic
        namespace: response_cache  # Prefix for all Redis keys
        required_to_start: false   # Set true to block startup if Redis is down
      invalidation:
        enabled: true
        shared_key: "${env.INVALIDATION_SHARED_KEY}"
    subgraphs:
      inventory:
        enabled: false  # Disable caching for a specific subgraph
      products:
        ttl: 10m  # Override TTL per subgraph
        redis:
          urls: ["${env.PRODUCTS_REDIS_URL:-redis://products-cache:6379}"]
```

### Redis URL formats

| Scheme | Description |
|--------|-------------|
| `redis://` | TCP to a single server |
| `rediss://` | TLS to a single server |
| `redis-cluster://` | TCP to a cluster |
| `rediss-cluster://` | TLS to a cluster |

Format: `redis[s][-cluster]://[[username:]password@]host[:port][/database]`

Clustered URLs can include `?node=host1:port1&node=host2:port2` query parameters or be provided as a YAML array of URLs.

## Redis deployment

This skill configures the router to talk to Redis, but does not cover deploying or operating Redis itself.  Plan for these operational concerns separately.

### Topology and high availability

For production workloads, use Redis with **replica sets** — a primary with one or more replicas and automated failover.  Most managed Redis services provide this natively (AWS ElastiCache replication groups, Azure Cache for Redis, GCP Memorystore Standard, Upstash, Redis Enterprise, etc.), handling failover transparently at the DNS level so a standalone URL continues to work.

- **Single-node** (`redis://`, `rediss://`) — fine for local development; not recommended for production.
- **Replica set** (still `redis://` or `rediss://`, pointing at the primary endpoint) — the typical production topology.
- **Redis Cluster** (`redis-cluster://`, `rediss-cluster://`) — for horizontal sharding when cache size exceeds a single node's memory.  Most response cache workloads do not need Cluster; start with a replica set and only move to Cluster if you outgrow it.

Keep Redis colocated with routers (same VPC, same region).  Every cache fetch is in the request path, so network latency directly affects user-facing response time.

### Eviction policy

Configure Redis with `allkeys-lru` (or `allkeys-lfu`) eviction.  With `noeviction` (Redis's default), writes fail once memory is full and the router logs insert errors instead of caching new responses.  `volatile-*` policies also work since the router sets TTLs on every entry.

### Graceful degradation

When Redis is unreachable, the router bypasses the cache and sends requests directly to subgraphs — your graph keeps serving traffic without caching benefits, but it does not fail.  This is the default behavior.

If you set `required_to_start: true`, the router refuses to start without a Redis connection.  Use this only when you cannot tolerate a cache-cold deployment.

### Cold start

After a router restart with empty Redis (or a Redis flush), the first wave of requests will all be cache misses, producing a spike of subgraph traffic.  Plan for this when rolling out caching to a high-traffic graph — consider a gradual rollout or pre-warming if your origins are sensitive to the spike.

## Cache-Control Header

The router determines caching behavior from the `Cache-Control` HTTP response header returned by each subgraph.  It does not read schema directives directly — only the headers they produce.

### What the router reads

| Header directive | Effect |
|-----------------|--------|
| `max-age=N` | Cache for N seconds (overrides the configured fallback `ttl`) |
| `private` | Data is user-specific — requires `private_id` to cache per-user |
| `no-store` | Do not cache this response |

If the subgraph returns **no `Cache-Control` header at all**, the response is **not cached** — even when a fallback `ttl` is configured.  The fallback TTL only applies when a `Cache-Control` header is present but lacks `max-age`.

When a subgraph response contains multiple entities with different TTLs, the router uses the minimum `max-age` across all of them.

### How to emit Cache-Control headers

**Apollo Server** — use the `@cacheControl` directive in your subgraph schema.  Apollo Server translates this into `Cache-Control` headers automatically.  See the [Apollo Server caching documentation](https://www.apollographql.com/docs/apollo-server/performance/caching) for the full directive reference (field-level vs. type-level TTLs, `inheritMaxAge`, dynamic TTLs in resolvers, etc.).

```graphql
type Product @key(fields: "id") @cacheControl(maxAge: 240) {
  id: ID!
  name: String!
  price: Int @cacheControl(maxAge: 60)  # Shorter TTL for volatile data
  viewerHasBookmarked: Boolean! @cacheControl(maxAge: 30, scope: PRIVATE)
}
```

**Other GraphQL servers** (Java, Go, Python, etc.) — set the `Cache-Control` header directly in your HTTP response using whatever mechanism your framework provides.  For example: `Cache-Control: public, max-age=240` or `Cache-Control: private, max-age=60`.

> The `@cacheControl` directive is Apollo Server-specific.  The router only sees the HTTP header — how your subgraph produces that header is an implementation detail.

### @cacheTag(format)

Tags cached data for active invalidation.  This is a federation directive (not Apollo Server-specific) — the router reads it from the composed supergraph schema.  Introduced in Federation v2.12.  Import it via:

```graphql
extend schema
  @link(
    url: "https://specs.apollo.dev/federation/v2.12"
    import: ["@key", "@cacheTag"]
  )
```

> `@cacheTag` controls invalidation tags, not whether data is cached.  The subgraph must still return `Cache-Control` headers (see above) for caching to take effect.

**On entities** — use `{$key.<field>}` for dynamic tags based on entity keys:

```graphql
type User @key(fields: "id")
  @cacheTag(format: "user-{$key.id}")
  @cacheTag(format: "user") {
  id: ID!
  name: String!
}
```

**On root query fields** — use `{$args.<field>}` for dynamic tags based on arguments:

```graphql
type Query {
  postsByUser(userId: ID!): [Post!]!
    @cacheTag(format: "posts-user-{$args.userId}")
}
```

**Rules:**
- Only applies to root query fields or resolvable entities (types with `@key` where `resolvable` is unset or `true`)
- For entities with multiple `@key` directives, you can only use fields present in **every** `@key`
- The `format` must always generate a valid string (not an object)

## Invalidation

### Passive (TTL-based)

Data automatically expires based on:
1. `Cache-Control: max-age=N` headers returned by the subgraph (set via `@cacheControl` in Apollo Server, or emitted directly by other servers)
2. The configured `ttl` fallback (used when a `Cache-Control` header is present but lacks `max-age`)

The router uses the minimum TTL across all components in a response.

### Active (tag-based)

Explicitly remove cached data before TTL expires. Requires the invalidation endpoint and `@cacheTag` directives.

**Configuration:**

```yaml
response_cache:
  enabled: true
  invalidation:
    listen: 127.0.0.1:4000  # Internal only — never bind to 0.0.0.0 in production
    path: /invalidation
  subgraph:
    all:
      enabled: true
      redis:
        urls: ["${env.CACHE_REDIS_URL:-redis://localhost:6379}"]
      invalidation:
        enabled: true
        shared_key: "${env.INVALIDATION_SHARED_KEY}"
```

**Invalidation request formats:**

By subgraph (all cached data for the subgraph):
```json
[{"kind": "subgraph", "subgraph": "accounts"}]
```

By entity type:
```json
[{"kind": "type", "subgraph": "accounts", "type": "User"}]
```

By cache tag:
```json
[{"kind": "cache_tag", "subgraphs": ["accounts"], "cache_tag": "user-42"}]
```

**curl example:**

```bash
curl --request POST \
  --header "authorization: $INVALIDATION_SHARED_KEY" \
  --header "content-type: application/json" \
  --url http://localhost:4000/invalidation \
  --data '[{"kind": "cache_tag", "subgraphs": ["posts"], "cache_tag": "user-42"}]'
```

Response: `{"count": 1}` — the number of invalidated Redis keys.

### Programmatic cache tags from subgraph responses

If tags depend on runtime data (not entity keys or field args), set them in the response `extensions`:

- **Entities**: Use `apolloEntityCacheTags` — an array of arrays, positionally matching the `_entities` array:
  ```json
  {
    "data": {"_entities": [
      {"__typename": "User", "id": 42, "name": "Alice"},
      {"__typename": "User", "id": 7, "name": "Bob"}
    ]},
    "extensions": {"apolloEntityCacheTags": [
      ["users", "user-42"],
      ["users", "user-7"]
    ]}
  }
  ```

- **Root fields**: Use `apolloCacheTags` — a flat array of tags for the entire response:
  ```json
  {
    "data": {"homepage": {"featuredProducts": [...]}},
    "extensions": {"apolloCacheTags": ["homepage", "featured"]}
  }
  ```

## Customization

### Private data caching

> **Security: see the [Security section](#security) above.** You MUST configure `private_id` and ensure user-specific subgraph responses include `Cache-Control: private` before enabling caching on subgraphs that serve user-specific data.  Failure to do so causes cross-user data leakage.

For full configuration, Rhai script examples, and the security checklist, see [Security](#security).

### Custom cache keys

Vary cache entries by request headers using the `apollo::response_cache::key` context entry.

**Multi-tenant example** (x-tenant-id header):

```rhai
fn supergraph_service(service) {
  let request_callback = |request| {
    let tenant_id = request.headers["x-tenant-id"];
    if tenant_id != () {
      request.context[Router.APOLLO_RESPONSE_CACHE_KEY]["all"] = tenant_id;
    }
  };
  service.map_request(request_callback);
}
```

**Locale example** (accept-language header):

```rhai
fn supergraph_service(service) {
  let request_callback = |request| {
    let locale = request.headers["accept-language"];
    if locale != () {
      request.context[Router.APOLLO_RESPONSE_CACHE_KEY]["all"] = locale;
    }
  };
  service.map_request(request_callback);
}
```

### Per-subgraph Redis instances

Override the global Redis for specific subgraphs:

```yaml
response_cache:
  enabled: true
  subgraph:
    all:
      enabled: true
      redis:
        urls: ["${env.CACHE_REDIS_URL:-redis://localhost:6379}"]
    subgraphs:
      products:
        redis:
          urls: ["${env.PRODUCTS_REDIS_URL:-redis://products-cache:6379}"]
          pool_size: 15
          namespace: products_response_cache
```

### Redis tuning reference

| Option | Default | Description |
|--------|---------|-------------|
| `fetch_timeout` | 150ms | Timeout for cache reads |
| `insert_timeout` | 500ms | Timeout for cache writes |
| `invalidate_timeout` | 1s | Timeout for invalidation operations |
| `pool_size` | 5 | Number of Redis connections |
| `namespace` | (none) | Prefix for all Redis keys |
| `required_to_start` | false | Block router startup if Redis is unreachable |

### TLS and authentication

```yaml
response_cache:
  enabled: true
  subgraph:
    all:
      enabled: true
      redis:
        urls: ["rediss://${env.REDIS_HOST}:6379"]
        username: "${env.REDIS_USERNAME}"
        password: "${env.REDIS_PASSWORD}"
        tls:
          certificate_authorities: "${file./path/to/ca.crt}"
          client_authentication:
            certificate_chain: "${file./path/to/certificate_chain.pem}"
            key: "${file./path/to/key.pem}"
```

## Observability

### What to watch

Four signals tell you whether response caching is healthy.  If you set up a dashboard for this feature, start with these:

1. **Cache hit rate** — derived from `apollo.router.response.cache` (counter that records hits and misses for subgraph requests; filter by cache status to compute hit / (hit + miss)).  No universal target — depends on TTLs and traffic shape — but watch for sudden drops, which usually mean subgraphs stopped emitting `Cache-Control` headers or Redis is misbehaving.

2. **Redis error rate** — `apollo.router.cache.redis.errors` (counter, tagged by error type: auth, timeout, io, etc.).  Any sustained non-zero rate warrants investigation.

3. **Cache fetch and insert latency** — `apollo.router.operations.response_cache.fetch` and `apollo.router.operations.response_cache.insert` (histograms, seconds).  The p99 should be well under your subgraph response time, otherwise the cache is *adding* latency rather than removing it.  If p99 climbs, check Redis CPU/memory and the `pool_size` setting.

4. **Reconnection rate** — `apollo.router.response_cache.reconnection`.  Spikes indicate network instability or Redis restarts.

For invalidation specifically, also watch `apollo.router.operations.response_cache.invalidation.error` (failed invalidations leave stale data) and `apollo.router.operations.response_cache.invalidation.duration` (slow invalidations indicate Redis pressure).

### Metrics

#### Fetch / insert

| Metric | Description | Unit |
|--------|-------------|------|
| `apollo.router.operations.response_cache.fetch` | Time to fetch from cache | s |
| `apollo.router.operations.response_cache.fetch.error` | Errors fetching from cache | {error} |
| `apollo.router.operations.response_cache.fetch.entity` | Entities per fetch node | {entity} |
| `apollo.router.operations.response_cache.insert` | Time to insert into cache | s |
| `apollo.router.operations.response_cache.insert.error` | Errors inserting into cache | {error} |

#### Invalidation

| Metric | Description | Unit |
|--------|-------------|------|
| `apollo.router.operations.response_cache.invalidation.event` | Batch invalidation requests received | {request} |
| `apollo.router.operations.response_cache.invalidation.error` | Invalidation errors | {error} |
| `apollo.router.operations.response_cache.invalidation.entry` | Entries invalidated | {entry} |
| `apollo.router.operations.response_cache.invalidation.request.entry` | Entries per invalidation request | {entry} |
| `apollo.router.operations.response_cache.invalidation.duration` | Invalidation execution time | s |

#### Internal / Redis

| Metric | Description |
|--------|-------------|
| `apollo.router.response_cache.reconnection` | Reconnections to cache storage |
| `apollo.router.response_cache.private_queries.lru.size` | LRU cache size for private queries |
| `apollo.router.cache.redis.clients` | Active Redis clients |
| `apollo.router.cache.redis.command_queue_length` | Commands waiting to send |
| `apollo.router.cache.redis.commands_executed` | Total Redis commands executed |
| `apollo.router.cache.redis.redelivery_count` | Commands retried (connection issues) |
| `apollo.router.cache.redis.errors` | Redis errors by type |

Experimental (may change): `experimental.apollo.router.cache.redis.network_latency_avg`, `latency_avg`, `request_size_avg`, `response_size_avg`.

### Telemetry configuration

```yaml
telemetry:
  instrumentation:
    instruments:
      cache:
        apollo.router.response.cache:
          attributes:
            graphql.type.name: true
            subgraph.name:
              subgraph_name: true
            supergraph.operation.name:
              supergraph_operation_name: string
```

### Trace spans

**`response_cache.lookup`** attributes:
- `kind`: `root` or `entity`
- `subgraph.name`: The subgraph name
- `graphql.type`: The type (or parent type for root fields)
- `cache.status`: `hit`, `partial_hit`, or `miss`
- `debug`, `private`, `contains_private_id`: Booleans
- `cache.key`: The primary cache key

**`response_cache.store`** attributes:
- `kind`: `root` or `entity`
- `subgraph.name`: The subgraph name
- `ttl`: Cache entry TTL
- `batch.size`: Entity batch size

### Log selectors (subgraph service)

| Selector | Values | Description |
|----------|--------|-------------|
| `response_cache` | `hit` or `miss` | Number of cache hits/misses for a subgraph request |
| `response_cache_status` | `hit`, `partial_hit`, `miss`, or `status` | When set to `hit`/`partial_hit`/`miss`: returns a count.  When set to `status`: returns the status string (`hit`, `partial_hit`, or `miss`) as an attribute value. |
| `response_cache_control` | `max_age`, `scope`, `no_store` | Data from the computed `Cache-Control` header |

Example — log uncached subgraph responses:

```yaml
telemetry:
  instrumentation:
    events:
      subgraph:
        response:
          level: info
          condition:
            all:
              - eq:
                  - subgraph_name: true
                  - static: posts
              - eq:
                  - response_cache: hit
                  - 0
```

### Cache debugger

Enable during **development only** with `response_cache.debug: true` and `sandbox.enabled: true`. Open Apollo Sandbox at the router URL to inspect:

- Cache status per entry (hit/miss, created-at, source subgraph)
- `Cache-Control` headers returned by subgraphs
- Entity keys and cache tags
- One-click `curl` commands for invalidation

**Never enable `debug: true` in production** — it exposes internal cache data.
