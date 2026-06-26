# WhoDB Data Exploration Context

WhoDB helps users inspect and manipulate database storage units across relational and document databases. This context records product language for database exploration views.

## Language

**MongoDB Collection**:
A MongoDB storage unit made of documents that may not all share the same fields.
_Avoid_: MongoDB table

**MongoDB Document**:
A single record inside a **MongoDB Collection**.
_Avoid_: row, JSON row

**Document Field Order**:
The order in which top-level fields are represented by a **MongoDB Document**.
_Avoid_: alphabetical field order, frontend object key order

**JSON View**:
The MongoDB collection view that shows each **MongoDB Document** as editable JSON.
_Avoid_: card-only view

**Collection Table View**:
A MongoDB collection view that presents documents in a grid using document fields as columns.
_Avoid_: MongoDB table

**Visible Field Set**:
A field list built from the currently visible **MongoDB Documents** and pending document changes for the **Collection Table View**.
_Avoid_: sampled schema, complete schema

**Visible Field Type Hint**:
A type label shown for a field in the **Visible Field Set**.
_Avoid_: schema type, fixed collection type

**Unset Field**:
A field that exists as a column in the **Collection Table View** but is absent from a specific **MongoDB Document**.
_Avoid_: null, empty string

**Editable Scalar Field**:
A top-level document field whose value can be edited directly in a **Collection Table View** cell.
_Avoid_: editable nested field

**Complex Document Field**:
A top-level document field whose value is an object or array and should not be edited inline in a **Collection Table View** cell.
_Avoid_: inline JSON cell

**Field JSON Editor**:
A focused editor for changing a single object or array field from the **Collection Table View**.
_Avoid_: document table mode

**Document JSON Editor**:
The editor for changing an entire **MongoDB Document** as JSON.
_Avoid_: document table mode, field-level editor

**Document Replacement Edit**:
A **Document JSON Editor** edit that replaces the stored **MongoDB Document** shape while preserving the original `_id`.
_Avoid_: field patch, `$set` edit

**Workspace Tab**:
A database exploration surface tied to a query or a storage unit.
_Avoid_: browser tab, panel

**Database Workspace**:
The workbench area where users explore connections, queries, and storage-unit **Workspace Tabs**.
_Avoid_: connections tab, sidebar mode

**Dashboard Workspace**:
The dashboard area where users view and edit analysis dashboards.
_Avoid_: analysis tab, chart sidebar

**Workspace Tab Leave Guard**:
A confirmation step shown before an action would discard unsaved database edits by closing a **Workspace Tab**, closing or refreshing the browser page, or switching from the database workspace to the dashboard.
_Avoid_: route guard, ordinary tab switch blocker, dirty tab blocker

**Sidebar Focus**:
The sidebar tree item that represents the active **Workspace Tab**'s closest database context.
_Avoid_: hover state, keyboard focus

**Sidebar Reveal**:
The behavior that makes the **Sidebar Focus** visible in the sidebar tree.
_Avoid_: expand all, jump to item

**Database Import**:
A database operation where a user brings external database content into the active database connection.
_Avoid_: SQL import as the top-level flow, restore-only flow

**Import Method**:
The kind of external content a **Database Import** uses.
_Avoid_: import mode, export format

**Import Entry Point**:
A UI action that opens **Database Import** from the user's current database context.
_Avoid_: separate import feature, method-specific shortcut

**Import Target Context**:
The target selected for the active **Import Method** in a **Database Import** run.
_Avoid_: fixed entry context, sidebar selection

**SQL Script Import**:
A **Database Import** method where a user provides SQL statements to execute against the active SQL database connection.
_Avoid_: SQL data import, table file import, CSV import, database dump import

**SQL Script Source**:
The user-provided SQL content for a **SQL Script Import**, supplied either by uploading a SQL file or pasting SQL text.
_Avoid_: import method, execution result

**Import Result State**:
The post-execution state of a **Database Import** that tells the user whether the import succeeded or failed.
_Avoid_: toast-only feedback, auto-close

**Table File Import**:
A **Database Import** method where a user maps rows from a CSV or Excel file into one target table.
_Avoid_: SQL script import, database dump import

## Relationships

- A **MongoDB Collection** contains zero or more **MongoDB Documents**.
- A **MongoDB Document** has a **Document Field Order**.
- A **JSON View** displays **MongoDB Documents** in their native document shape.
- A **Collection Table View** presents **MongoDB Documents** as rows while preserving MongoDB's flexible field model.
- A **Visible Field Set** guides the columns shown in a **Collection Table View** but does not represent a complete MongoDB schema.
- A **Visible Field Set** follows **Document Field Order** by adding fields the first time they appear in visible documents.
- A **Visible Field Type Hint** describes a field in the **Visible Field Set** and may be `mixed` when observed or inferred field values use multiple types.
- A **Visible Field Type Hint** does not add fields to the **Visible Field Set**.
- A field outside the **Visible Field Set** is not shown as a **Collection Table View** column.
- An **Unset Field** is distinct from a field whose stored value is `null`.
- Editing an **Unset Field** creates that field on the affected **MongoDB Document**.
- An **Editable Scalar Field** can be edited inline in the **Collection Table View**.
- Editing an **Editable Scalar Field** preserves the existing field type when the field already exists, except when the input is a complete, valid, unquoted JSON object or array.
- A **Complex Document Field** in the **Collection Table View** can open a **Field JSON Editor**.
- A **Field JSON Editor** accepts any valid JSON value, even when that changes an object or array field into a scalar or `null`.
- A **MongoDB Document** is edited through a **Document JSON Editor**.
- A **Document JSON Editor** creates a **Document Replacement Edit** when the user changes values, adds fields, or removes fields.
- A **Document Replacement Edit** persists the **Document Field Order** authored in the JSON editor, except MongoDB's `_id` field remains first and immutable.
- Changing only **Document Field Order** is not a submitable database edit.
- A **Complex Document Field** is not edited through a separate field-level interaction inside the **Document JSON Editor**.
- The **Database Workspace** contains zero or more **Workspace Tabs**.
- The **Dashboard Workspace** is separate from the **Database Workspace**.
- A **Workspace Tab** has zero or one **Sidebar Focus**.
- A storage-unit **Workspace Tab** focuses its table, view, collection, or Redis key.
- A query **Workspace Tab** focuses the schema, database, or connection, whichever is most specific.
- A **Workspace Tab Leave Guard** protects unsaved edits in SQL table and MongoDB collection **Workspace Tabs**.
- A **Workspace Tab Leave Guard** does not apply to switching between open **Workspace Tabs**.
- A **Workspace Tab Leave Guard** applies when a protected **Workspace Tab** would be closed, the browser page would close or refresh, or the **Database Workspace** would be replaced by the **Dashboard Workspace**.
- A **Workspace Tab Leave Guard** does not submit database edits; users submit edits through the storage unit's normal review and apply flow before leaving.
- A **Workspace Tab Leave Guard** protects unsaved database edits, not unsaved query text.
- A **Workspace Tab Leave Guard** protects storage-unit data edits, not SQL table-structure edits made in a modal.
- A **Workspace Tab Leave Guard** is triggered by unsaved database edits only, even when a **Workspace Tab** also has unsaved query text.
- A **Workspace Tab** indicates when it has unsaved database edits, even though ordinary switching between open **Workspace Tabs** remains allowed.
- A protected **Workspace Tab** owns how its unsaved database edits are discarded; the **Workspace Tab Leave Guard** only coordinates the confirmed discard before continuing the leave action.
- A **Workspace Tab Leave Guard** shows one summary confirmation when one action would close multiple protected **Workspace Tabs** or switch away from the database workspace while multiple protected **Workspace Tabs** have unsaved edits.
- Confirming a **Workspace Tab Leave Guard** discards the protected **Workspace Tab**'s unsaved database edits before the leave action continues.
- A **Workspace Tab Leave Guard** does not apply when returning from the **Dashboard Workspace** to the **Database Workspace**.
- A **Sidebar Reveal** expands collapsed ancestors of the **Sidebar Focus** and scrolls the focus into view.
- A **Database Import** has exactly one **Import Method** for a single import run.
- A **Database Import** can start from multiple **Import Entry Points** before the user chooses an **Import Method**.
- An **Import Entry Point** can appear in supported import contexts even when that context does not currently have a comparable export action.
- When a comparable export action exists, the **Import Entry Point** should appear near it.
- An **Import Entry Point** may preselect context such as a table, but it should not hide valid **Import Methods** only because the entry came from that context.
- An **Import Entry Point** provides the initial **Import Target Context**.
- A user can change the **Import Target Context** inside **Database Import** before execution when the active **Import Method** uses that target.
- A **SQL Script Import** is an **Import Method** that executes SQL statements rather than mapping file rows to table columns.
- A **SQL Script Import** uses database context; schema and table context from an **Import Entry Point** do not change SQL execution target.
- A **SQL Script Import** lets users change the target database before execution.
- A **SQL Script Import** does not use table context.
- A **SQL Script Import** has exactly one **SQL Script Source**.
- A **SQL Script Source** can be an uploaded SQL file or pasted SQL text.
- A **SQL Script Import** shows an uploaded SQL file as a read-only review before execution.
- A user can convert an uploaded SQL file source into editable pasted SQL text; after conversion, the **SQL Script Source** is pasted SQL text, not the uploaded file.
- A **SQL Script Import** review is the confirmation step for execution; it does not require a second confirmation modal or a required acknowledgement checkbox.
- A **SQL Script Import** submits the script as one backend import operation; the frontend does not split the script into individual statements.
- A **Database Import** stays open after execution and shows an **Import Result State**.
- A successful **Database Import** refreshes database navigation state and any relevant active table context.
- A **Table File Import** maps CSV or Excel rows to columns in exactly one target table.
- A **SQL Script Import** and a **Table File Import** are separate **Import Methods** under **Database Import**.
- A disabled **Table File Import** option is shown from every first-release SQL **Import Entry Point**, so the **Database Import** flow communicates multiple import methods consistently.
- A ClickHouse **SQL Script Import** accepts one SQL statement; multi-statement ClickHouse scripts are outside the first supported flow.

## Example Dialogue

> **Dev:** "Should MongoDB open in the table by default?"
> **Domain expert:** "Yes. Open MongoDB collections in the **Collection Table View** by default because users expect a grid for browsing. Keep the **JSON View** available as a switchable document-focused view."
>
> **Dev:** "When users switch between **Workspace Tabs**, should the sidebar keep the last clicked tree item?"
> **Domain expert:** "No. The sidebar should show the **Sidebar Focus** for the active **Workspace Tab**."
>
> **Dev:** "If that focus is hidden under a collapsed folder or outside the visible sidebar area, should we leave the tree as-is?"
> **Domain expert:** "No. Use **Sidebar Reveal** so the focused item is visible without expanding unrelated branches."
>
> **Dev:** "If a user edits rows in one **Workspace Tab** and clicks another open tab, should we block the switch?"
> **Domain expert:** "No. Keep the edits in their original **Workspace Tab** and allow ordinary tab switching. Use the **Workspace Tab Leave Guard** only when edits would be discarded by closing the tab, closing or refreshing the browser page, or switching to the dashboard."
>
> **Dev:** "Should uploading a `.sql` file use the same flow as uploading a CSV file?"
> **Domain expert:** "They belong to the same **Database Import** entry point, but use different **Import Methods**. A `.sql` file is a **SQL Script Import** because the user is providing statements to execute. A CSV or Excel file is a **Table File Import** because the user is mapping rows into one table."
>
> **Dev:** "Should Import only be available from the database menu?"
> **Domain expert:** "No. Use multiple **Import Entry Points**. When a SQL database or table context has an export action, place an import action beside it and open the same **Database Import** flow with that context preselected."
>
> **Dev:** "If Import opens from a table, should SQL import disappear?"
> **Domain expert:** "No. A table **Import Entry Point** can preselect that table for table-aware methods, but **SQL Script Import** remains available because a SQL script can still be the chosen **Import Method**."
>
> **Dev:** "If Import opens from a table, is that target fixed?"
> **Domain expert:** "No. The entry point provides initial context, but the active **Import Method** decides which parts matter. **SQL Script Import** uses database context only, while schema and table context belong to table-aware methods such as **Table File Import**."
>
> **Dev:** "Can users change where a **SQL Script Import** runs?"
> **Domain expert:** "They can change the target database before execution. They cannot change schema or table target for **SQL Script Import**."
>
> **Dev:** "Should the first Import modal hide CSV and Excel until they work?"
> **Domain expert:** "No. Show **Table File Import** as a disabled **Import Method** from every first-release SQL **Import Entry Point** so users understand Import supports multiple methods, while only **SQL Script Import** is enabled."
>
> **Dev:** "Should a SQL file import execute immediately after upload?"
> **Domain expert:** "No. A **SQL Script Import** should show the uploaded script first, then execute only after the user confirms the import."
>
> **Dev:** "After users review the SQL, should we show another confirmation dialog before execution?"
> **Domain expert:** "No. The **SQL Script Import** review is the confirmation step. Use a clear execution button and inline risk message instead of another modal or required checkbox."
>
> **Dev:** "Can users edit the SQL after uploading a file?"
> **Domain expert:** "Not directly in the uploaded file review. They can convert the uploaded file into editable pasted SQL text, making the edited text the active **SQL Script Source**."
>
> **Dev:** "Should users be able to paste SQL instead of uploading a `.sql` file?"
> **Domain expert:** "Yes. A **SQL Script Import** can use either uploaded file content or pasted SQL text as its **SQL Script Source**, but a single import run uses only one source."
>
> **Dev:** "Should the frontend split a SQL script into statements before import?"
> **Domain expert:** "No. A **SQL Script Import** submits the **SQL Script Source** as one backend import operation so SQL parsing stays out of the frontend."
>
> **Dev:** "Should the Import modal close automatically after success?"
> **Domain expert:** "No. Keep the **Database Import** open and show an **Import Result State** so users can see success or failure before leaving."

## Flagged Ambiguities

### Terminology

- "table view" in MongoDB means **Collection Table View**, not a relational database table.
- The document editor should be a **Document JSON Editor**, not a table view, field list, or field-level editor.
- "focus" in the sidebar means **Sidebar Focus**, not hover state or keyboard focus.
- "auto expand" in the sidebar means **Sidebar Reveal**, not expanding every folder in the tree.
- "leave protection" means **Workspace Tab Leave Guard**, not only the browser's refresh or close-page warning.
- "SQL import" means the **SQL Script Import** method inside **Database Import**, not the top-level import flow.
- "Import entry" means an **Import Entry Point** into the same **Database Import** flow, not a separate method-specific import feature.
- "Table import entry" means an **Import Entry Point** opened from table context, not a restriction to **Table File Import** only.
- "Import target" means the chosen **Import Target Context** in the modal, not necessarily the sidebar node that opened the modal.
- "SQL import target" means database context, not schema or table context.
- "Postgres schema context" in **SQL Script Import** means entry metadata only; the first enabled SQL flow does not provide schema switching or implicitly apply schema context.
- "Editing an uploaded SQL file" means converting the file review into pasted SQL text; it does not mutate the uploaded file or keep the file as the active source.
- "SQL import confirmation" means pressing the execution button from the reviewed SQL state, not completing an additional modal or checkbox.
- "ClickHouse SQL import" means a single-statement **SQL Script Import** unless multi-statement support is explicitly added later.
- "original fields" in MongoDB means **Document Field Order**, not alphabetical field order.
- The **Collection Table View** is the default MongoDB collection view.
- The **Collection Table View** should not infer fields from an extra collection sample.
- Type information in the **Collection Table View** is a **Visible Field Type Hint**, not a complete schema guarantee.
- The **Collection Table View** shows fields from the current visible documents and pending document changes.
- The **Collection Table View** should preserve first-seen **Document Field Order** rather than sorting fields alphabetically.
- The **Collection Table View** supports sorting and filtering on top-level document fields.

### MongoDB Table Editing

- MongoDB inline editing is limited to **Editable Scalar Fields**; object and array cells in the **Collection Table View** open a **Field JSON Editor**.
- A **Field JSON Editor** validates JSON syntax only. It does not force the edited value to remain an object or array.
- Empty input or clearing an existing field is not field deletion; field deletion must be a distinct action.
- Editing a `null` or **Unset Field** in the **Collection Table View** creates a string value unless the input is a complete, valid, unquoted JSON object or array.
- Typing a complete, valid, unquoted JSON object or array into any **Editable Scalar Field** changes that field into a **Complex Document Field**.
- Editing an existing field value in the **Collection Table View** must not move that field in the **Document Field Order**; newly created fields from field-level edits are appended to the document's visible field order.
- Editing through the **Document JSON Editor** treats field additions, field removals, and value changes as the authored **Document Replacement Edit**. **Document Field Order** changes are saved only as part of those content changes.

### MongoDB View State

- Switching between **Collection Table View** and **JSON View** preserves pending document changes.
- Pending changes from **Collection Table View** and **JSON View** share the same document-level preview and submission flow.
