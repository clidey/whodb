# PRD: Database Import Entry Points and SQL Script Import

## Problem Statement

WhoDB users can export data from SQL database and table contexts, but they do not have a matching **Database Import** experience nearby. Users who want to bring SQL scripts into Postgres, MySQL, or ClickHouse need a clear import entry beside export actions, a way to choose an **Import Method**, and a safe review step before executing high-risk SQL.

The first release should establish the product model for **Database Import** without pretending import is only a SQL feature. **SQL Script Import** is the first enabled **Import Method**. **Table File Import** for CSV / Excel should be visible but disabled so the UI communicates the intended future shape.

## Solution

Add multiple **Import Entry Points** for supported SQL database contexts. Existing export actions are placement anchors, not strict prerequisites: when export is present, Import should appear nearby; when a supported import context has no export action, Import can still appear if it helps users start **Database Import** from their current work area. Each entry opens the same **Database Import** modal with the current database context preselected.

The modal presents import methods:

- **SQL file**: enabled in the first release.
- **CSV / Excel**: visible but disabled.

The **SQL Script Import** method supports two **SQL Script Source** choices: uploading a `.sql` file or pasting SQL text. Users review the script in the modal, then explicitly execute the import. The frontend submits the script as one backend import operation; it does not split SQL into individual statements.

After execution, the modal remains open and shows an **Import Result State**. A successful import refreshes database navigation state and, when the entry came from an active table context, refreshes relevant table data.

## User Stories

1. As a SQL database user, I want an Import action in database contexts, so that importing is available from the work area I am already using.
2. As a SQL table user, I want an Import action beside Export Data, so that I can bring data or scripts in from the same context where I export.
3. As a SQL table toolbar user, I want Import next to Export, so that repeated data workflows are discoverable without right-clicking.
4. As a Postgres user, I want Database Import to be available from database and table contexts, so that I can import SQL into my current work area.
5. As a MySQL user, I want Database Import to be available from database and table contexts, so that I can import SQL scripts without leaving the database workspace.
6. As a ClickHouse user, I want Database Import to be available from database and table contexts, so that I can import supported SQL statements from the same places I export.
7. As a user opening Import from a table, I want SQL Script Import to remain available, so that table context does not hide valid import methods.
8. As a user opening Import from a table, I want the table context preserved for future table-aware methods, so that CSV / Excel import can later default to that table.
9. As a user opening Import from a database, I want the database preselected, so that the modal starts in the context I chose.
10. As a user importing SQL, I want to change the execution database before running the import, so that I can correct the target without closing the modal.
11. As a Postgres user opening Import from a schema or table context, I want that context preserved as entry metadata, so that future table-aware import methods can use it without changing first-release SQL execution semantics.
12. As a SQL import user, I want to choose between uploading a `.sql` file and pasting SQL, so that I can use the source format I already have.
13. As a SQL import user, I want a `.sql` upload control, so that I can import scripts saved on disk.
14. As a SQL import user, I want a paste area for SQL text, so that I can import scripts copied from another tool.
15. As a SQL import user, I want the UI to prevent using both upload and paste sources at once, so that the import source is unambiguous.
16. As a SQL import user, I want to review the SQL before execution, so that I can catch dangerous or incorrect statements.
17. As a SQL import user reviewing an uploaded file, I want to convert it into editable SQL text, so that I can make a small correction without re-uploading a new file.
18. As a SQL import user, I want the reviewed SQL state to be the confirmation step, so that I can execute without another blocking confirmation dialog.
19. As a SQL import user, I want the Import button disabled until a script source exists, so that I do not submit an empty import.
20. As a SQL import user, I want execution progress feedback, so that I know the import is running.
21. As a SQL import user, I want the modal to stay open after success, so that I can see whether the import completed.
22. As a SQL import user, I want clear failure feedback that preserves my script source, so that I can edit or retry.
23. As a SQL import user who converted an uploaded file into text, I want retry to preserve the edited SQL text, so that my corrections are not lost.
24. As a SQL import user, I want database navigation to refresh after success, so that newly created or changed objects appear.
25. As a table-context user, I want relevant table data to refresh after success, so that visible table contents reflect imported changes.
26. As a ClickHouse user, I want multi-statement scripts to fail with a clear unsupported message, so that I understand the first supported flow is single-statement SQL.
27. As a Postgres or MySQL user, I want multi-statement scripts submitted as one import operation, so that SQL parsing remains backend-owned and scripts with comments or function bodies are not broken by frontend splitting.
28. As a user planning future imports, I want CSV / Excel visible in the Import modal, so that I understand Database Import is not only for SQL.
29. As a user seeing CSV / Excel, I want it clearly disabled, so that I do not attempt an unavailable import method.
30. As a keyboard user, I want every Import control reachable and labeled, so that I can use the modal without a mouse.
31. As a screen reader user, I want disabled import methods announced as unavailable, so that the modal state is understandable.
32. As a localized app user, I want all Import UI text localized, so that the feature is consistent with the rest of DataFlow.
33. As a developer maintaining import behavior, I want a single import modal and shared import state model, so that multiple entry points do not drift into separate implementations.
34. As a developer maintaining GraphQL behavior, I want SQL import to use the GraphQL API, so that new API functionality follows the project GraphQL-first rule.
35. As a developer maintaining SQL safety, I want target context handling implemented without interpolating user input into SQL unsafely, so that schema context support does not introduce SQL injection risk.

## Implementation Decisions

- Add a shared **Database Import** modal opened by multiple **Import Entry Points**.
- Add Import actions for supported SQL database contexts: database context menus, Postgres schema context menus, table context menus, and SQL table toolbars. Place them beside existing SQL export actions where those actions exist, but do not make export availability a prerequisite for import placement.
- Keep the first release scoped to SQL database types: Postgres, MySQL, and ClickHouse.
- Do not add Import actions for view, MongoDB collection, or Redis key contexts in this PRD.
- The Import modal contains a method selector. The enabled first-release method is **SQL Script Import**.
- The **Table File Import** option is shown as disabled and labeled as unavailable / coming soon from every first-release SQL import entry point.
- The modal should use stable, accessible controls: selectable method options, segmented or tabbed source selection, labeled upload/paste inputs, loading state, and result feedback.
- **SQL Script Import** supports exactly one **SQL Script Source** at a time: uploaded SQL file or pasted SQL text.
- Uploaded SQL is shown in a read-only preview before execution.
- Uploaded SQL can be converted into editable pasted SQL text. After conversion, the active **SQL Script Source** is pasted SQL text and execution submits `Script`, not `File`.
- Pasted SQL is editable before execution.
- SQL script execution happens only when the user presses Import.
- The reviewed SQL state is the execution confirmation step. Do not add a second confirmation modal or required acknowledgement checkbox in the first release.
- The execution button should use explicit execution language and sit near concise risk feedback explaining that the current SQL will run against the selected database.
- The frontend submits the script as one backend import operation. It does not split SQL statements.
- For SQL import, table context is not an execution target. Table context may be carried only as entry metadata and future default context for table-aware methods.
- SQL import execution target is database context only in the first release. Schema and table context may be carried as entry metadata, but the SQL execution target does not change schema and the modal does not provide schema switching.
- The SQL import modal should include a labeled target database selector. It defaults to the entry point database and can be changed before execution.
- The SQL import modal should not include schema or table target selectors in the first release.
- ClickHouse SQL import supports one SQL statement in the first release. Multi-statement ClickHouse scripts return the existing unsupported multi-statement import error.
- Postgres and MySQL SQL imports use the backend's existing multi-statement execution path.
- The backend GraphQL import mutation remains the API surface for SQL Script Import.
- Do not extend the first-release SQL Script Import GraphQL input for Postgres schema execution context. Users can schema-qualify SQL or include their own SQL context statements when needed.
- Import result should return enough structured status for the modal to show success or failure. The first release can use the existing status/detail shape if it is sufficient for localized UI.
- All user-facing Import strings must be added to localization messages with no fallback strings.
- GraphQL operations should be added for the frontend import modal and generated client types refreshed.
- A small deep module should own import modal state: selected method, source kind, source content/file metadata, target database/schema, execution state, and result state.
- A small deep module should own SQL import request construction and client-side validation: exactly one source, non-empty script, accepted file type, and disabled Import button state.
- Existing sidebar and table refresh mechanisms should be reused after successful import.
- The modal stays open after execution and transitions into **Import Result State**. Success offers a Done close path; failure preserves source content for retry.

## Testing Decisions

- Good tests should assert external behavior: visible entry points, enabled/disabled methods, button state, GraphQL calls, result feedback, and refresh triggers. Avoid testing internal hook state directly unless extracted as a pure deep module.
- Add context menu tests showing Import appears beside Export for supported SQL database and table contexts.
- Add table toolbar tests showing Import appears beside Export for supported SQL table contexts.
- Add modal behavior tests for method selection, disabled CSV / Excel method, source-kind switching, upload preview, paste input, disabled Import button for empty source, success state, and failure state.
- Add modal behavior tests showing uploaded SQL preview is read-only and that converting it to editable text changes the active source to pasted SQL text.
- Add modal behavior tests showing SQL import executes from the review state without opening a second confirmation modal.
- Add client request-construction tests for the SQL import deep module: uploaded file source, pasted script source, converted file-to-text source, and invalid dual-source state.
- Add GraphQL resolver tests around SQL import behavior if backend schema/context changes are introduced.
- Add tests for ClickHouse multi-statement SQL import returning the unsupported multi-statement detail rather than a generic failure.
- Add localization coverage by ensuring new message keys are used in rendered UI and no hardcoded user-facing strings are introduced.
- Reuse existing frontend testing patterns for context menu behavior and modal rendering.
- Reuse existing backend import helper and mutation resolver test patterns for GraphQL import behavior.
- Verification for implementation should include DataFlow typecheck, build, relevant frontend tests, backend build, and any changed backend tests.

## Out of Scope

- Implementing enabled CSV / Excel **Table File Import** UI.
- Implementing dump import.
- Adding MongoDB, Redis, Elasticsearch, or SQLite import entry points.
- Adding REST import endpoints.
- Building a full SQL parser in the frontend.
- Splitting SQL scripts into statements in the frontend.
- Making ClickHouse multi-statement import work in the first release.
- Table-targeted SQL execution semantics.
- Schema switching or implicit schema execution context for **SQL Script Import**.
- Automatically closing the modal after import success.
- Large import job history, background job tracking, resumable uploads, or cancellation.
- Drag-and-drop folder imports or compressed archives.

## Further Notes

- The current product glossary defines **Database Import**, **Import Method**, **Import Entry Point**, **Import Target Context**, **SQL Script Import**, **SQL Script Source**, **Import Result State**, and **Table File Import**. Implementation language should use these terms.
- Existing backend work already includes GraphQL import shapes for SQL script import and table file import. The first release should reuse and refine those capabilities rather than adding a parallel API.
- Existing table-file parsing and import backend behavior should not force the first UI release to enable CSV / Excel. Showing it disabled is an intentional product cue, not a partial implementation.
- The UI should stay quiet and work-focused, matching the existing database management surface rather than becoming a marketing-style import wizard.
