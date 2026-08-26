package com.clidey.whodb;

import com.clidey.whodb.gen.Operations;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * The {@code client.ontology("User")} facade: reads and record writes for one
 * ontology entity, addressed by apiName. Entity metadata is fetched once per
 * handle and reused for pk lookups and row hydration.
 */
public final class OntologyHandle {
    static final int DEFAULT_PAGE_SIZE = 100;
    private static final ObjectMapper JSON = new ObjectMapper();

    private final WhoDB client;
    private final String apiName;
    private Map<String, Object> entity;

    /** Options for {@link #list(ListOptions)}: filter, sort, and paging. */
    public record ListOptions(Map<String, Object> where, List<Map<String, Object>> sort, int pageSize, int offset) {
        /** Options with no filter and default paging. */
        public static ListOptions defaults() {
            return new ListOptions(null, null, 0, 0);
        }
    }

    OntologyHandle(WhoDB client, String apiName) {
        this.client = client;
        this.apiName = apiName;
    }

    /** Resolves and caches the entity metadata backing this handle. */
    @SuppressWarnings("unchecked")
    public synchronized Map<String, Object> entityMeta() {
        if (entity != null) {
            return entity;
        }
        ManifestCheck.warnIfFlagged("OntologyEntities");
        Object result = client.execute(Operations.ontologyEntitiesRequest(Map.of("projectId", client.projectId())));
        if (result instanceof List<?> entities) {
            for (Object entry : entities) {
                Map<String, Object> candidate = (Map<String, Object>) entry;
                if (apiName.equals(candidate.get("apiName"))) {
                    entity = candidate;
                    return entity;
                }
            }
        }
        throw new WhoDBException(WhoDBException.Kind.NOT_FOUND,
            "ontology entity \"" + apiName + "\" not found in this project");
    }

    private String entityId() {
        return String.valueOf(entityMeta().getOrDefault("id", ""));
    }

    private String primaryKey() {
        Object pk = entityMeta().get("primaryKey");
        return pk == null ? "" : pk.toString();
    }

    /** Describes the entity: schema, properties, links. */
    @SuppressWarnings("unchecked")
    public Map<String, Object> describe() {
        entityMeta(); // NOT_FOUND for unknown entities
        ManifestCheck.warnIfFlagged("OntologyDescribe");
        Object result = client.execute(Operations.ontologyDescribeRequest(Map.of(
            "projectId", client.projectId(),
            "input", Map.of("entities", List.of(apiName), "includeInferred", true))));
        return result instanceof Map<?, ?> ? (Map<String, Object>) result : Map.of();
    }

    @SuppressWarnings("unchecked")
    private Map<String, Object> query(Map<String, Object> input) {
        ManifestCheck.warnIfFlagged("OntologyQuery");
        input.put("entity", apiName);
        Object result = client.execute(Operations.ontologyQueryRequest(Map.of(
            "projectId", client.projectId(),
            "input", input)));
        return result instanceof Map<?, ?> ? (Map<String, Object>) result : Map.of();
    }

    private String toJson(Object value) {
        try {
            return JSON.writeValueAsString(value);
        } catch (Exception error) {
            throw new WhoDBException(WhoDBException.Kind.VALIDATION, "failed to encode filter as JSON");
        }
    }

    /** Fetches a single record by primary key; null when absent. */
    public Map<String, Object> get(Object pk) {
        String primaryKey = primaryKey();
        if (primaryKey.isEmpty()) {
            throw new WhoDBException(WhoDBException.Kind.VALIDATION,
                "entity \"" + apiName + "\" has no primary key — use list with a where filter");
        }
        Map<String, Object> input = new HashMap<>();
        input.put("whereJson", toJson(Map.of(primaryKey, Map.of("eq", String.valueOf(pk)))));
        input.put("pageSize", 1);
        input.put("offset", 0);
        Map<String, Object> result = query(input);
        List<Map<String, Object>> rows = Hydrator.hydrateRows(result, Hydrator.propertyTypesOf(entityMeta()));
        return rows.isEmpty() ? null : rows.get(0);
    }

    /** Returns one page of records with optional filter/sort. */
    public List<Map<String, Object>> list(ListOptions options) {
        Map<String, Object> meta = entityMeta();
        int pageSize = options.pageSize() == 0 ? DEFAULT_PAGE_SIZE : options.pageSize();
        Map<String, Object> input = new HashMap<>();
        input.put("pageSize", pageSize);
        input.put("offset", options.offset());
        if (options.where() != null) {
            input.put("whereJson", toJson(options.where()));
        }
        if (options.sort() != null) {
            input.put("sort", options.sort());
        }
        Map<String, Object> result = query(input);
        return Hydrator.hydrateRows(result, Hydrator.propertyTypesOf(meta));
    }

    /**
     * Walks every page, collecting them until a short page ends the result
     * set.
     */
    public List<List<Map<String, Object>>> pages(ListOptions options) {
        int pageSize = options.pageSize() == 0 ? DEFAULT_PAGE_SIZE : options.pageSize();
        List<List<Map<String, Object>>> pages = new ArrayList<>();
        int offset = options.offset();
        while (true) {
            List<Map<String, Object>> rows =
                list(new ListOptions(options.where(), options.sort(), pageSize, offset));
            pages.add(rows);
            if (rows.size() < pageSize) {
                return pages;
            }
            offset += pageSize;
        }
    }

    /**
     * Converts a data map to GraphQL RecordInput pairs, JSON-encoding objects
     * and lowercasing booleans (cross-language behavior).
     */
    private List<Map<String, String>> toRecordInputs(Map<String, Object> values) {
        List<Map<String, String>> records = new ArrayList<>(values.size());
        for (Map.Entry<String, Object> entry : values.entrySet()) {
            Object value = entry.getValue();
            String encoded;
            if (value == null) {
                encoded = "";
            } else if (value instanceof Boolean flag) {
                encoded = flag ? "true" : "false";
            } else if (value instanceof String text) {
                encoded = text;
            } else if (value instanceof Map || value instanceof List) {
                encoded = toJson(value);
            } else {
                encoded = String.valueOf(value);
            }
            Map<String, String> record = new LinkedHashMap<>();
            record.put("Key", entry.getKey());
            record.put("Value", encoded);
            records.add(record);
        }
        return records;
    }

    /** Inserts one record. Values are field name/value pairs. */
    public void create(Map<String, Object> values) {
        String entityId = entityId();
        ManifestCheck.warnIfFlagged("OntologyAddRow");
        client.execute(Operations.ontologyAddRowRequest(Map.of(
            "projectId", client.projectId(),
            "entityId", entityId,
            "values", toRecordInputs(values))));
    }

    /** Inserts many records; a non-empty idempotencyKey makes retries safe. */
    @SuppressWarnings("unchecked")
    public Map<String, Object> createMany(List<Map<String, Object>> rows, String idempotencyKey) {
        String entityId = entityId();
        ManifestCheck.warnIfFlagged("OntologyAddRows");
        List<Map<String, Object>> wireRows = new ArrayList<>(rows.size());
        for (Map<String, Object> row : rows) {
            wireRows.add(Map.of("values", toRecordInputs(row)));
        }
        Map<String, Object> variables = new HashMap<>();
        variables.put("projectId", client.projectId());
        variables.put("entityId", entityId);
        variables.put("rows", wireRows);
        if (idempotencyKey != null && !idempotencyKey.isEmpty()) {
            variables.put("idempotencyKey", idempotencyKey);
        }
        Object result = client.execute(Operations.ontologyAddRowsRequest(variables));
        return result instanceof Map<?, ?> ? (Map<String, Object>) result : Map.of();
    }

    /** Updates one record identified by primary key. */
    public void update(Object pk, Map<String, Object> values) {
        String primaryKey = primaryKey();
        if (primaryKey.isEmpty()) {
            throw new WhoDBException(WhoDBException.Kind.VALIDATION,
                "entity \"" + apiName + "\" has no primary key — updates are not supported");
        }
        String entityId = entityId();
        ManifestCheck.warnIfFlagged("OntologyUpdateRow");
        Map<String, Object> merged = new LinkedHashMap<>(values);
        List<String> updatedColumns = new ArrayList<>(values.keySet());
        merged.put(primaryKey, String.valueOf(pk));
        client.execute(Operations.ontologyUpdateRowRequest(Map.of(
            "projectId", client.projectId(),
            "entityId", entityId,
            "values", toRecordInputs(merged),
            "updatedColumns", updatedColumns)));
    }

    /** Deletes one record identified by primary key. */
    public void delete(Object pk) {
        String primaryKey = primaryKey();
        if (primaryKey.isEmpty()) {
            throw new WhoDBException(WhoDBException.Kind.VALIDATION,
                "entity \"" + apiName + "\" has no primary key — deletes are not supported");
        }
        String entityId = entityId();
        ManifestCheck.warnIfFlagged("OntologyDeleteRow");
        client.execute(Operations.ontologyDeleteRowRequest(Map.of(
            "projectId", client.projectId(),
            "entityId", entityId,
            "values", toRecordInputs(Map.of(primaryKey, String.valueOf(pk))))));
    }

    /** Follows an outgoing link from one record to its related records. */
    @SuppressWarnings("unchecked")
    public List<Map<String, Object>> followLink(Object pk, String linkApiName, int pageSize, int offset) {
        String entityId = entityId();
        ManifestCheck.warnIfFlagged("OntologyFollowLink");
        Object result = client.execute(Operations.ontologyFollowLinkRequest(Map.of(
            "projectId", client.projectId(),
            "entityId", entityId,
            "pk", String.valueOf(pk),
            "linkApiName", linkApiName,
            "pageSize", pageSize == 0 ? DEFAULT_PAGE_SIZE : pageSize,
            "pageOffset", offset)));
        Map<String, Object> wire = result instanceof Map<?, ?> ? (Map<String, Object>) result : Map.of();
        return Hydrator.hydrateRows(wire, null);
    }
}
