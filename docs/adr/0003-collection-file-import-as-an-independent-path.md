# Collection File Import Is an Independent Path, Not an Extension of Table File Import

**Collection File Import** (JSON/CSV/Excel into a **MongoDB Collection**) is built as its own GraphQL mutation with a MongoDB-shaped input and its own frontend modal, rather than by extending the SQL-shaped `ImportFileInput` / `ImportTableFile` / `DatabaseImportModal`. Code is reused only at the helper layer — CSV/Excel parsing (`parseImportFile`) and the MongoDB plugin's document operations — not at the GraphQL or modal layer.

The domain model already treats **Collection File Import** as an **Import Method** distinct from **Table File Import** (see `CONTEXT.md`). The existing import flow is relational by construction (column mappings, per-column types, primary keys, generated columns, transactional batch writes) and does not even run for MongoDB: the new-table path calls `AddStorageUnitWithOptions`, which MongoDB inherits from `BasePlugin` as `errors.ErrUnsupported`, and `GetDatabaseMetadata` returns `nil`. MongoDB documents are schemaless and need Extended JSON type fidelity and non-atomic batch writes — concepts that have no place in the SQL input. Forcing both semantics through one input/modal would violate the project's plugin-architecture rule and entangle two unrelated flows.

## Consequences

- MongoDB import is **not atomic**: there is no real transaction (`BasePlugin.WithTransaction` runs the operation with a `nil` tx), so writes land batch-by-batch and a mid-import failure cannot be rolled back. Write errors are skipped-and-reported rather than aborting, and Overwrite mode warns that the pre-clear is irreversible.
- JSON is parsed as documents via the driver's `bson.UnmarshalExtJSON` (the same path the JSON View editor already uses), so it cannot reuse the flat `[][]string` row model; CSV/Excel rows still flow through the `engine.Record` model and the existing `BulkAddRows`.

## Considered Options

- **Extend the unified flow** (add a MongoDB target to `ImportTargetMode`, reuse `ImportFileInput` and `DatabaseImportModal`). Rejected: the input is saturated with relational concepts that are noise for documents, the existing resolver path is unsupported for MongoDB, and it would re-couple flows the domain model keeps separate.
