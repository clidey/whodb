# whodb (Go SDK)

Official Go SDK for the [WhoDB](https://whodb.com) hosted platform — your
ontology as in-code function APIs.

## Install

```bash
go get github.com/clidey/whodb/sdk/packages/go
```

Requires Go ≥ 1.22.

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

// Iterate everything, page by page:
err = users.Pages(ctx, whodb.ListOptions{PageSize: 500}, func(rows []whodb.Row) bool {
    fmt.Println(len(rows))
    return true // false stops iteration
})
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
