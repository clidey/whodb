import { hydrationRules, hydrationDefault } from './generated/hydration.js';
/** Coerces one stringly-typed cell into its native type per the shared rules. */
export function coerceValue(raw, columnType) {
    if (raw === null || raw === undefined)
        return null;
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
            }
            catch {
                return raw;
            }
        }
        default:
            return raw;
    }
}
/**
 * Normalizes the two wire result shapes to one:
 * - DatasetQueryResult: { columns: string[] (names only), rows, total }
 * - RowsResult (CE-derived): { Columns: {Name,Type}[], Rows, TotalCount }
 * DatasetQueryResult carries no column types — coercion for it comes from
 * ontology property metadata (dataType), falling back to string.
 */
function normalize(result) {
    const any = result;
    if (Array.isArray(any.columns)) {
        return {
            columns: any.columns.map(name => ({ name, type: '' })),
            rows: any.rows ?? [],
            totalCount: any.total ?? null,
        };
    }
    return {
        columns: (any.Columns ?? []).map(c => ({ name: c.Name, type: c.Type })),
        rows: (any.Rows ?? []),
        totalCount: any.TotalCount ?? null,
    };
}
/** Builds a property-type map from ontology entity metadata. */
export function propertyTypesOf(entity) {
    const map = new Map();
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
export function hydrateRows(result, propertyTypes) {
    const { columns, rows, totalCount } = normalize(result);
    const hydrated = rows.map(cells => {
        const row = {};
        columns.forEach((column, index) => {
            const type = propertyTypes?.get(column.name) ?? column.type;
            row[column.name] = coerceValue(cells[index] ?? null, type);
        });
        return row;
    });
    return { rows: hydrated, totalCount: totalCount ?? null };
}
