import type { DatasetQueryResult, OntologyObjectType, RowsResult } from './generated/types.js';
/** A hydrated row: column name → native-typed value. */
export type Row = Record<string, unknown>;
/** Coerces one stringly-typed cell into its native type per the shared rules. */
export declare function coerceValue(raw: string | null, columnType: string): unknown;
/** Property-metadata type overrides: property apiName → ontology dataType. */
export type PropertyTypes = Map<string, string>;
/** Builds a property-type map from ontology entity metadata. */
export declare function propertyTypesOf(entity: OntologyObjectType): PropertyTypes;
/**
 * Hydrates a stringly-typed result into native-typed row objects. Ontology
 * property metadata, when supplied, overrides the wire column type — the
 * ontology's dataType is more precise than the storage column type.
 */
export declare function hydrateRows(result: DatasetQueryResult | RowsResult, propertyTypes?: PropertyTypes): {
    rows: Row[];
    totalCount: number | null;
};
