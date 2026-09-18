package com.clidey.whodb;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertInstanceOf;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertSame;

import java.time.OffsetDateTime;
import java.util.List;
import java.util.Map;
import org.junit.jupiter.api.Test;

/** Unit tests mirroring the Go/Rust SDK test coverage. */
class WhoDBTest {

    private static List<Object> graphQLError(String code) {
        return List.of(Map.of("message", "x", "extensions", Map.of("code", code)));
    }

    @Test
    void mapsGraphQLErrorCodesToTaxonomy() {
        assertEquals(WhoDBException.Kind.AUTH, WhoDBException.fromGraphQLErrors(graphQLError("UNAUTHENTICATED")).kind());
        assertEquals(WhoDBException.Kind.AUTH, WhoDBException.fromGraphQLErrors(graphQLError("FORBIDDEN")).kind());
        assertEquals(WhoDBException.Kind.NOT_FOUND, WhoDBException.fromGraphQLErrors(graphQLError("NOT_FOUND")).kind());
        assertEquals(WhoDBException.Kind.VALIDATION, WhoDBException.fromGraphQLErrors(graphQLError("BAD_USER_INPUT")).kind());
        WhoDBException platform = WhoDBException.fromGraphQLErrors(graphQLError("OTHER"));
        assertEquals(WhoDBException.Kind.PLATFORM, platform.kind());
        assertEquals("OTHER", platform.code());
        assertEquals(WhoDBException.Kind.PLATFORM, WhoDBException.fromGraphQLErrors(List.of()).kind());
    }

    @Test
    void interpretsUnknownOperationAsVersionError() {
        WhoDBException unknownField = new WhoDBException(WhoDBException.Kind.PLATFORM,
            "Cannot query field \"New\" on type \"Query\"");
        assertEquals(WhoDBException.Kind.VERSION, WhoDBException.interpretServerError(unknownField).kind());
        WhoDBException other = new WhoDBException(WhoDBException.Kind.TRANSPORT, "connection refused");
        assertSame(other, WhoDBException.interpretServerError(other));
    }

    @Test
    void coercesValuesPerSharedRules() {
        assertEquals(42L, Hydrator.coerceValue("42", "bigint"));
        assertEquals(3.14, Hydrator.coerceValue("3.14", "numeric"));
        assertEquals(true, Hydrator.coerceValue("true", "boolean"));
        assertEquals(false, Hydrator.coerceValue("f", "boolean"));
        assertInstanceOf(OffsetDateTime.class, Hydrator.coerceValue("2026-01-05T00:00:00Z", "timestamptz"));
        assertEquals(Map.of("a", 1), Hydrator.coerceValue("{\"a\": 1}", "jsonb"));
        assertEquals("hello", Hydrator.coerceValue("hello", "text"));
        assertNull(Hydrator.coerceValue(null, "bigint"));
        assertEquals("not-a-number", Hydrator.coerceValue("not-a-number", "bigint"));
        // Non-string values pass through (IPC delivers native types).
        assertEquals(7.0, Hydrator.coerceValue(7.0, "bigint"));
    }

    @Test
    void propertyMetadataOverridesColumnType() {
        Map<String, Object> result = Map.of(
            "columns", List.of("age"),
            "rows", List.of(List.of("42")),
            "total", 1);
        List<Map<String, Object>> typed = Hydrator.hydrateRows(result, Map.of("age", "Integer"));
        assertEquals(42L, typed.get(0).get("age"));
        List<Map<String, Object>> untyped = Hydrator.hydrateRows(result, null);
        assertEquals("42", untyped.get(0).get("age"));
    }

    @Test
    void hydratesPascalCaseRowsResult() {
        Map<String, Object> result = Map.of(
            "Columns", List.of(Map.of("Name", "n", "Type", "int4")),
            "Rows", List.of(List.of("7")),
            "TotalCount", 1);
        List<Map<String, Object>> rows = Hydrator.hydrateRows(result, null);
        assertEquals(7L, rows.get(0).get("n"));
    }
}
