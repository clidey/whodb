import React from "react";
import {
  Terminal, Plus, Download, Upload, Edit2, Trash2,
  RefreshCw, Eraser, Copy, Eye, EyeOff,
} from "lucide-react";
import type { ContextMenuItem } from "@/components/ui/ContextMenu";
import type { MessageKey } from "@/i18n/messages";

type ConnectionType = "MYSQL" | "POSTGRES" | "MONGODB" | "REDIS" | "CLICKHOUSE";

interface MenuCallbacks {
  onAction: (action: string) => void;
  t: (key: MessageKey) => string;
}

interface SystemObjectsState {
  systemSchemas: string[];
  showSystemObjects: boolean;
}

const CONNECTION_TYPES_WITH_DATABASE_CREATE: ReadonlySet<ConnectionType> = new Set([
  "MYSQL",
  "MONGODB",
  "POSTGRES",
  "CLICKHOUSE",
]);

const CONNECTION_TYPES_WITH_DATABASE_RENAME: ReadonlySet<ConnectionType> = new Set([
  "POSTGRES",
]);

const CONNECTION_TYPES_WITH_DATABASE_DELETE: ReadonlySet<ConnectionType> = new Set([
  "MYSQL",
  "MONGODB",
  "POSTGRES",
  "CLICKHOUSE",
]);

const CONNECTION_TYPES_WITH_DATABASE_EXPORT: ReadonlySet<ConnectionType> = new Set([
  "MYSQL",
  "POSTGRES",
  "CLICKHOUSE",
]);

const CONNECTION_TYPES_WITH_SQL_IMPORT: ReadonlySet<ConnectionType> = new Set([
  "MYSQL",
  "POSTGRES",
  "CLICKHOUSE",
]);

const CONNECTION_TYPES_WITH_TABLE_COPY: ReadonlySet<ConnectionType> = new Set([
  "MYSQL",
  "POSTGRES",
]);

const CONNECTION_TYPES_WITH_TABLE_RENAME: ReadonlySet<ConnectionType> = new Set([
  "MYSQL",
  "POSTGRES",
]);

function refreshItem(
  onAction: (action: string) => void,
  t: (key: MessageKey) => string
): ContextMenuItem {
  return {
    label: t("sidebar.menu.refresh"),
    onClick: () => onAction("refresh"),
    icon: React.createElement(RefreshCw, { className: "h-4 w-4" }),
  };
}

export function getConnectionMenuItems(
  connectionType: ConnectionType,
  callbacks: MenuCallbacks,
  systemObjectsState?: SystemObjectsState
): ContextMenuItem[] {
  const { onAction, t } = callbacks;
  const canCreateDatabase = CONNECTION_TYPES_WITH_DATABASE_CREATE.has(connectionType);
  // Connection-level toggle only applies to types where systemSchemas are
  // database names (MongoDB, MySQL, ClickHouse). For Postgres, systemSchemas
  // are schema names — the toggle belongs at the database level instead.
  const systemItems: ContextMenuItem[] = connectionType !== "POSTGRES" && systemObjectsState && systemObjectsState.systemSchemas.length > 0
    ? [
        { separator: true },
        {
          label: systemObjectsState.showSystemObjects
            ? t("sidebar.menu.hideSystemObjects")
            : t("sidebar.menu.showSystemObjects"),
          onClick: () => onAction("toggle_system_objects"),
          icon: React.createElement(
            systemObjectsState.showSystemObjects ? EyeOff : Eye,
            { className: "h-4 w-4" }
          ),
        },
      ]
    : [];
  return [
    { label: t("sidebar.menu.newQuery"), onClick: () => onAction("new_query"), icon: React.createElement(Terminal, { className: "h-4 w-4" }) },
    ...(canCreateDatabase
      ? [
          { separator: true },
          { label: t("sidebar.menu.newDatabase"), onClick: () => onAction("new_database"), icon: React.createElement(Plus, { className: "h-4 w-4" }) },
        ] as ContextMenuItem[]
      : []),
    { separator: true },
    refreshItem(onAction, t),
    ...systemItems,
  ];
}

export function getDatabaseMenuItems(
  connectionType: ConnectionType,
  callbacks: MenuCallbacks,
  systemObjectsState?: SystemObjectsState
): ContextMenuItem[] {
  const { onAction, t } = callbacks;
  const canRenameDatabase = CONNECTION_TYPES_WITH_DATABASE_RENAME.has(connectionType);
  const canDeleteDatabase = CONNECTION_TYPES_WITH_DATABASE_DELETE.has(connectionType);
  const canExportDatabase = CONNECTION_TYPES_WITH_DATABASE_EXPORT.has(connectionType);
  const canImportDatabase = CONNECTION_TYPES_WITH_SQL_IMPORT.has(connectionType);
  // Database-level toggle only applies to Postgres where systemSchemas are
  // schema names filtered within a database. For other types, collections/tables
  // have no frontend-level system object filtering.
  const systemItems: ContextMenuItem[] = connectionType === "POSTGRES" && systemObjectsState && systemObjectsState.systemSchemas.length > 0
    ? [
        { separator: true },
        {
          label: systemObjectsState.showSystemObjects
            ? t("sidebar.menu.hideSystemObjects")
            : t("sidebar.menu.showSystemObjects"),
          onClick: () => onAction("toggle_system_objects"),
          icon: React.createElement(
            systemObjectsState.showSystemObjects ? EyeOff : Eye,
            { className: "h-4 w-4" }
          ),
        },
      ]
    : [];
  const creationItems: ContextMenuItem[] = connectionType === "MONGODB"
    ? [
        { label: t("sidebar.menu.newCollection"), onClick: () => onAction("new_collection"), icon: React.createElement(Plus, { className: "h-4 w-4" }) },
      ]
    : connectionType !== "REDIS"
    ? [
        { label: t("sidebar.menu.newTable"), onClick: () => onAction("new_table"), icon: React.createElement(Plus, { className: "h-4 w-4" }) },
      ]
    : [];

  const managementItems: ContextMenuItem[] = [
    ...(canImportDatabase
      ? [
          { label: t("sidebar.menu.import"), onClick: () => onAction("import_database"), icon: React.createElement(Upload, { className: "h-4 w-4" }) },
        ] as ContextMenuItem[]
      : []),
    ...(canExportDatabase
      ? [
          { label: t("sidebar.menu.exportDatabase"), onClick: () => onAction("export_database"), icon: React.createElement(Download, { className: "h-4 w-4" }) },
        ] as ContextMenuItem[]
      : []),
    ...(canRenameDatabase
      ? [{ label: t("sidebar.menu.renameDatabase"), onClick: () => onAction("edit_database"), icon: React.createElement(Edit2, { className: "h-4 w-4" }) }] as ContextMenuItem[]
      : []),
  ];

  const destructiveItems: ContextMenuItem[] = canDeleteDatabase
    ? [
        { label: t("sidebar.menu.deleteDatabase"), onClick: () => onAction("delete_database"), icon: React.createElement(Trash2, { className: "h-4 w-4 text-red-500" }), danger: true },
      ]
    : [];

  return [
    { label: t("sidebar.menu.newQuery"), onClick: () => onAction("new_query"), icon: React.createElement(Terminal, { className: "h-4 w-4" }) },
    ...(creationItems.length > 0 ? [{ separator: true }, ...creationItems] as ContextMenuItem[] : []),
    ...(managementItems.length > 0 ? [{ separator: true }, ...managementItems] as ContextMenuItem[] : []),
    ...(destructiveItems.length > 0 ? [{ separator: true }, ...destructiveItems] as ContextMenuItem[] : []),
    { separator: true },
    refreshItem(onAction, t),
    ...systemItems,
  ];
}

export function getSchemaMenuItems(connectionType: ConnectionType, callbacks: MenuCallbacks): ContextMenuItem[] {
  const { onAction, t } = callbacks;
  const canImportSchema = CONNECTION_TYPES_WITH_SQL_IMPORT.has(connectionType);
  return [
    { label: t("sidebar.menu.newQuery"), onClick: () => onAction("new_query"), icon: React.createElement(Terminal, { className: "h-4 w-4" }) },
    { separator: true },
    { label: t("sidebar.menu.newTable"), onClick: () => onAction("new_table"), icon: React.createElement(Plus, { className: "h-4 w-4" }) },
    ...(canImportSchema
      ? [
          { separator: true },
          { label: t("sidebar.menu.import"), onClick: () => onAction("import_database"), icon: React.createElement(Upload, { className: "h-4 w-4" }) },
        ] as ContextMenuItem[]
      : []),
    { separator: true },
    refreshItem(onAction, t),
  ];
}

export function getTableFolderMenuItems(callbacks: MenuCallbacks): ContextMenuItem[] {
  const { onAction, t } = callbacks;
  return [
    { label: t("sidebar.menu.newTable"), onClick: () => onAction("new_table"), icon: React.createElement(Plus, { className: "h-4 w-4" }) },
    { separator: true },
    refreshItem(onAction, t),
  ];
}

export function getViewFolderMenuItems(callbacks: MenuCallbacks): ContextMenuItem[] {
  const { onAction, t } = callbacks;
  return [
    refreshItem(onAction, t),
  ];
}

export function getTableMenuItems(connectionType: ConnectionType, callbacks: MenuCallbacks): ContextMenuItem[] {
  const { onAction, t } = callbacks;
  const canCopyTable = CONNECTION_TYPES_WITH_TABLE_COPY.has(connectionType);
  const canRenameTable = CONNECTION_TYPES_WITH_TABLE_RENAME.has(connectionType);
  const canImportTable = CONNECTION_TYPES_WITH_SQL_IMPORT.has(connectionType);
  return [
    ...(canImportTable
      ? [{ label: t("sidebar.menu.import"), onClick: () => onAction("import_database"), icon: React.createElement(Upload, { className: "h-4 w-4" }) }] as ContextMenuItem[]
      : []),
    { label: t("sidebar.menu.exportData"), onClick: () => onAction("export_data"), icon: React.createElement(Download, { className: "h-4 w-4" }) },
    ...(canCopyTable
      ? [{ label: t("sidebar.menu.duplicateTable"), onClick: () => onAction("copy_table"), icon: React.createElement(Copy, { className: "h-4 w-4" }) }] as ContextMenuItem[]
      : []),
    { separator: true },
    ...(connectionType === "CLICKHOUSE"
      ? []
      : [{ label: t("sidebar.menu.designTable"), onClick: () => onAction("edit_table"), icon: React.createElement(Edit2, { className: "h-4 w-4" }) }] as ContextMenuItem[]),
    ...(canRenameTable
      ? [{ label: t("sidebar.menu.renameTable"), onClick: () => onAction("rename_table"), icon: React.createElement(Edit2, { className: "h-4 w-4" }) }] as ContextMenuItem[]
      : []),
    { separator: true },
    { label: t("sidebar.menu.clearData"), onClick: () => onAction("clear_table_data"), icon: React.createElement(Eraser, { className: "h-4 w-4" }) },
    { label: t("sidebar.menu.deleteTable"), onClick: () => onAction("delete_table"), icon: React.createElement(Trash2, { className: "h-4 w-4 text-red-500" }), danger: true },
    { separator: true },
    refreshItem(onAction, t),
  ];
}

export function getViewMenuItems(callbacks: MenuCallbacks): ContextMenuItem[] {
  const { onAction, t } = callbacks;
  return [
    { label: t("sidebar.menu.exportData"), onClick: () => onAction("export_data"), icon: React.createElement(Download, { className: "h-4 w-4" }) },
    { separator: true },
    refreshItem(onAction, t),
  ];
}

export function getCollectionMenuItems(callbacks: MenuCallbacks): ContextMenuItem[] {
  const { onAction, t } = callbacks;
  return [
    { label: t("sidebar.menu.exportCollection"), onClick: () => onAction("export_collection"), icon: React.createElement(Download, { className: "h-4 w-4" }) },
    { separator: true },
    { label: t("sidebar.menu.dropCollection"), onClick: () => onAction("drop_collection"), icon: React.createElement(Trash2, { className: "h-4 w-4 text-red-500" }), danger: true },
    { separator: true },
    refreshItem(onAction, t),
  ];
}

export function getRedisKeysFolderMenuItems(callbacks: MenuCallbacks): ContextMenuItem[] {
  const { onAction, t } = callbacks;
  return [
    { label: t("sidebar.menu.newKey"), onClick: () => onAction("new_redis_key"), icon: React.createElement(Plus, { className: "h-4 w-4" }) },
    { separator: true },
    refreshItem(onAction, t),
  ];
}

export function getRedisKeyMenuItems(callbacks: MenuCallbacks): ContextMenuItem[] {
  const { onAction, t } = callbacks;
  return [
    { label: t("sidebar.menu.exportKey"), onClick: () => onAction("export_redis_key"), icon: React.createElement(Download, { className: "h-4 w-4" }) },
    { separator: true },
    { label: t("sidebar.menu.deleteKey"), onClick: () => onAction("delete_redis_key"), icon: React.createElement(Trash2, { className: "h-4 w-4 text-red-500" }), danger: true },
    { separator: true },
    refreshItem(onAction, t),
  ];
}
