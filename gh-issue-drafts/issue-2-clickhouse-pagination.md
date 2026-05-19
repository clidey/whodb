## Summary

The generic GORM row-fetch path still delegates pagination to GORM `Limit(...).Offset(...)`. For ClickHouse, we should verify that the generated SQL does not rely on bound parameters for `LIMIT` / `OFFSET`, because that is a known incompatibility in some ClickHouse paths.

## Current evidence

- Generic row fetch applies pagination through GORM:
  - `core/src/plugins/gorm/plugin.go:318`
  - `core/src/plugins/gorm/plugin.go:387`
- The EE bridge path already emits literal integer pagination SQL instead of bound params:
  - `ee/core/src/plugins/bridge/plugin.go:1031`
  - `ee/core/src/plugins/bridge/plugin.go:1061`

## Why this should be tracked

The broad draft issue was overstated. I do not see evidence for a current MySQL-wide problem, but the ClickHouse/GORM path is still a reasonable narrow target because the CE row path does not control the generated pagination SQL directly.

## Scope

- Verify the exact SQL emitted for ClickHouse row pagination in the CE GORM path.
- If the driver/GORM layer binds `LIMIT` / `OFFSET`, override the ClickHouse row path so pagination is emitted as validated integer literals.
- Keep the fix ClickHouse-specific. Do not broaden this into generic SQL string interpolation.

## Acceptance criteria

- ClickHouse row browsing works with pagination on the explore page.
- The fix is confined to the ClickHouse row path or an equivalent plugin-specific extension point.
- No new raw SQL interpolation is introduced for user-controlled predicates or sort fields.

## Notes

- The original draft should be rewritten before filing. This is not a general MySQL issue based on the current code review.
