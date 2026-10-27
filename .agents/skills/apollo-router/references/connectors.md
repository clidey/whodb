# Router Connectors Configuration (v2)

Apollo Connectors are configured in Router v2 under the `connectors` key.

## Core Rules

- Use `connectors` (GA), not `preview_connectors` (early v2 preview).
- Source keys use `<subgraph>.<source>` naming.
- `<source>` must match the `@source(name: "...")` used in your connector schema.

## Minimal Router v2 Example

```yaml
connectors:
  sources:
    my_subgraph.my_api:
      $config:
        api_version: "v2"
        feature_flag: true
```

Use `${env.VAR}` for sensitive values instead of hardcoding secrets in `$config`.

## Migration Note

If you have early v2 preview config:

- Rename `preview_connectors` -> `connectors`
- Move source mapping from `subgraphs.<name>.sources.<source>` to `sources.<subgraph>.<source>`

## Parameter Checklist

- Subgraph name (for example: `products`)
- Source name (for example: `inventory`)
- `$config` keys used by your schema (for example: `api_version`)

## Validate

```bash
router config validate router.yaml
```
