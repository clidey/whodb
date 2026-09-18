package com.clidey.whodb;

import com.clidey.whodb.gen.Hydration;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.time.OffsetDateTime;
import java.time.format.DateTimeParseException;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;

/**
 * Row hydration: stringly-typed wire results → native-typed rows. Behavior is
 * pinned by the shared cross-language conformance fixtures.
 */
final class Hydrator {
    private static final ObjectMapper JSON = new ObjectMapper();

    private Hydrator() {}

    /**
     * Coerces one stringly-typed cell into its native type per the shared
     * rules. Non-string values pass through unchanged (the IPC transport
     * delivers natively-typed values). Timestamps hydrate to
     * {@link OffsetDateTime}.
     */
    static Object coerceValue(Object raw, String columnType) {
        if (raw == null) {
            return null;
        }
        if (!(raw instanceof String text)) {
            return raw;
        }
        String kind = Hydration.RULES.getOrDefault(columnType.toLowerCase(Locale.ROOT), Hydration.DEFAULT_KIND);
        switch (kind) {
            case "int":
                try {
                    return Long.parseLong(text);
                } catch (NumberFormatException ignored) {
                    return text;
                }
            case "float":
                try {
                    return Double.parseDouble(text);
                } catch (NumberFormatException ignored) {
                    return text;
                }
            case "bool":
                return text.equals("true") || text.equals("t") || text.equals("1");
            case "timestamp":
            case "date":
                try {
                    String normalized = text.length() == 10 ? text + "T00:00:00Z" : text;
                    return OffsetDateTime.parse(normalized);
                } catch (DateTimeParseException ignored) {
                    return text;
                }
            case "json":
                try {
                    return JSON.readValue(text, Object.class);
                } catch (Exception ignored) {
                    return text;
                }
            default:
                return text;
        }
    }

    private record WireColumn(String name, String type) {}

    /** Builds a property-type map (apiName → dataType) from entity metadata. */
    @SuppressWarnings("unchecked")
    static Map<String, String> propertyTypesOf(Map<String, Object> entity) {
        Map<String, String> types = new LinkedHashMap<>();
        Object properties = entity.get("properties");
        if (properties instanceof List<?> list) {
            for (Object property : list) {
                if (property instanceof Map<?, ?> propertyMap) {
                    Object apiName = propertyMap.get("apiName");
                    Object dataType = propertyMap.get("dataType");
                    if (apiName != null && dataType != null) {
                        types.put(apiName.toString(), dataType.toString());
                    }
                }
            }
        }
        return types;
    }

    /**
     * Hydrates a wire result into native-typed rows. Ontology property
     * metadata, when supplied, overrides the wire column type. Normalizes
     * both DatasetQueryResult (lowercase, names-only columns) and CE-derived
     * RowsResult (PascalCase, typed columns).
     */
    @SuppressWarnings("unchecked")
    static List<Map<String, Object>> hydrateRows(Map<String, Object> result, Map<String, String> propertyTypes) {
        List<WireColumn> columns = new ArrayList<>();
        List<Object> rawRows;
        if (result.get("columns") instanceof List<?> names) {
            for (Object name : names) {
                columns.add(new WireColumn(String.valueOf(name), ""));
            }
            rawRows = (List<Object>) result.getOrDefault("rows", List.of());
        } else {
            Object rawColumns = result.getOrDefault("Columns", List.of());
            for (Object column : (List<Object>) rawColumns) {
                Map<String, Object> columnMap = (Map<String, Object>) column;
                columns.add(new WireColumn(
                    String.valueOf(columnMap.getOrDefault("Name", "")),
                    String.valueOf(columnMap.getOrDefault("Type", ""))));
            }
            rawRows = (List<Object>) result.getOrDefault("Rows", List.of());
        }
        List<Map<String, Object>> rows = new ArrayList<>(rawRows.size());
        for (Object rawRow : rawRows) {
            List<Object> cells = rawRow instanceof List<?> list ? (List<Object>) list : List.of();
            Map<String, Object> row = new LinkedHashMap<>();
            for (int index = 0; index < columns.size(); index++) {
                WireColumn column = columns.get(index);
                String columnType = propertyTypes != null
                    ? propertyTypes.getOrDefault(column.name(), column.type())
                    : column.type();
                Object cell = index < cells.size() ? cells.get(index) : null;
                row.put(column.name(), coerceValue(cell, columnType));
            }
            rows.add(row);
        }
        return rows;
    }
}
