package com.clidey.whodb.conformance;

import com.clidey.whodb.OntologyHandle;
import com.clidey.whodb.Transport;
import com.clidey.whodb.WhoDB;
import com.clidey.whodb.WhoDBException;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.time.OffsetDateTime;
import java.time.ZoneOffset;
import java.time.format.DateTimeFormatter;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.TreeMap;

/**
 * The Java SDK fixture runner: reads fixtures as JSON lines on stdin,
 * executes each against the SDK with a mock transport, and writes one JSON
 * result line per fixture to stdout. Protocol shared with the other language
 * runners.
 */
public final class Main {
    private static final ObjectMapper JSON = new ObjectMapper();

    private Main() {}

    /** Mock transport replaying a fixture's scripted transcript. */
    private static final class MockTransport implements Transport {
        private final JsonNode steps;
        private int index;

        MockTransport(JsonNode steps) {
            this.steps = steps;
        }

        @Override
        @SuppressWarnings("unchecked")
        public Map<String, Object> execute(String operation, String document, Map<String, Object> variables) {
            if (index >= steps.size()) {
                throw new RuntimeException("unexpected call #" + (index + 1) + ": " + operation);
            }
            JsonNode step = steps.get(index++);
            JsonNode expect = step.path("expectRequest");
            String expectedOperation = expect.path("operation").asText("");
            if (!expectedOperation.isEmpty() && !expectedOperation.equals(operation)) {
                throw new RuntimeException("expected operation " + expectedOperation + ", got " + operation);
            }
            assertContains(variables, expect.path("variables"), "variables");
            assertContains(variables, expect.path("variablesContain"), "variables");
            JsonNode inputContains = expect.path("inputContains");
            if (inputContains.isObject() && inputContains.size() > 0) {
                Object input = variables.get("input");
                Map<String, Object> inputMap = input instanceof Map<?, ?> map ? (Map<String, Object>) map : Map.of();
                assertContains(inputMap, inputContains, "input");
            }
            JsonNode errors = step.path("response").path("errors");
            if (errors.isArray() && errors.size() > 0) {
                List<Object> converted = JSON.convertValue(errors, List.class);
                throw WhoDBException.fromGraphQLErrors(converted);
            }
            return JSON.convertValue(step.path("response").path("data"), Map.class);
        }

        private void assertContains(Map<String, Object> actual, JsonNode expected, String prefix) {
            if (!expected.isObject()) {
                return;
            }
            expected.fields().forEachRemaining(entry -> {
                String want = canonicalJson(JSON.convertValue(entry.getValue(), Object.class));
                String got = canonicalJson(actual.get(entry.getKey()));
                if (!want.equals(got)) {
                    throw new RuntimeException("mismatch at " + prefix + "." + entry.getKey()
                        + ": expected " + want + ", got " + got);
                }
            });
        }
    }

    /** Serializes a value with sorted object keys for order-independent comparison. */
    private static String canonicalJson(Object value) {
        try {
            return JSON.writeValueAsString(sortKeys(value));
        } catch (Exception error) {
            return String.valueOf(value);
        }
    }

    @SuppressWarnings("unchecked")
    private static Object sortKeys(Object value) {
        if (value instanceof Map<?, ?> map) {
            Map<String, Object> sorted = new TreeMap<>();
            for (Map.Entry<?, ?> entry : map.entrySet()) {
                sorted.put(String.valueOf(entry.getKey()), sortKeys(entry.getValue()));
            }
            return sorted;
        }
        if (value instanceof List<?> list) {
            List<Object> out = new ArrayList<>(list.size());
            for (Object item : list) {
                out.add(sortKeys(item));
            }
            return out;
        }
        if (value instanceof Integer number) {
            return number.longValue(); // normalize int vs long across JSON parses
        }
        return value;
    }

    /**
     * Serializes results for comparison; OffsetDateTime becomes
     * "@date:&lt;ISO-8601 Z&gt;" per the shared protocol.
     */
    private static Object canonical(Object value) {
        if (value instanceof OffsetDateTime timestamp) {
            return "@date:" + timestamp.withOffsetSameInstant(ZoneOffset.UTC)
                .format(DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm:ss'Z'"));
        }
        if (value instanceof Map<?, ?> map) {
            Map<String, Object> out = new LinkedHashMap<>();
            for (Map.Entry<?, ?> entry : map.entrySet()) {
                out.put(String.valueOf(entry.getKey()), canonical(entry.getValue()));
            }
            return out;
        }
        if (value instanceof List<?> list) {
            List<Object> out = new ArrayList<>(list.size());
            for (Object item : list) {
                out.add(canonical(item));
            }
            return out;
        }
        return value;
    }

    /** Maps a WhoDBException to the shared taxonomy names used in fixtures. */
    private static String errorType(Throwable error) {
        if (error instanceof WhoDBException whodbError) {
            return switch (whodbError.kind()) {
                case AUTH -> "AuthError";
                case NOT_FOUND -> "NotFoundError";
                case VALIDATION -> "ValidationError";
                case VERSION -> "WhoDBVersionError";
                case CLI_CREDENTIALS -> "CliCredentialsError";
                case TRANSPORT_CAPABILITY -> "TransportCapabilityError";
                default -> "PlatformError";
            };
        }
        return "PlatformError";
    }

    @SuppressWarnings("unchecked")
    private static Map<String, Object> runFixture(JsonNode fixture) {
        String name = fixture.path("name").asText();
        Map<String, Object> result = new LinkedHashMap<>();
        result.put("name", name);
        JsonNode call = fixture.path("call");
        if (!"ontology".equals(call.path("domain").asText())) {
            result.put("pass", false);
            result.put("reason", "unsupported fixture domain: " + call.path("domain").asText());
            return result;
        }
        MockTransport transport = new MockTransport(fixture.path("transcript"));
        WhoDB client = new WhoDB(new WhoDB.Config(null, null, null,
            "00000000-0000-0000-0000-000000000001", "proj-1", null, transport));
        OntologyHandle handle = client.ontology(call.path("handle").asText());
        JsonNode args = call.path("args");

        Object value = null;
        Throwable callError = null;
        try {
            switch (call.path("method").asText()) {
                case "get" -> value = handle.get(JSON.convertValue(args.get(0), Object.class));
                case "describe" -> value = handle.describe();
                case "list" -> {
                    JsonNode options = args.size() > 0 ? args.get(0) : JSON.createObjectNode();
                    Map<String, Object> where = options.has("where")
                        ? JSON.convertValue(options.get("where"), Map.class) : null;
                    OntologyHandle.ListOptions listOptions = new OntologyHandle.ListOptions(
                        where, null, options.path("pageSize").asInt(0), 0);
                    if ("pages".equals(call.path("collect").asText())) {
                        value = handle.pages(listOptions);
                    } else {
                        value = handle.list(listOptions);
                    }
                }
                case "create" -> handle.create(JSON.convertValue(args.get(0), Map.class));
                case "createMany" -> {
                    List<Map<String, Object>> rows = JSON.convertValue(args.get(0), List.class);
                    String idempotencyKey = args.size() > 1 ? args.get(1).path("idempotencyKey").asText("") : "";
                    value = handle.createMany(rows, idempotencyKey);
                }
                case "update" -> handle.update(
                    JSON.convertValue(args.get(0), Object.class), JSON.convertValue(args.get(1), Map.class));
                case "delete" -> handle.delete(JSON.convertValue(args.get(0), Object.class));
                default -> {
                    result.put("pass", false);
                    result.put("reason", "unsupported fixture method: " + call.path("method").asText());
                    return result;
                }
            }
        } catch (Throwable error) {
            callError = error;
        }

        JsonNode expectError = fixture.path("expectError");
        if (callError != null) {
            if (!expectError.isMissingNode() && !expectError.isNull()) {
                String gotType = errorType(callError);
                String wantType = expectError.path("type").asText();
                String messageContains = expectError.path("messageContains").asText("");
                String message = callError.getMessage() == null ? "" : callError.getMessage();
                if (gotType.equals(wantType) && message.contains(messageContains)) {
                    result.put("pass", true);
                } else {
                    result.put("pass", false);
                    result.put("reason", "expected " + wantType + "(" + messageContains + "), got "
                        + gotType + ": " + message);
                }
            } else {
                result.put("pass", false);
                result.put("reason", callError.toString());
            }
            return result;
        }
        if (!expectError.isMissingNode() && !expectError.isNull()) {
            result.put("pass", false);
            result.put("reason", "expected " + expectError.path("type").asText() + ", got success");
            return result;
        }

        Object want = fixture.has("expectResult")
            ? JSON.convertValue(fixture.get("expectResult"), Object.class) : null;
        Object got = canonical(value);
        if (!canonicalJson(got).equals(canonicalJson(want))) {
            result.put("pass", false);
            result.put("reason", "result mismatch:\n  want " + canonicalJson(want) + "\n  got  " + canonicalJson(got));
        } else {
            result.put("pass", true);
        }
        return result;
    }

    /** Entry point: JSON-lines fixtures on stdin → JSON-lines results on stdout. */
    public static void main(String[] args) throws Exception {
        BufferedReader reader = new BufferedReader(new InputStreamReader(System.in, StandardCharsets.UTF_8));
        String line;
        while ((line = reader.readLine()) != null) {
            line = line.trim();
            if (line.isEmpty()) {
                continue;
            }
            Map<String, Object> result;
            try {
                result = runFixture(JSON.readTree(line));
            } catch (Exception error) {
                result = Map.of("name", "(parse error)", "pass", false, "reason", String.valueOf(error));
            }
            System.out.println(JSON.writeValueAsString(result));
            System.out.flush();
        }
    }
}
