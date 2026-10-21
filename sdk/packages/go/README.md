# whodb (Go SDK)

Official Go SDK for the [WhoDB](https://whodb.com) hosted platform — your
ontology, datasets, sources, transforms, functions, and org administration as
in-code function APIs.

## Install

```bash
go get github.com/clidey/whodb/sdk/packages/go
```

Requires Go ≥ 1.27.3.

## Quickstart

```go
import whodb "github.com/clidey/whodb/sdk/packages/go"

// Production: API key (create one in org settings → API keys).
client, err := whodb.New(whodb.Config{APIKey: os.Getenv("WHODB_API_KEY")})

// Local development: zero config — reuses your `whodb login` session.
// client, err := whodb.New(whodb.Config{})

users := client.Ontology("User")

user, err := users.Get(ctx, "u_123")
rows, err := users.List(ctx, whodb.ListOptions{
    Where:    map[string]any{"status": map[string]any{"eq": "active"}},
    PageSize: 100,
})
err = users.Create(ctx, map[string]any{"email": "a@b.co"})
_, err = users.CreateMany(ctx, rows, "import-42") // idempotency key
err = users.Update(ctx, "u_123", map[string]any{"plan": "pro"})
orders, err := users.FollowLink(ctx, "u_123", "orders", 50, 0)

// Behavior-managed workflow:
order := client.Ontology("Order")
capabilities, err := order.Capabilities(ctx, "order_123")
version := int(capabilities["recordVersion"].(float64))
_, err = order.PreviewAction(ctx, "approve", "order_123", map[string]any{"note": "Looks good"})
_, err = order.Action(ctx, "approve", "order_123", map[string]any{"note": "Looks good"}, whodb.ActionOptions{
    ExpectedVersion: &version,
    IdempotencyKey:  "approve-order-123",
})
history, err := order.ActionExecutions(ctx, "order_123", 50)

// Iterate everything, page by page:
err = users.Pages(ctx, whodb.ListOptions{PageSize: 500}, func(rows []whodb.Row) bool {
    fmt.Println(len(rows))
    return true // false stops iteration
})

// Datasets and sources:
kpis, err := client.Dataset("weekly_kpis").Rows(ctx, whodb.DatasetRowsOptions{PageSize: 100})
raw, err := client.Source("src_...").Rows(ctx, map[string]any{"Kind": "Table", "Locator": "events"}, whodb.SourceRowsOptions{})

// Transforms: run a pipeline and wait for it to finish.
rollup := client.Transform("nightly-rollup")
run, err := rollup.Run(ctx)
finished, err := rollup.Wait(ctx, run["id"].(string), whodb.WaitOptions{Timeout: 5 * time.Minute})

// Functions: synchronous execute, or a tracked run.
normalize := client.Function("normalize")
_, err = normalize.Promote(ctx, "v1") // snapshot + set active production version
result, err := normalize.Execute(ctx, map[string]any{"name": "Ada"}, nil)
queued, err := normalize.StartRun(ctx, map[string]any{"batch": 7}, nil)
_, err = normalize.Wait(ctx, queued["id"].(string), whodb.WaitOptions{})

// Org administration (requires org manage rights):
sa, err := client.ServiceAccounts().Create(ctx, "ci-deployer", "")
key, err := client.ServiceAccounts().CreateKey(ctx, sa["id"].(string), "") // key["rawKey"] is shown once
_, err = client.Sharing().Share(ctx, "ont_...", "organization", nil)
```

## Authentication

Credential precedence: `Config` fields (`APIKey` / `Token` / `TokenFunc` /
`Credentials`) → `WHODB_API_KEY` env var → the `whodb` CLI's stored login.
Workspace (`Org` / `Project`) is optional with an API key; set `Project`
when the key has access to more than one project.

Inside a WhoDB Function, `whodb.New(whodb.Config{})` auto-detects the runtime
and needs no configuration at all.

## Errors

Sentinel errors for `errors.Is`: `ErrAuth`, `ErrNotFound`, `ErrValidation`,
`ErrVersion` (SDK outdated for the platform API — upgrade the module),
`ErrCliCredentials`, `ErrTransportCapability`; plus `*PlatformError` (via
`errors.As`) carrying the platform's error code.

## License

Apache-2.0
