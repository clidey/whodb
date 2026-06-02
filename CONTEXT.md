# WhoDB Data Exploration Context

WhoDB helps users inspect and manipulate database storage units across relational and document databases. This context records product language for database exploration views.

## Language

**MongoDB Collection**:
A MongoDB storage unit made of documents that may not all share the same fields.
_Avoid_: MongoDB table

**MongoDB Document**:
A single record inside a **MongoDB Collection**.
_Avoid_: row, JSON row

**JSON View**:
The MongoDB collection view that shows each **MongoDB Document** as editable JSON.
_Avoid_: card-only view

**Collection Table View**:
A MongoDB collection view that presents documents in a grid using document fields as columns.
_Avoid_: MongoDB table

**Sampled Field Set**:
A field list inferred from a limited sample of documents for the **Collection Table View**.
_Avoid_: complete schema

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

## Relationships

- A **MongoDB Collection** contains zero or more **MongoDB Documents**.
- A **JSON View** displays **MongoDB Documents** in their native document shape.
- A **Collection Table View** presents **MongoDB Documents** as rows while preserving MongoDB's flexible field model.
- A **Sampled Field Set** guides the columns shown in a **Collection Table View** but does not represent a complete MongoDB schema.
- An **Unset Field** is distinct from a field whose stored value is `null`.
- Editing an **Unset Field** creates that field on the affected **MongoDB Document**.
- An **Editable Scalar Field** can be edited inline in the **Collection Table View**.
- Editing an **Editable Scalar Field** preserves the existing field type when the field already exists, except when the input is a complete, valid, unquoted JSON object or array.
- A **Complex Document Field** in the **Collection Table View** can open a **Field JSON Editor**.
- A **Field JSON Editor** accepts any valid JSON value, even when that changes an object or array field into a scalar or `null`.
- A **MongoDB Document** is edited through a **Document JSON Editor**.
- A **Complex Document Field** is not edited through a separate field-level interaction inside the **Document JSON Editor**.

## Example Dialogue

> **Dev:** "Should MongoDB open in the table by default?"
> **Domain expert:** "Yes. Open MongoDB collections in the **Collection Table View** by default because users expect a grid for browsing. Keep the **JSON View** available as a switchable document-focused view."

## Flagged Ambiguities

### Terminology

- "table view" in MongoDB means **Collection Table View**, not a relational database table.
- The document editor should be a **Document JSON Editor**, not a table view, field list, or field-level editor.
- The **Collection Table View** is the default MongoDB collection view.
- The **Collection Table View** should build its first column set from a limited default sample, not by scanning the full collection.
- The **Collection Table View** supports sorting and filtering on top-level document fields.

### MongoDB Table Editing

- MongoDB inline editing is limited to **Editable Scalar Fields**; object and array cells in the **Collection Table View** open a **Field JSON Editor**.
- A **Field JSON Editor** validates JSON syntax only. It does not force the edited value to remain an object or array.
- Empty input or clearing an existing field is not field deletion; field deletion must be a distinct action.
- Editing a `null` or **Unset Field** in the **Collection Table View** creates a string value unless the input is a complete, valid, unquoted JSON object or array.
- Typing a complete, valid, unquoted JSON object or array into any **Editable Scalar Field** changes that field into a **Complex Document Field**.

### MongoDB View State

- Switching between **Collection Table View** and **JSON View** preserves pending document changes.
- Pending changes from **Collection Table View** and **JSON View** share the same document-level preview and submission flow.
