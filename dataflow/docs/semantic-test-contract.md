# DataFlow Semantic Test Contract

## 1. Module Info

- Module: DataFlow frontend automation semantics
- Page: standalone login, database workspace, SQL editor/table detail, MongoDB collection detail, Redis key detail, analysis dashboard
- Version: initial contract for semantic tag coverage
- Owner: frontend maintainers

## 2. Page Entries

| Page | Route | Notes |
| --- | --- | --- |
| Standalone login | SPA root when auth status is unauthenticated | Manual database session creation |
| Database workspace | SPA root after auth, connections activity tab | Sidebar tree, tabs, SQL editor, SQL/Mongo/Redis detail views |
| Analysis dashboard | SPA root after auth, analysis activity tab | Dashboard list, editor canvas, chart widgets |

## 3. Semantic Tags

| Element | Code location | data-testid | Type | Business semantics | data-qa-* | Operable | Assertable | Evidence source | Risk |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Auth bootstrap loading | `src/main.tsx` | `auth.bootstrap.loading` | state | Session bootstrap is loading | `module=auth`, `object=session`, `state=loading` | No | Yes | auth store state | Low |
| Auth bootstrap error | `src/main.tsx` | `auth.bootstrap.error` | error | Session bootstrap failed | `module=auth`, `object=session`, `state=error`, `error-code=bootstrap_failed` | No | Yes | auth store state | Medium |
| Standalone login disabled | `src/main.tsx` | `auth.standalone.disabled` | state | Manual login is disabled by runtime config | `module=auth`, `object=standalone-login`, `state=disabled`, `disabled-reason=standalone_login_disabled` | No | Yes | auth store state | Low |
| Standalone login page | `src/components/auth/StandaloneLogin.tsx` | `auth.standalone.page` | panel | Manual database connection entry | `module=auth`, `object=standalone-login`, `state=ready/submitting/error` | No | Yes | auth component state | Medium |
| Standalone login form | `src/components/auth/StandaloneLogin.tsx` | `auth.standalone.form` | action | Creates standalone session | `object=standalone-session`, `action=create`, `state=ready/submitting` | Yes | Yes | form submit handler | Medium |
| Standalone login fields | `src/components/auth/StandaloneLogin.tsx` | `auth.standalone.*-input`, `auth.standalone.database-type-select` | field | Host, port, username, password, database, type | `field=host/port/username/password/database/database_type`, `disabled-reason=submitting` | Yes | Yes | form state | Medium |
| Standalone login submit | `src/components/auth/StandaloneLogin.tsx` | `auth.standalone.submit-button` | action | Submits connection credentials | `action=create`, `state=ready/submitting`, `disabled-reason=submitting` | Yes | Yes | form submit handler | Medium |
| Standalone login error | `src/components/auth/StandaloneLogin.tsx` | `auth.standalone.error` | error | Session creation failed | `state=error`, `error-code=standalone_session_create_failed` | No | Yes | submit catch branch | Medium |
| App shell | `src/components/layout/MainLayout.tsx` | `layout.shell` | panel | Root workspace shell | `module=layout`, `object=app-shell`, `state=connections/analysis` | No | Yes | layout store | Low |
| Activity bar | `src/components/layout/ActivityBar.tsx` | `layout.activity-bar`, `layout.activity.tab` | action | Switches workspace mode | `object=activity-tab`, `action=switch`, `resource-id=connections/analysis`, `state=active/inactive` | Yes | Yes | layout store | Low |
| Sidebar region | `src/components/layout/MainLayout.tsx` | `layout.sidebar-region`, `layout.sidebar-resize-handle` | panel/action | Current sidebar and resize handle | `object=sidebar`, `action=resize`, `state=connections/analysis` | Yes | Yes | layout state | Low |
| Main region | `src/components/layout/MainLayout.tsx` | `layout.main-region` | panel | Current main workspace | `object=main`, `state=connections/analysis` | No | Yes | layout state | Low |
| Database sidebar | `src/components/sidebar/Sidebar.tsx` | `database.sidebar`, `database.sidebar.tree` | panel | Connection object tree | `module=database`, `object=connection-tree`, `state=ready/empty/loading` | No | Yes | connection store | Medium |
| Database tree node | `src/components/sidebar/SidebarTree/SidebarTree.Node.tsx` | `database.sidebar.tree-node` | item | Connection/database/schema/table/view/collection/Redis key node | `object=sidebar-node`, `resource-type`, `resource-id`, `connection-id`, `database`, `schema`, `state=selected/idle expanded/collapsed/leaf loading` | Yes | Yes | tree node data | Medium |
| Database tree toggle | `src/components/sidebar/SidebarTree/SidebarTree.Node.tsx` | `database.sidebar.tree-node-toggle` | action | Expand or collapse tree node | `action=expand/collapse`, `resource-type`, `resource-id`, `state=expanded/collapsed/loading` | Yes | Yes | tree node state | Low |
| Tab bar item | `src/components/layout/TabBar.tsx` | `layout.tab.item` | item/action | Open workspace tab | `object=tab`, `action=activate`, `tab-type`, `resource-id`, `connection-id`, `database`, `schema`, `state=active/inactive dirty` | Yes | Yes | tab store | Medium |
| Tab close/new query | `src/components/layout/TabBar.tsx` | `layout.tab.close-button`, `layout.tab.new-query-button` | action | Closes a tab or creates query tab | `action=close/create-query`, `resource-id=tab id`, `disabled-reason=not_ready` | Yes | Yes | tab store | Medium |
| Tab content | `src/components/layout/TabContent.tsx` | `layout.tab-content.panel`, `layout.tab-content.empty` | panel/state | Active tab render area | `tab-type`, `resource-id`, `connection-id`, `database`, `schema`, `state=active/inactive/empty` | No | Yes | tab store | Medium |
| SQL editor view | `src/components/editor/SQLEditorView.tsx` | `sql.editor.view` | panel | Query editor and execution result workspace | `module=sql`, `object=editor`, `connection-id`, `database`, `schema`, `state=ready/executing/completed/error`, `loading=true/false` | No | Yes | editor state, GraphQL query | High |
| SQL editor actions | `src/components/editor/SQLEditorView.tsx` | `sql.editor.run-button`, `sql.editor.format-button`, `sql.editor.create-chart-button` | action | Run, format, or chart query result | `object=query/query-result`, `action=execute/format/create-chart`, `risk=query_execution`, `disabled-reason=executing/empty_query/not_ready` | Yes | Yes | handlers and state | High |
| SQL execution context | `src/components/editor/SQLEditorView.tsx` | `sql.editor.database-select`, `sql.editor.schema-select`, `sql.editor.*-option` | field/item | Chooses database/schema for execution | `object=execution-context`, `field=database/schema`, `resource-type=database/schema`, `resource-id` | Yes | Yes | database/schema queries | Medium |
| SQL result pane | `src/components/editor/SQLEditorView.tsx` | `sql.editor.result-pane`, `sql.editor.result-set`, `sql.editor.result-table`, `sql.editor.message-item` | state/item/error | Execution output, rows, and messages | `object=query-result/statement-result/result-table/statement-message`, `state=empty/loading/success/error`, `row-count`, `error-code=query_execution_failed` | No | Yes | RawExecute result | High |
| SQL result row/cell | `src/components/editor/SQLEditorView.tsx` | `sql.editor.result-row`, `sql.editor.result-cell` | item | Query result evidence | `object=result-row/result-cell`, `field`, `row-index` | No | Yes | RawExecute result rows | Medium |
| SQL table detail | `src/components/database/sql/TableDetailView.tsx` | `sql.table.detail` | panel | Table data detail | `connection-id`, `database`, `schema`, `resource-type=table`, `resource-id=tableName`, `state=ready/loading/error` | No | Yes | table provider state | High |
| SQL table toolbar | `src/components/database/sql/TableView/TableView.Toolbar.tsx` | `sql.table.toolbar` | panel | Table operation toolbar | `object=table-toolbar`, `resource-id=tableName`, `state=ready/loading/dirty` | No | Yes | changeset state | High |
| SQL table actions | `src/components/database/sql/TableView/TableView.Toolbar.tsx` | `sql.table.*-button` | action | Refresh, row add/delete, undo, preview, submit, export, query, chart | `action=refresh/create/mark-delete/undo/preview/submit/export/open-query/create-chart`, `risk=resource_mutation`, `disabled-reason` | Yes | Yes | table actions | High |
| SQL table grid | `src/components/database/sql/TableView/TableView.DataGrid.tsx` | `sql.table.grid`, `sql.table.grid-scroll`, `sql.table.grid-loading` | state/panel | Table rows and loading state | `object=table-grid`, `state=ready/empty/loading`, `row-count` | No | Yes | table provider rows | High |
| SQL table row/cell | `src/components/database/sql/TableView/TableView.DataGrid.tsx` | `sql.table.row`, `sql.table.row-selector`, `sql.table.cell`, `sql.table.cell-editor` | item/field/action | Editable table rows and cells | `object=table-row/table-cell`, `field`, `resource-id=rowKey`, `state=ready/selected/inserted/deleted/editable/editing/changed/read_only`, `disabled-reason=read_only/primary_key/row_deleted` | Yes | Yes | rendered row state | High |
| Shared data view filter/error | `src/components/database/shared/*` | `data-view.filter-button`, `data-view.error`, `data-view.retry-button` | action/error | Data filter and load failure retry | `module=data-view`, `object=filter/data-load`, `action=open/retry`, `state=active/inactive/error`, `error-code=data_load_failed` | Yes | Yes | shared data view state | Medium |
| MongoDB collection detail | `src/components/database/mongodb/CollectionDetailView.tsx` | `mongodb.collection.detail`, `mongodb.collection.detail-loading` | panel/state | Collection document detail | `connection-id`, `database`, `resource-type=collection`, `resource-id=collectionName`, `state=ready/loading/error` | No | Yes | collection provider state | High |
| MongoDB toolbar actions | `src/components/database/mongodb/CollectionView/CollectionView.Toolbar.tsx` | `mongodb.collection.*-button`, `mongodb.collection.view-toggle-button` | action | Refresh, switch collection view, add/delete document, undo, preview, submit, export, query, chart | `action=refresh/switch-to-table/switch-to-json/create/mark-delete/undo/preview/submit/export/open-query/create-chart`, `state=table/json`, `risk=resource_mutation`, `disabled-reason` | Yes | Yes | collection actions and view mode | High |
| MongoDB document list/card | `src/components/database/mongodb/CollectionView/CollectionView.DocumentList.tsx` | `mongodb.collection.document-list-region`, `mongodb.collection.document-card`, `mongodb.collection.edit-document-button`, `mongodb.collection.document-list-empty` | item/action/state | Documents, edit action, empty list | `object=document/document-list`, `state=ready/selected/insert/update/delete/empty`, `resource-type=document`, `resource-id=rowKey` | Yes | Yes | document changeset state | High |
| Redis key detail | `src/components/database/redis/RedisKeyDetailView.tsx` | `redis.key.detail`, `redis.key.detail-loading` | panel/state | Redis key value detail | `connection-id`, `database`, `resource-type=redis_key`, `resource-id=keyName`, `key-type`, `state=ready/loading/mutating/error` | No | Yes | Redis key rows | High |
| Redis key toolbar actions | `src/components/database/redis/RedisKeyDetailView.tsx` | `redis.key.*-button` | action | Refresh, add/delete row, export, query, chart | `action=refresh/create/delete/export/open-query/create-chart`, `risk=resource_mutation`, `disabled-reason` | Yes | Yes | Redis key handlers | High |
| Redis grid row/cell | `src/components/database/redis/RedisKeyDetailView.tsx` | `redis.key.grid`, `redis.key.row`, `redis.key.row-selector`, `redis.key.cell`, `redis.key.cell-editor`, `redis.key.new-row`, `redis.key.new-row-input`, `redis.key.empty`, `redis.key.error` | item/field/state/error | Redis value rows and inline editing | `object=key-grid/key-row/key-cell`, `field`, `resource-id=keyName:index`, `state=ready/selected/editable/editing/read_only/creating/empty/error`, `error-code=redis_key_operation_failed` | Yes | Yes | Redis row state | High |
| Analysis view | `src/components/analysis/AnalysisView.tsx` | `analysis.view`, `analysis.view.error`, `analysis.dashboard.empty` | panel/state/error | Dashboard workspace loading and empty/error state | `module=analysis`, `object=dashboard-view/dashboard`, `resource-type=dashboard`, `resource-id`, `state=active/empty/error`, `error-code=dashboard_load_failed` | No | Yes | analysis definition store | Medium |
| Dashboard sidebar | `src/components/dashboard-sidebar/DashboardSidebar.tsx` | `analysis.dashboard.sidebar`, `analysis.dashboard.create-button`, `analysis.dashboard.list-item`, `analysis.dashboard.item-menu-button` | panel/action/item | Dashboard list and actions | `object=dashboard`, `action=create/open/open-menu`, `resource-type=dashboard`, `resource-id`, `state=ready/empty/active/inactive` | Yes | Yes | dashboard store | Medium |
| Dashboard editor | `src/components/analysis/editor/DashboardEditor.tsx` | `analysis.dashboard.editor`, `analysis.dashboard.toolbar`, `analysis.dashboard.refresh-button`, `analysis.dashboard.add-widget-button`, `analysis.dashboard.editor-empty`, `analysis.dashboard.empty-add-widget-button` | panel/action/state | Dashboard editor and widget creation | `object=dashboard-editor/dashboard/widget`, `action=refresh/create`, `resource-type=dashboard`, `resource-id`, `state=ready/empty` | Yes | Yes | dashboard store | Medium |
| Dashboard canvas/widget | `src/components/analysis/editor/EditorCanvas.tsx`, `src/components/analysis/editor/DashboardWidget.tsx` | `analysis.dashboard.canvas`, `analysis.dashboard.widget-layout`, `analysis.dashboard.widget`, `analysis.dashboard.widget-title`, `analysis.dashboard.widget-title-input`, `analysis.dashboard.widget-menu-button`, `analysis.dashboard.widget-error` | item/action/field/error | Widget layout, title editing, menu, runtime error | `object=widget/widget-layout`, `field=title`, `action=open-menu`, `resource-type=widget`, `resource-id`, `state=idle/loading/success/error/editing`, `error-code=widget_query_failed` | Yes | Yes | widget definition/runtime stores | Medium |

## 4. State Enums

| Element | `data-qa-state` values | Notes |
| --- | --- | --- |
| Auth/session | `loading`, `error`, `disabled`, `ready`, `submitting` | Authentication and standalone session lifecycle |
| Layout/activity/tab | `connections`, `analysis`, `active`, `inactive`, `dirty`, `empty` | Shell navigation state |
| Sidebar node | `selected`, `idle`, `expanded`, `collapsed`, `leaf`, `loading` | Combined as a space-separated state string when multiple apply |
| SQL editor/result | `ready`, `executing`, `completed`, `loading`, `success`, `error`, `empty` | Query execution and result display |
| SQL/Mongo/Redis mutation surfaces | `ready`, `loading`, `dirty`, `mutating`, `inserted`, `deleted`, `changed`, `selected`, `editable`, `editing`, `read_only`, `creating`, `table`, `json` | Table/document/key editing and MongoDB collection view mode |
| Analysis dashboard/widget | `ready`, `empty`, `active`, `inactive`, `editable`, `read_only`, `idle`, `loading`, `success`, `error`, `editing` | Dashboard and widget runtime |

## 5. Disabled Reason Enums

| disabled reason | Meaning | Automation expectation |
| --- | --- | --- |
| `submitting` | Login submission in progress | Wait for form state to settle |
| `loading` | Data is loading | Wait before clicking refresh-dependent action |
| `executing` | Query is running | Wait for result pane |
| `empty_query` | Format action has no query text | Fill editor before format |
| `not_ready` | Required result/context is unavailable | Produce prerequisite state first |
| `no_databases` | Database selector has no options | Verify database metadata loading |
| `database_required` | Schema selector requires selected database | Select database first |
| `no_schemas` | Schema selector has no options | Verify schema metadata |
| `no_selection` | Row/document delete needs selection | Select rows/documents first |
| `no_pending_undo` | Undo stack is empty | Create a pending change first |
| `no_pending_changes` | Preview/submit needs pending changes | Create a pending change first |
| `read_only` | Cell/row cannot be edited | Do not attempt edit action |
| `primary_key` | Primary key cell is not editable | Assert non-editable primary key |
| `row_deleted` | Deleted row cell cannot be edited | Undo or skip edit |
| `pending_row_active` | Redis new row already open | Complete or cancel active row |
| `mutating` | Redis mutation in progress | Wait for mutation completion |
| `standalone_login_disabled` | Runtime configuration disables manual login | Do not attempt standalone login |

## 6. Error Code Enums

| error code | Meaning | Automation expectation |
| --- | --- | --- |
| `bootstrap_failed` | Auth bootstrap failed | Capture session/bootstrap error evidence |
| `standalone_session_create_failed` | Manual session creation failed | Assert failure and error region |
| `query_execution_failed` | SQL editor statement failed | Assert message or result-set error item |
| `data_load_failed` | Shared data detail load failed | Retry may be available |
| `redis_key_operation_failed` | Redis key operation failed | Capture Redis detail error banner |
| `dashboard_load_failed` | Dashboard metadata load failed | Assert analysis error surface |
| `widget_query_failed` | Analysis widget query/runtime failed | Assert widget error state |

## 7. Resource Binding

| Element | Resource fields | Notes |
| --- | --- | --- |
| Activity tab | `data-qa-resource-type=activity-tab`, `data-qa-resource-id` | Stable IDs: `connections`, `analysis` |
| Sidebar nodes | `data-qa-resource-type`, `data-qa-resource-id`, `data-qa-connection-id`, `data-qa-database`, `data-qa-schema` | Node IDs come from tree data; test ID remains stable across instances |
| Tabs and tab panels | `data-qa-resource-type=tab`, `data-qa-resource-id`, `data-qa-tab-type`, `data-qa-connection-id`, `data-qa-database`, `data-qa-schema` | Tab ID is resource binding, not part of `data-testid` |
| SQL table detail | `data-qa-resource-type=table`, `data-qa-resource-id=tableName`, plus connection/database/schema | Table identity for API correlation |
| SQL table rows/cells | `data-qa-resource-type=table-row`, `data-qa-resource-id=rowKey`, `data-qa-field` | Row key is data-source row key or pending-row key |
| MongoDB collection | `data-qa-resource-type=collection`, `data-qa-resource-id=collectionName`, plus connection/database | Collection identity |
| MongoDB document cards | `data-qa-resource-type=document`, `data-qa-resource-id=rowKey` | Row key is changeset/document position key, not Mongo `_id` |
| Redis key detail | `data-qa-resource-type=redis_key`, `data-qa-resource-id=keyName`, `data-qa-key-type` | Key name is visible resource identity |
| Redis rows/cells | `data-qa-resource-type=redis_key_row`, `data-qa-resource-id=keyName:index`, `data-qa-field` | Index-based binding reflects Redis row display position |
| Dashboards/widgets | `data-qa-resource-type=dashboard/widget`, `data-qa-resource-id` | Store IDs for dashboard and widget correlation |

## 8. Coverage Notes

| Requirement element | Coverage state | Notes |
| --- | --- | --- |
| Core user action buttons | covered | Login, activity switch, tabs, query run/format, table/collection/Redis operations, dashboard/widget operations |
| Automation-facing fields | covered | Login fields, SQL database/schema selectors, editable table/Redis cells, widget title edit |
| Key states | covered | Loading/empty/error/ready/dirty/executing/mutating/selected/editing states |
| Disabled reasons | covered | Common disabled action reasons use stable enum values |
| Error surfaces | covered | Auth, data load, SQL query, Redis operation, analysis dashboard/widget errors |
| Resource rows/cards/detail roots | covered | Sidebar nodes, tabs, table rows, Mongo documents, Redis rows, dashboards/widgets |
| Pure layout/decorative elements | skipped | Icons, spacing wrappers, separators, visual-only containers intentionally remain untagged |

## 9. Change Rules

- New core operations must add a stable `data-testid` and relevant `data-qa-*` semantics.
- Repeated business objects must keep a shared stable `data-testid`; instance identity belongs in `data-qa-resource-*`.
- Do not add dynamic IDs, array indexes, visible text, CSS class names, or component library internals to `data-testid`.
- New disabled or error branches must use stable enum strings and update this document.
- Renaming or deleting a tag in this file is a test-contract change and must be called out in review.
