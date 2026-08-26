import { hydrationRules, hydrationDefault } from './generated/hydration.js';
import type { DatasetQueryResult, OntologyObjectType, RowsResult } from './generated/types.js';

/** A hydrated row: column name → native-typed value. */
export type Row = Record<string, unknown>;

/** Coerces one stringly-typed cell into its native type per the shared rules. */
export function coerceValue(raw: string | null, columnType: string): unknown {
  if (raw === null || raw === undefined) return null;
  const kind = hydrationRules[columnType.toLowerCase()] ?? hydrationDefault;
  switch (kind) {
    case 'int': {
      const parsed = Number.parseInt(raw, 10);
      return Number.isNaN(parsed) ? raw : parsed;
    }
    case 'float': {
      const parsed = Number.parseFloat(raw);
      return Number.isNaN(parsed) ? raw : parsed;
    }
    case 'bool':
      return raw === 'true' || raw === 't' || raw === '1';
    case 'timestamp':
    case 'date': {
      const parsed = new Date(raw);
      return Number.isNaN(parsed.getTime()) ? raw : parsed;
    }
    case 'json': {
      try {
        return JSON.parse(raw);
      } catch {
        return raw;
      }
    }
    default:
      return raw;
  }
}

interface ColumnsAndRows {
  columns: Array<{ name: string; type: string }>;
  rows: Array<Array<string | null>>;
  totalCount?: number | null;
}

/**
 * Normalizes the two wire result shapes to one:
 * - DatasetQueryResult: { columns: string[] (names only), rows, total }
 * - RowsResult (CE-derived): { Columns: {Name,Type}[], Rows, TotalCount }
 * DatasetQueryResult carries no column types — coercion for it comes from
 * ontology property metadata (dataType), falling back to string.
 */
function normalize(result: DatasetQueryResult | RowsResult): ColumnsAndRows {
  const any = result as unknown as Record<string, unknown>;
  if (Array.isArray(any.columns)) {
    return {
      columns: (any.columns as string[]).map(name => ({ name, type: '' })),
      rows: (any.rows as Array<Array<string | null>>) ?? [],
      totalCount: (any.total as number | null) ?? null,
    };
  }
  return {
    columns: ((any.Columns as Array<{ Name: string; Type: string }>) ?? []).map(c => ({ name: c.Name, type: c.Type })),
    rows: ((any.Rows as Array<Array<string | null>>) ?? []),
    totalCount: (any.TotalCount as number | null) ?? null,
  };
}

/** Property-metadata type overrides: property apiName → ontology dataType. */
export type PropertyTypes = Map<string, string>;

/** Builds a property-type map from ontology entity metadata. */
export function propertyTypesOf(entity: OntologyObjectType): PropertyTypes {
  const map = new Map<string, string>();
  for (const property of entity.properties ?? []) {
    if (property?.apiName && property?.dataType) {
      map.set(property.apiName, property.dataType);
    }
  }
  return map;
}

/**
 * Hydrates a stringly-typed result into native-typed row objects. Ontology
 * property metadata, when supplied, overrides the wire column type — the
 * ontology's dataType is more precise than the storage column type.
 */
export function hydrateRows(
  result: DatasetQueryResult | RowsResult,
  propertyTypes?: PropertyTypes,
): { rows: Row[]; totalCount: number | null } {
  const { columns, rows, totalCount } = normalize(result);
  const hydrated = rows.map(cells => {
    const row: Row = {};
    columns.forEach((column, index) => {
      const type = propertyTypes?.get(column.name) ?? column.type;
      row[column.name] = coerceValue(cells[index] ?? null, type);
    });
    return row;
  });
  return { rows: hydrated, totalCount: totalCount ?? null };
}
