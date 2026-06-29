import type { GetStorageUnitRowsQuery } from '@graphql';

type GqlColumn = GetStorageUnitRowsQuery['Row']['Columns'][number];

/**
 * Data format expected by TableDetailView and related components.
 */
export interface TableData {
  /** Column names in display order. */
  columns: string[];
  /** Column type name keyed by column name. */
  columnTypes: Record<string, string>;
  /** Row objects: { columnName: cellValue }. */
  rows: Record<string, string>[];
  /** Primary key column name, if any. */
  primaryKey: string | null;
  /** All primary key column names in result column order. */
  primaryKeyColumns: string[];
  /** Column names that are foreign keys. */
  foreignKeyColumns: string[];
  /** Total row count (server-side, for pagination). */
  total: number;
  /** Whether row editing is disabled for this storage unit. */
  disableUpdate: boolean;
}

/**
 * Convert a GraphQL RowsResult into the format TableDetailView expects.
 */
export function transformRowsResult(result: GetStorageUnitRowsQuery['Row']): TableData {
  const columns = result.Columns.map((c) => c.Name);

  const columnTypes: Record<string, string> = {};
  for (const col of result.Columns) {
    columnTypes[col.Name] = col.Type;
  }

  const rows = result.Rows.map((row) => {
    const obj: Record<string, string> = {};
    for (let i = 0; i < result.Columns.length; i++) {
      obj[result.Columns[i].Name] = row[i];
    }
    return obj;
  });

  const primaryKeyColumns = result.Columns
    .filter((c) => c.IsPrimary)
    .map((c) => c.Name);
  const primaryKey = primaryKeyColumns[0] ?? null;

  const foreignKeyColumns = result.Columns
    .filter((c) => c.IsForeignKey)
    .map((c) => c.Name);

  return {
    columns,
    columnTypes,
    rows,
    primaryKey,
    primaryKeyColumns,
    foreignKeyColumns,
    total: result.TotalCount,
    disableUpdate: result.DisableUpdate,
  };
}
