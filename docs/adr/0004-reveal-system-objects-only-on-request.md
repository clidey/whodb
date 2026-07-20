# Reveal System Objects Only on Request, Classified from Catalog Structure

Sealos-provisioned PostgreSQL clusters ship a set of operator and extension objects in the `public` schema of the default logical database (Spilo-style `post_init` provisioning: `postgres_log*` file_fdw tables, `failed_authentication_*` views, `pg_stat_*`/`pg_auth_mon` extension views), and they interleave with user tables in the sidebar. DataFlow already hides system schemas (`pg_catalog`, `information_schema`, …) behind the per-database "Show system objects" context-menu toggle, but objects inside user schemas carried no classification at all. We decided these are **System Objects** (see `CONTEXT.md`): the sidebar omits them from table and view listings by default, and the existing per-database toggle reveals both layers — system schemas and **System Objects** inside user schemas — as one concept. When revealed, **System Objects** stay inline in the normal listing with muted styling rather than in a separate group, and a **Database Export** follows the same visibility rule: hidden objects are excluded from the export plan unless revealed.

Classification is computed where catalog structure is visible — the postgres plugin — from structural evidence, not names: extension membership (`pg_depend` with `deptype='e'`), foreign tables on a `file_fdw` server, and regular tables whose inheritance children all match; the only name-based rule is Spilo's pinned `failed_authentication_[0-7]` views (`relkind='v'`). The fingerprint query runs separately from the table-listing query and fails open: any classification error yields "no System Objects", never a broken listing. The label travels as a `whodb:system-object` StorageUnit attribute — a transport-only marker — and the connection store promotes it to a typed `system` flag at the same mapping point where `Type` is already promoted, so everything past the store consumes a typed contract.

## Consequences

- Scope is PostgreSQL only; other engines keep their existing schema-level behavior, and their storage units are simply never marked.
- The toggle now refreshes the whole subtree of the database node, because for PostgreSQL it governs table listings two levels below the toggled node; previously it only re-fetched direct children.
- A misclassified object is hidden by default rather than merely regrouped; the structural fingerprints and the toggle's recovery path make that acceptable.
- Revealed `postgres_log_*` tables are file_fdw tables and surface real read errors when the underlying log file is absent; excluding them from default exports also removes a source of partial-failure noise.

## Considered Options

- **Hiding System Objects unconditionally at the plugin layer** (the upstream precedent for MongoDB `system.*` and Elasticsearch dot-indices) was rejected because `pg_stat_statements` is genuinely useful for users diagnosing slow databases, and objects filtered from a shared user schema deserve a recovery path.
- **A second, object-level toggle** was rejected because "system objects" is one concept to the user; splitting the control would expose the implementation layering in the UI.
- **Classifying in the frontend from name lists** was rejected because the frontend only sees object names, and name matching is exactly what can misfile user tables.
- **A first-class GraphQL field** (`StorageUnit.IsSystem`) was rejected to keep the schema diff against upstream nil; the attribute-to-flag promotion in the connection store already has a precedent (`Type`) and gives consumers the same typed contract.
- **Rewriting the main table-listing query with catalog joins** was rejected because a fingerprint incompatibility must not break the core listing; the separate fail-open query keeps the blast radius at zero.
