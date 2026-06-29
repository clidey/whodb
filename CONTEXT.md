# WhoDB DataFlow Context

DataFlow is the database workspace in WhoDB: users explore connected databases, run database commands, edit data, import or export data, and turn results into analysis dashboards. This context records stable product language only; feature acceptance details and implementation trade-offs belong in specs, tests, or ADRs.

## Language

**Database Workspace**:
The workspace area for browsing database resources, running commands, and editing data.
_Avoid_: connections tab, database page

**Dashboard Workspace**:
The workspace area for creating and editing analysis dashboards.
_Avoid_: analysis tab, chart page

**Database Connection**:
A configured access path to one database engine or database service.
_Avoid_: account, cluster, datasource

**Database Resource**:
A user-visible database object that can be located and acted on from the workspace.
_Avoid_: storage unit, file, asset

**Workspace Tab**:
An open database exploration surface tied to a query or a **Database Resource**.
_Avoid_: browser tab, panel

**SQL Table**:
A relational **Database Resource** made of rows and columns.
_Avoid_: dataset, spreadsheet

**SQL View**:
A relational **Database Resource** whose rows are derived from a stored database query.
_Avoid_: table

**MongoDB Collection**:
A MongoDB **Database Resource** made of documents that may not all share the same fields.
_Avoid_: MongoDB table

**MongoDB Document**:
A single record inside a **MongoDB Collection**.
_Avoid_: row, JSON row

**Collection Table View**:
A MongoDB collection view that presents documents in a grid while preserving MongoDB document terminology.
_Avoid_: MongoDB table

**JSON View**:
A MongoDB collection view that shows each **MongoDB Document** as editable JSON.
_Avoid_: card-only view

**Redis Key**:
A Redis **Database Resource** addressed by key name.
_Avoid_: table, collection

**Database Export**:
A database operation where a user extracts database content from the active **Database Connection** into a downloadable format.
_Avoid_: backup when the export is not restorable, dump when only row data is exported

**SQL Data Export**:
A **Database Export** where relational row data is written as SQL INSERT statements.
_Avoid_: database dump, backup, restore script

**SQL Update Export**:
A **SQL Data Export** where relational row data is written as SQL UPDATE statements that identify rows by primary key.
_Avoid_: upsert export, import mapping, freeform update script

**Database Dump**:
A **Database Export** intended to recreate database content from database-native or database-equivalent script output.
_Avoid_: SQL data export, CSV export, table export

**Database Import**:
A database operation where a user brings external database content into the active **Database Connection**.
_Avoid_: restore-only flow, SQL import as the top-level flow

**Import Method**:
The kind of external content a **Database Import** uses.
_Avoid_: import mode, export format

**SQL Script Import**:
A **Database Import** method where a user provides SQL statements to execute against a SQL database.
_Avoid_: table file import, CSV import, database dump import

**Table File Import**:
A **Database Import** method where a user maps rows from a CSV or Excel file into one existing or newly created **SQL Table**.
_Avoid_: SQL script import, database dump import

**Collection File Import**:
A **Database Import** method where a user loads records from a JSON, CSV, or Excel file into one existing or newly created **MongoDB Collection**.
_Avoid_: table file import, SQL script import

**Dashboard**:
An analysis surface made of charts created from database query results.
_Avoid_: report, database view

**Chart**:
A visual summary of query result data used inside a **Dashboard**.
_Avoid_: widget when the data visualization is meant

## Relationships

- The **Database Workspace** and **Dashboard Workspace** are separate workspace areas.
- A **Database Workspace** uses one or more **Database Connections**.
- A **Database Connection** exposes zero or more **Database Resources**.
- A **Workspace Tab** belongs to the **Database Workspace**.
- A **Workspace Tab** can be tied to one **Database Resource**.
- A **SQL Table**, **SQL View**, **MongoDB Collection**, and **Redis Key** are each a kind of **Database Resource**.
- A **MongoDB Collection** contains zero or more **MongoDB Documents**.
- A **Collection Table View** and a **JSON View** are alternate views of a **MongoDB Collection**.
- A **Database Export** extracts content from exactly one active **Database Connection**.
- A **SQL Data Export** is a **Database Export** and is not a **Database Dump**.
- A **SQL Update Export** is a **SQL Data Export** and requires the source **SQL Table** to have a primary key.
- A **Database Dump** is a **Database Export** and may include database structure as well as data.
- A **Database Import** has exactly one **Import Method** for a single import run.
- A **SQL Script Import**, a **Table File Import**, and a **Collection File Import** are separate **Import Methods**.
- A **Table File Import** targets exactly one **SQL Table**.
- A **Table File Import** does not target a **MongoDB Collection** or **Redis Key**.
- A **Collection File Import** targets exactly one **MongoDB Collection** and does not target a **SQL Table** or **Redis Key**.
- A **Table File Import** that creates a new **SQL Table** may create that table without a primary key.
- When a **Table File Import** creates a new **SQL Table**, any selected primary key columns come from the source file columns rather than from DataFlow-generated columns.
- A **Dashboard** contains one or more **Charts**.
- A **Chart** is created from database query result data.

## Example Dialogue

> **Dev:** "Can we call every browsable thing a table in the sidebar?"
> **Domain expert:** "No. Use **Database Resource** only when you need a generic term. Say **SQL Table**, **MongoDB Collection**, or **Redis Key** when the database model matters."
>
> **Dev:** "Should MongoDB browsing use the same language as relational browsing?"
> **Domain expert:** "No. The grid is a **Collection Table View**, but the stored records are still **MongoDB Documents**, not rows."
>
> **Dev:** "When users upload SQL, is that the whole import feature?"
> **Domain expert:** "No. **SQL Script Import** is one **Import Method** inside **Database Import**. A CSV or Excel upload into a table is a **Table File Import**."
>
> **Dev:** "Can we call exported INSERT statements a dump?"
> **Domain expert:** "No. That is a **SQL Data Export**. A **Database Dump** is the restorable export path and has a stronger product promise."
>
> **Dev:** "If SQL export can generate UPDATE statements, are we configuring an import?"
> **Domain expert:** "No. A **SQL Update Export** still exports a script; it simply requires a primary key so each exported UPDATE statement targets one existing row."
>
> **Dev:** "Are charts part of database browsing?"
> **Domain expert:** "They start from query result data, but a saved analysis surface belongs to the **Dashboard Workspace** as a **Dashboard**."

## Flagged Ambiguities

- "table" means **SQL Table** unless explicitly referring to the MongoDB **Collection Table View**.
- "MongoDB table" is not a domain term; use **MongoDB Collection** or **Collection Table View** depending on whether the discussion is about storage or UI.
- "row" means a relational row unless the discussion is explicitly about a grid rendering; a stored MongoDB record is a **MongoDB Document**.
- "storage unit" is an internal or old documentation term; use **Database Resource** in product context.
- "import" means **Database Import** unless a specific **Import Method** such as **SQL Script Import** or **Table File Import** is named.
- "CSV/Excel import" means **Table File Import** when the target is a **SQL Table**.
- "primary key" in a new-table **Table File Import** means a source file column selected as the new **SQL Table** primary key; it does not mean DataFlow adds a generated id column.
- "SQL export" means **SQL Data Export** unless the discussion explicitly includes database structure or restore semantics.
- "update export" means **SQL Update Export**, not upsert or an import workflow.
- "dump" means **Database Dump**, not **SQL Data Export**.
- "dashboard" means an analysis surface in the **Dashboard Workspace**, not the database browsing workspace.
