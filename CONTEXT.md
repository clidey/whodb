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

## Relationships

- A **MongoDB Collection** contains zero or more **MongoDB Documents**.
- A **MongoDB Document** has a **Document Field Order**.
- A **JSON View** displays **MongoDB Documents** in their native document shape.
- A **Collection Table View** presents **MongoDB Documents** as rows while preserving MongoDB's flexible field model.
- A **Visible Field Set** guides the columns shown in a **Collection Table View** but does not represent a complete MongoDB schema.
- A **Visible Field Set** follows **Document Field Order** by adding fields the first time they appear in visible documents.
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

## Flagged Ambiguities

### Terminology

- "table view" in MongoDB means **Collection Table View**, not a relational database table.
- The document editor should be a **Document JSON Editor**, not a table view, field list, or field-level editor.
- "focus" in the sidebar means **Sidebar Focus**, not hover state or keyboard focus.
- "auto expand" in the sidebar means **Sidebar Reveal**, not expanding every folder in the tree.
- "leave protection" means **Workspace Tab Leave Guard**, not only the browser's refresh or close-page warning.
- "original fields" in MongoDB means **Document Field Order**, not alphabetical field order.
- The **Collection Table View** is the default MongoDB collection view.
- The **Collection Table View** should not infer fields from an extra collection sample.
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
