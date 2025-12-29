# WhoDB Feature List

This document contains a comprehensive list of WhoDB features, compiled for test coverage planning.

## 1. Database Support & Connectivity

### Supported Databases

| Database | Edition | Category | Plugin Location |
|----------|---------|----------|-----------------|
| PostgreSQL | CE | SQL | `core/src/plugins/postgres` |
| MySQL | CE | SQL | `core/src/plugins/mysql` |
| MariaDB | CE | SQL | `core/src/plugins/mariadb` |
| SQLite | CE | SQL | `core/src/plugins/sqlite` |
| MongoDB | CE | Document | `core/src/plugins/mongo` |
| Redis | CE | Key-Value | `core/src/plugins/redis` |
| Elasticsearch | CE | Search Engine | `core/src/plugins/elasticsearch` |
| ClickHouse | CE | OLAP | `core/src/plugins/clickhouse` |
| MSSQL | EE | SQL | `ee/core/src/plugins/mssql` |
| Oracle | EE | SQL | `ee/core/src/plugins/oracle` |
| DynamoDB | EE | Document | `ee/core/src/plugins/dynamodb` |

### Connection Features

- [ ] Profile-based connection storage (local/browser storage)
- [ ] Connection history for quick reconnection
- [ ] Standard credential fields (host, port, user, password, database)
- [ ] Schema selection (for databases that support it)
- [ ] Advanced connection options
- [ ] Cookie-based authentication (`AuthKey_Token`)
- [ ] Desktop: OS Keychain credential storage

## 2. Data Exploration & Navigation

### Schema Browser

- [ ] List all tables/collections/indices
- [ ] Schema/database selection dropdown
- [ ] Table search/filter
- [ ] View mode toggle (List vs Card)

### Metadata Exploration

- [ ] View table type (BASE TABLE, VIEW, etc.)
- [ ] View table size
- [ ] View row count
- [ ] View column names and data types
- [ ] View primary keys
- [ ] View foreign keys
- [ ] View indexes

## 3. Data Viewing

### Table/Grid Display

- [ ] Display tabular data with columns and rows
- [ ] Column headers with field names
- [ ] Row selection (checkbox column)
- [ ] Scrollable content for large datasets

### Pagination

- [ ] Configurable page size (dropdown)
- [ ] Page navigation (next/previous)
- [ ] Current page indicator
- [ ] Total count display

### Sorting

- [ ] Click column header to sort
- [ ] Ascending/Descending toggle
- [ ] Sort indicator on column header

### Search

- [ ] Global search input
- [ ] Real-time filtering
- [ ] Cell highlighting for matches
- [ ] Navigate between matches

### Where Conditions (Filtering)

- [ ] Single condition filtering
- [ ] Multiple conditions (AND logic)
- [ ] Condition operators: =, !=, >, <, >=, <=
- [ ] Database-specific operators (e.g., MongoDB: eq, match)
- [ ] Condition popover mode
- [ ] Condition sheet mode (for many conditions)
- [ ] "+N more" badge for overflow conditions
- [ ] Edit existing conditions
- [ ] Remove individual conditions
- [ ] Clear all conditions

## 4. CRUD Operations

### Create (Add Row/Document)

- [ ] Add row button
- [ ] Form dialog with field inputs
- [ ] Type validation
- [ ] Submit and verify insertion
- [ ] Cancel without saving

### Read

- [ ] View data in table format
- [ ] View individual cell values
- [ ] View nested/JSON data (document databases)

### Update (Edit Row/Document)

- [ ] Click cell to edit
- [ ] Save changes
- [ ] Revert to original value
- [ ] Cancel edit without saving
- [ ] Type casting for numeric values

### Delete

- [ ] Context menu delete option
- [ ] Delete confirmation (if applicable)
- [ ] Verify row removal

## 5. Type Casting

- [ ] String to integer conversion
- [ ] String to bigint conversion (large numbers like 5000000000)
- [ ] String to smallint conversion
- [ ] String to decimal/numeric conversion
- [ ] Zero value handling
- [ ] Negative number handling

## 6. Data Export

### Export Formats

- [ ] CSV export
- [ ] Excel XLSX export

### CSV Options

- [ ] Comma delimiter
- [ ] Semicolon delimiter
- [ ] Pipe delimiter
- [ ] Tab delimiter

### Export Scope

- [ ] Export all rows
- [ ] Export selected rows only
- [ ] Row selection UI for selective export

### Export Dialog

- [ ] Format selection dropdown
- [ ] Delimiter selection (for CSV)
- [ ] Selected row count indicator
- [ ] Download trigger

## 7. Schema Visualization (Graph)

### Graph Display

- [ ] Interactive node-based graph
- [ ] Table nodes with metadata
- [ ] Foreign key relationship edges
- [ ] One-to-One relationships
- [ ] One-to-Many relationships
- [ ] Many-to-Many relationships

### Graph Interaction

- [ ] Click node to view details
- [ ] Navigate from node to data view
- [ ] Zoom controls (in/out/fit)
- [ ] Pan/drag canvas
- [ ] Layout controls

### Node Information

- [ ] Table name
- [ ] Table type
- [ ] Size information
- [ ] Column list

## 8. SQL Scratchpad (Query Editor)

### Editor Features

- [ ] Multi-cell notebook interface
- [ ] Syntax highlighting
- [ ] Auto-completion
- [ ] Code editor (Monaco/CodeMirror)

### Cell Management

- [ ] Add new cell
- [ ] Remove cell
- [ ] Reorder cells

### Page Management

- [ ] Create new page
- [ ] Delete page
- [ ] Switch between pages
- [ ] Page tabs/navigation

### Query Execution

- [ ] Execute SELECT queries
- [ ] Execute UPDATE queries
- [ ] Execute INSERT queries
- [ ] Execute DELETE queries
- [ ] Display results in table format
- [ ] Display "Action Executed" for mutations
- [ ] Error message display for invalid queries

### Embedded Scratchpad Drawer

- [ ] Open from data view
- [ ] Pre-populated query (e.g., SELECT TOP 5 / LIMIT 5)
- [ ] Execute query in drawer
- [ ] View results in drawer
- [ ] Close drawer (ESC key)

## 9. Query History

- [ ] Automatic query persistence
- [ ] History panel display
- [ ] Search/filter history
- [ ] Copy query to clipboard
- [ ] Clone query to editor
- [ ] Execute query from history
- [ ] Timestamp display

## 10. Mock Data Generation

### Mock Data Dialog

- [ ] Open mock data sheet
- [ ] Row count input
- [ ] Maximum row limit enforcement (200)
- [ ] Append mode
- [ ] Overwrite mode
- [ ] Overwrite confirmation dialog

### Mock Data Validation

- [ ] Reject for tables with foreign keys
- [ ] Reject for unsupported table types
- [ ] Error toast for unsupported operations

## 11. AI Chat Assistant

### Chat Interface

- [ ] Chat panel/drawer
- [ ] Message input
- [ ] Send message
- [ ] Message history display
- [ ] User messages vs AI responses

### AI Providers

- [ ] OpenAI (ChatGPT)
- [ ] Anthropic (Claude)
- [ ] Ollama (Local LLMs)
- [ ] Provider selection
- [ ] Model selection

### Chat Capabilities

- [ ] Text responses
- [ ] Natural language to SQL conversion
- [ ] Generated SQL display
- [ ] Toggle between table view and SQL view
- [ ] Execute generated queries
- [ ] Display query results

### SQL Operations via Chat

- [ ] SELECT query generation
- [ ] Filtered query generation
- [ ] Aggregate query generation (COUNT, etc.)
- [ ] INSERT operation with confirmation
- [ ] UPDATE operation with confirmation
- [ ] DELETE operation with confirmation

### Chat Management

- [ ] Navigate chat history (arrow keys)
- [ ] Clear chat history
- [ ] New chat button
- [ ] Move query to scratchpad

### Error Handling

- [ ] Display errors for invalid queries
- [ ] Handle non-existent tables

## 12. UI/UX Features

### Theme & Customization

- [ ] Dark mode
- [ ] Light mode
- [ ] Font size options (Small/Medium/Large)
- [ ] Border radius options (None/Small/Medium/Large)
- [ ] Spacing options (Compact/Comfortable/Spacious)

### Layout

- [ ] Sidebar navigation
- [ ] Database/schema selector in sidebar
- [ ] Main content area
- [ ] Responsive design

### Notifications

- [ ] Toast notifications for success
- [ ] Toast notifications for errors
- [ ] Toast auto-dismiss

### Onboarding

- [ ] Product tour
- [ ] Tour step navigation
- [ ] Tour completion tracking

### Keyboard Shortcuts

- [ ] Mod+C (Copy)
- [ ] Mod+S (Save)
- [ ] Context menu shortcuts
- [ ] ESC to close dialogs/drawers

## 13. Authentication & Session

- [ ] Login page
- [ ] Database type selection
- [ ] Credential input form
- [ ] Login submission
- [ ] Session persistence
- [ ] Logout functionality
- [ ] Telemetry opt-in/opt-out dialog

## 14. Database-Specific Features

### Redis

- [ ] Hash data type display
- [ ] List data type display
- [ ] Set data type display
- [ ] Sorted set (zset) data type display
- [ ] String data type display
- [ ] Delete hash fields

### MongoDB

- [ ] Document/JSON display
- [ ] Nested object handling
- [ ] ObjectId handling
- [ ] Collection metadata

### Elasticsearch

- [ ] Index listing
- [ ] Document display
- [ ] Match operator for filtering

### DynamoDB (EE)

- [ ] Partition key display
- [ ] Item count metadata
- [ ] Table size metadata
- [ ] PartiQL query support

### Oracle (EE)

- [ ] FETCH FIRST N ROWS syntax
- [ ] Schema-qualified table names

### MSSQL (EE)

- [ ] SELECT TOP N syntax
- [ ] Schema-qualified table names

## 15. Features NOT Currently Implemented

These features do not exist in the codebase but could be future additions:

- [ ] Data Import (CSV/Excel)
- [ ] Connection profile sync to server
- [ ] Backup/Restore functionality
- [ ] Table/Schema creation (DDL)
- [ ] Index management
- [ ] Query performance analysis

---

## Test Coverage Matrix

Use this matrix to track which features have tests:

| Feature | Unit Test | E2E Test | Notes |
|---------|-----------|----------|-------|
| Connection | - | ✓ | All databases |
| Table listing | - | ✓ | `tables-list.cy.js` |
| Metadata | - | ✓ | `explore.cy.js` |
| Data display | - | ✓ | `data-view.cy.js` |
| Pagination | - | ✓ | `pagination.cy.js` |
| Sorting | - | ✓ | `sorting.cy.js` |
| Search | - | ✓ | `search.cy.js` |
| Where conditions | - | ✓ | `where-conditions.cy.js` |
| CRUD | - | ✓ | `crud.cy.js` |
| Type casting | - | ✓ | `type-casting.cy.js` |
| Export | - | ✓ | `export.cy.js` |
| Graph | - | ✓ | `graph.cy.js` |
| Scratchpad | - | ✓ | `scratchpad.cy.js` |
| Query history | - | ✓ | `query-history.cy.js` |
| Mock data | - | ✓ | `mock-data.cy.js` |
| AI Chat | - | ✓ | `chat.cy.js` |
| Settings | - | ✓ | `settings.cy.js` |
| Keyboard shortcuts | - | ✓ | `keyboard-shortcuts.cy.js` (ESC key, context menu) |
| Theme switching | - | ✓ | `settings.cy.js` (ModeToggle) |
| Graph zoom | - | ✗ | Not tested |
| Tour | - | ✗ | Not tested |
