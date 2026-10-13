/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package com.clidey.whodb.conformance;

import com.clidey.whodb.DatasetHandle;
import com.clidey.whodb.FunctionHandle;
import com.clidey.whodb.FunctionsHandle;
import com.clidey.whodb.LineageProvidersHandle;
import com.clidey.whodb.OntologyHandle;
import com.clidey.whodb.RowsPage;
import com.clidey.whodb.ServiceAccountsHandle;
import com.clidey.whodb.SharingHandle;
import com.clidey.whodb.SourceHandle;
import com.clidey.whodb.TransformHandle;
import com.clidey.whodb.TransformsHandle;
import com.clidey.whodb.Transport;
import com.clidey.whodb.WaitOptions;
import com.clidey.whodb.WhoDB;
import com.clidey.whodb.WhoDBException;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
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
        MockTransport transport = new MockTransport(fixture.path("transcript"));
        WhoDB client = new WhoDB(new WhoDB.Config(null, null, null,
            "00000000-0000-0000-0000-000000000001", "proj-1", null, transport));
        String domain = call.path("domain").asText();
        String handleName = call.path("handle").asText("");
        String method = call.path("method").asText();
        JsonNode args = call.path("args");

        Object value = null;
        Throwable callError = null;
        try {
            if ("ontology".equals(domain)) {
                value = runOntology(client, handleName, method, args, "pages".equals(call.path("collect").asText()));
            } else {
                value = runOther(client, domain, handleName, method, args);
            }
        } catch (UnsupportedFixture unsupported) {
            result.put("pass", false);
            result.put("reason", unsupported.getMessage());
            return result;
        } catch (Throwable error) {
            callError = error;
        }
        return compare(fixture, result, value, callError);
    }

    /** Signals a fixture call the Java SDK does not model. */
    private static final class UnsupportedFixture extends RuntimeException {
        UnsupportedFixture(String message) {
            super(message);
        }
    }

    private static String str(JsonNode args, int index) {
        return args.size() > index && !args.get(index).isNull() ? args.get(index).asText() : null;
    }

    @SuppressWarnings("unchecked")
    private static Map<String, Object> map(JsonNode args, int index) {
        return args.size() > index && args.get(index).isObject()
            ? JSON.convertValue(args.get(index), Map.class) : new LinkedHashMap<>();
    }

    @SuppressWarnings("unchecked")
    private static List<String> strings(JsonNode args, int index) {
        return args.size() > index && args.get(index).isArray() ? JSON.convertValue(args.get(index), List.class) : null;
    }

    private static Object any(JsonNode args, int index) {
        return args.size() > index ? JSON.convertValue(args.get(index), Object.class) : null;
    }

    private static WaitOptions waitOptions(Map<String, Object> options) {
        WaitOptions wait = WaitOptions.defaults();
        if (options.get("timeoutMs") instanceof Number timeout) {
            wait = new WaitOptions(Duration.ofMillis(timeout.longValue()), wait.poll());
        }
        if (options.get("pollMs") instanceof Number poll) {
            wait = new WaitOptions(wait.timeout(), Duration.ofMillis(poll.longValue()));
        }
        return wait;
    }

    private static int intOf(Map<String, Object> options, String key) {
        return options.get(key) instanceof Number number ? number.intValue() : 0;
    }

    private static Map<String, Object> pageValue(RowsPage page) {
        Map<String, Object> out = new LinkedHashMap<>();
        out.put("rows", page.rows());
        out.put("totalCount", page.totalCount());
        return out;
    }

    @SuppressWarnings("unchecked")
    private static Object runOther(WhoDB client, String domain, String handleName, String method, JsonNode args) {
        switch (domain) {
            case "dataset" -> {
                DatasetHandle handle = client.dataset(handleName);
                Map<String, Object> options = map(args, 0);
                return switch (method) {
                    case "meta" -> handle.meta();
                    case "rows" -> pageValue(handle.rows(
                        new DatasetHandle.RowsOptions(intOf(options, "pageSize"), intOf(options, "pageOffset"))));
                    default -> throw new UnsupportedFixture("unsupported fixture call: " + domain + "." + method);
                };
            }
            case "source" -> {
                SourceHandle handle = client.source(handleName);
                return switch (method) {
                    case "objects" -> {
                        Map<String, Object> options = map(args, 0);
                        yield handle.objects(new SourceHandle.ObjectsOptions(
                            (Map<String, Object>) options.get("parent"), intOf(options, "pageSize"), intOf(options, "pageOffset")));
                    }
                    case "columns" -> handle.columns(map(args, 0));
                    case "rows" -> {
                        Map<String, Object> options = map(args, 1);
                        yield handle.rows(map(args, 0), new SourceHandle.RowsOptions(
                            (Map<String, Object>) options.get("where"), (List<Map<String, Object>>) options.get("sort"),
                            intOf(options, "pageSize"), intOf(options, "offset")));
                    }
                    default -> throw new UnsupportedFixture("unsupported fixture call: " + domain + "." + method);
                };
            }
            case "transforms" -> {
                TransformsHandle handle = client.transforms();
                return switch (method) {
                    case "list" -> handle.list();
                    case "create" -> handle.create(map(args, 0));
                    case "preview" -> pageValue(handle.preview(str(args, 0)));
                    default -> throw new UnsupportedFixture("unsupported fixture call: " + domain + "." + method);
                };
            }
            case "transform" -> {
                TransformHandle handle = client.transform(handleName);
                switch (method) {
                    case "meta": return handle.meta();
                    case "runs": return handle.runs(intOf(map(args, 0), "limit"));
                    case "run": return handle.run();
                    case "cancelRun": handle.cancelRun(str(args, 0)); return null;
                    case "wait": return handle.wait(str(args, 0), waitOptions(map(args, 1)));
                    case "update": return handle.update(map(args, 0));
                    case "delete": handle.delete(); return null;
                    default: throw new UnsupportedFixture("unsupported fixture call: " + domain + "." + method);
                }
            }
            case "functions" -> {
                FunctionsHandle handle = client.functions();
                return switch (method) {
                    case "list" -> handle.list();
                    case "create" -> handle.create(map(args, 0));
                    default -> throw new UnsupportedFixture("unsupported fixture call: " + domain + "." + method);
                };
            }
            case "function" -> {
                FunctionHandle handle = client.function(handleName);
                switch (method) {
                    case "meta": return handle.meta();
                    case "execute": return handle.execute(any(args, 0), strings(args, 1));
                    case "test": {
                        Map<String, Object> options = map(args, 1);
                        return handle.test(any(args, 0), new FunctionHandle.TestOptions(
                            (List<Map<String, Object>>) options.get("files"), (List<String>) options.get("inputFileIds")));
                    }
                    case "startRun": {
                        Object flag = map(args, 1).get("useActiveVersion");
                        return handle.startRun(any(args, 0), flag instanceof Boolean bool ? bool : null);
                    }
                    case "run": return handle.run(str(args, 0));
                    case "runs": return handle.runs(intOf(map(args, 0), "limit"));
                    case "wait": return handle.wait(str(args, 0), waitOptions(map(args, 1)));
                    case "cancelRun": return handle.cancelRun(str(args, 0));
                    case "deploy": handle.deploy(); return null;
                    case "redeploy": handle.redeploy(); return null;
                    case "update": return handle.update(map(args, 0));
                    case "delete": handle.delete(); return null;
                    default: throw new UnsupportedFixture("unsupported fixture call: " + domain + "." + method);
                }
            }
            case "serviceAccounts" -> {
                ServiceAccountsHandle handle = client.serviceAccounts();
                switch (method) {
                    case "list": return handle.list();
                    case "create": return handle.create(str(args, 0), str(args, 1));
                    case "createKey": return handle.createKey(str(args, 0), str(args, 1));
                    case "revokeKey": handle.revokeKey(str(args, 0)); return null;
                    case "delete": handle.delete(str(args, 0)); return null;
                    default: throw new UnsupportedFixture("unsupported fixture call: " + domain + "." + method);
                }
            }
            case "lineageProviders" -> {
                LineageProvidersHandle handle = client.lineageProviders();
                switch (method) {
                    case "list": return handle.list();
                    case "save": return handle.save(map(args, 0));
                    case "delete": handle.delete(str(args, 0)); return null;
                    default: throw new UnsupportedFixture("unsupported fixture call: " + domain + "." + method);
                }
            }
            case "sharing" -> {
                SharingHandle handle = client.sharing();
                switch (method) {
                    case "shared": return handle.shared();
                    case "discover": return handle.discover();
                    case "references": return handle.references();
                    case "referenceImpact": return handle.referenceImpact(str(args, 0));
                    case "share": return handle.share(str(args, 0), str(args, 1), strings(args, 2));
                    case "unshare": handle.unshare(str(args, 0)); return null;
                    case "createReference": return handle.createReference(str(args, 0), str(args, 1), str(args, 2));
                    case "removeReference": handle.removeReference(str(args, 0)); return null;
                    default: throw new UnsupportedFixture("unsupported fixture call: " + domain + "." + method);
                }
            }
            default -> throw new UnsupportedFixture("unsupported fixture domain: " + domain);
        }
    }

    @SuppressWarnings("unchecked")
    private static Object runOntology(WhoDB client, String handleName, String method, JsonNode args, boolean collectPages) {
        OntologyHandle handle = client.ontology(handleName);
        Object value = null;
        {
            switch (method) {
                case "get" -> value = handle.get(JSON.convertValue(args.get(0), Object.class));
                case "describe" -> value = handle.describe();
                case "list" -> {
                    JsonNode options = args.size() > 0 ? args.get(0) : JSON.createObjectNode();
                    Map<String, Object> where = options.has("where")
                        ? JSON.convertValue(options.get("where"), Map.class) : null;
                    OntologyHandle.ListOptions listOptions = new OntologyHandle.ListOptions(
                        where, null, options.path("pageSize").asInt(0), 0);
                    if (collectPages) {
                        value = handle.pages(listOptions);
                    } else {
                        value = handle.list(listOptions);
                    }
                }
                case "capabilities" -> value = handle.capabilities(
                    args.size() > 0 && !args.get(0).isNull() ? JSON.convertValue(args.get(0), Object.class) : null);
                case "previewAction" -> value = handle.previewAction(
                    args.get(0).asText(),
                    args.get(1).isNull() ? null : JSON.convertValue(args.get(1), Object.class),
                    JSON.convertValue(args.get(2), Map.class));
                case "action" -> {
                    JsonNode options = args.size() > 3 ? args.get(3) : JSON.createObjectNode();
                    Integer expectedVersion = options.hasNonNull("expectedVersion")
                        ? options.get("expectedVersion").asInt() : null;
                    String idempotencyKey = options.hasNonNull("idempotencyKey")
                        ? options.get("idempotencyKey").asText() : null;
                    value = handle.action(
                        args.get(0).asText(),
                        args.get(1).isNull() ? null : JSON.convertValue(args.get(1), Object.class),
                        JSON.convertValue(args.get(2), Map.class),
                        new OntologyHandle.ActionOptions(expectedVersion, idempotencyKey));
                }
                case "actionExecutions" -> value = handle.actionExecutions(
                    JSON.convertValue(args.get(0), Object.class),
                    args.size() > 1 ? args.get(1).asInt() : 50);
                case "create" -> handle.create(JSON.convertValue(args.get(0), Map.class));
                case "createMany" -> {
                    List<Map<String, Object>> rows = JSON.convertValue(args.get(0), List.class);
                    String idempotencyKey = args.size() > 1 ? args.get(1).path("idempotencyKey").asText("") : "";
                    value = handle.createMany(rows, idempotencyKey);
                }
                case "update" -> handle.update(
                    JSON.convertValue(args.get(0), Object.class), JSON.convertValue(args.get(1), Map.class));
                case "delete" -> handle.delete(JSON.convertValue(args.get(0), Object.class));
                case "followLink" -> {
                    JsonNode options = args.size() > 2 ? args.get(2) : JSON.createObjectNode();
                    value = handle.followLink(
                        JSON.convertValue(args.get(0), Object.class),
                        args.get(1).asText(),
                        options.path("pageSize").asInt(0),
                        options.path("offset").asInt(0));
                }
                default -> throw new UnsupportedFixture("unsupported fixture method: " + method);
            }
        }
        return value;
    }

    private static Map<String, Object> compare(JsonNode fixture, Map<String, Object> result, Object value, Throwable callError) {
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
