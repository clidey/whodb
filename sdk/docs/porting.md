# Porting the WhoDB SDK to a New Language

The contract, learned from the TypeScript → Python port. Read this before
starting Go/Rust/Java. Languages differ in ergonomics, never in behavior —
the shared conformance fixtures are the acceptance test.

## 1. Inputs (all committed under `sdk/spec/`)

- `platform-schema.graphql` — merged SDL, full type detail
- `platform-manifest.json` — the curated public operation list (allowlist);
  carries per-op versioning-policy fields (`deprecated`, `sunsetAt`,
  `behaviorChanged`, `note`)
- `surface.yaml` — operation → facade-method mapping; `excluded:` ops must
  hard-error in the generator; `internal: true` domains are init-time only
- `fixtures/*.json` — the portable behavior contract

## 2. Generated wire core

Add a renderer at `tools/render/<language>.mjs` exporting
`render<Language>(ir) → [[relPath, content], ...]`, and register it in the
`renderers` list in `tools/generate-core.mjs`. The IR per operation:
`{domain, method, internal, name, kind, args[{name,type,required,list}],
returns, returnTypeName, document, autofill, rename, paginated,
deprecated, sunsetAt, behaviorChanged, note}` plus `typeClosure` (GraphQL
type names), `manifestHash`, `hydrationRules`, `protocolVersion`.

Emit five artifacts (committed, headed "GENERATED — DO NOT EDIT"):
- **operations**: per-op document constants + request builders. Make them
  TRANSPORT-AGNOSTIC (build request / parse response separately) so sync and
  async clients share all generated code.
- **types**: language-native shapes for the type closure. Gotcha: GraphQL
  field names can be language keywords (`as` broke Python's TypedDict class
  syntax — use a keyword-safe construction).
- **manifest**: embedded per-op policy map + `manifestHash` +
  `manifestProtocolVersion`.
- **hydration**: the column-type → kind rules verbatim from
  `tools/hydration-rules.json`.
- The TS renderer also emits a `surface` map; optional elsewhere.

`generate-core.mjs --check` must pass for your language (byte-identical
re-render) — it runs in CE CI.

## 3. Handwritten layer (the facade)

Mirror the TypeScript package file-for-file in idiomatic naming:

- **Transport seam**: one interface `execute(operation, document, variables)
  → GraphQL data object`. Implementations: HTTP (POST `{host}/api/query`,
  bearer auth, `X-Whodb-Org-Id`/`X-Whodb-Project-Id`, `User-Agent:
  clidey-whodb-<lang>/<version>`, one 5xx retry, one 401 refresh-retry) and
  IPC (functions runtime; ontology ops only, others raise the
  transport-capability error; entity IDs resolve to apiNames via a cached
  `/entities` call — see the Python `_transport_ipc.py` for the endpoint map
  and request shapes).
- **Credentials**: static key/token, callback, CLI helper (exec
  `whodb auth print-token --format json`, argv list not shell, 15s timeout,
  cache until `expires_at - 60s`; missing binary → the "install it or set
  WHODB_API_KEY" error). Precedence: explicit → `WHODB_API_KEY` → CLI.
- **Errors**: `WhoDBError` base; `AuthError` (UNAUTHENTICATED/FORBIDDEN),
  `NotFoundError`, `ValidationError` (BAD_USER_INPUT/
  GRAPHQL_VALIDATION_FAILED), `PlatformError{code}` (everything else),
  `WhoDBVersionError`, `CliCredentialsError`, `TransportCapabilityError`.
- **Version check**: deprecated/behaviorChanged → warn once per process;
  server "unknown field/type" rejections → `WhoDBVersionError` naming the
  SDK version and package.
- **Hydration**: normalize `{columns: [name], rows, total}` (DatasetQuery)
  and `{Columns: [{Name,Type}], Rows, TotalCount}` (RowsResult); coerce via
  the rules; ontology property `dataType` overrides the wire column type;
  non-string values pass through (IPC delivers native types).
- **Client**: workspace resolution (slug→ID via MyOrganizations/Projects,
  UUIDs pass through; API keys may omit org/project — discover via
  `MyWorkspace`, hard-error when projectId comes back null); custom
  transports skip resolution; IPC autodetect on `WHODB_IPC_TOKEN`.
- **Facades**: ontology (entity metadata cached per handle; `get` = pk-eq
  query with pageSize 1; `list`/`query` go through OntologyQuery, NOT
  OntologyRows; writes convert values to RecordInput pairs, JSON-encoding
  objects and lowercasing booleans), dataset, source, pagination
  (short page terminates).

## 4. Conformance (the gate)

Ship a fixture runner at `packages/<language>/scripts/conformance.<ext>`
speaking JSON lines on stdin/stdout: read one fixture per line, run it
against the real client with a mock transport, emit
`{"name", "pass", "reason"?}`. Semantics:

- Mock transport replays `transcript[]`, asserting `expectRequest`
  (`operation`, `variables` exact, `variablesContain`, `inputContains`);
  a step with `response.errors` raises the mapped error.
- `call`: `{domain, handle, method, args, collect?}`. Method names are
  TS-camelCase in fixtures — map to your language's convention. Trailing
  option objects map to your language's options idiom.
- `collect: "pages"` walks pagination; a bare list call compares the first
  page's rows.
- Canonical results: datetimes serialize as `@date:<ISO-8601 with Z>`;
  compare as JSON.

Register the run command in `RUNNERS` in `tools/conformance-runner.mjs`.
`--lang <language>` must report 7/7 (or the current fixture count) before the
port is real.

## 5. Package + release

- Version `0.0.0` in the manifest; teach `tools/sync-versions.mjs` to read
  and stamp it (`--set`/`--check`), and add a stamp step for the version
  constant that feeds User-Agent.
- Wire tests into `pnpm -r test` (a private harness package.json is fine for
  non-JS languages — see packages/python/package.json).
- Add a publish job modeled on the `deploy-sdk-pypi` job in `release-ce.yml`
  (inline it there, not a reusable workflow, if the registry validates
  attestations against the top-level workflow — PyPI does; trusted publishing
  where the registry supports it) and gate it on `deploy-sdk` in
  `release-ce.yml`.
- Vet every new runtime dependency with deptrust before pinning; prefer zero
  or one dependency (Python's single dep is httpx, chosen for sync+async+UDS). Package names: npm @clidey/whodb-sdk, PyPI whodb-sdk (import name stays whodb).
