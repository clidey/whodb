package com.clidey.whodb;

import com.fasterxml.jackson.databind.ObjectMapper;
import java.io.IOException;
import java.net.URI;
import java.net.UnixDomainSocketAddress;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.channels.SocketChannel;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * Runs the same facades inside the WhoDB Functions runtime over the
 * in-container IPC server (unix socket in Docker, TCP in K8s). Only the
 * ontology operations are available; others raise
 * {@code Kind.TRANSPORT_CAPABILITY}. GraphQL entity IDs resolve to apiNames
 * via a cached {@code /entities} call.
 */
public final class IpcTransport implements Transport {
    private static final ObjectMapper JSON = new ObjectMapper();

    /**
     * IPC transport configuration.
     *
     * @param address unix socket path (starts with {@code /}) or TCP host:port
     * @param jobId the runtime job ID sent as X-Job-ID
     * @param token the IPC auth token
     */
    public record Config(String address, String jobId, String token) {
        /**
         * Reads the runtime's WHODB_IPC_ADDRESS / WHODB_JOB_ID /
         * WHODB_IPC_TOKEN env vars.
         */
        public static Config fromEnv() {
            return new Config(env("WHODB_IPC_ADDRESS"), env("WHODB_JOB_ID"), env("WHODB_IPC_TOKEN"));
        }

        private static String env(String name) {
            String value = System.getenv(name);
            return value == null ? "" : value;
        }
    }

    private final Config config;
    private final HttpClient tcpClient;
    private List<Map<String, Object>> entities;

    /**
     * Creates the IPC transport. WhoDB clients construct it automatically
     * inside the Functions runtime; explicit use is for tests.
     */
    public IpcTransport(Config config) {
        this.config = config;
        this.tcpClient = config.address().startsWith("/") ? null : HttpClient.newHttpClient();
    }

    @SuppressWarnings("unchecked")
    private Object post(String path, Object body) {
        String encoded;
        try {
            encoded = JSON.writeValueAsString(body);
        } catch (Exception error) {
            throw new WhoDBException(WhoDBException.Kind.TRANSPORT, "failed to serialize IPC request " + path);
        }
        int status;
        String responseBody;
        if (tcpClient != null) {
            HttpRequest request = HttpRequest.newBuilder(URI.create("http://" + config.address() + path))
                .header("Content-Type", "application/json")
                .header("X-Job-ID", config.jobId())
                .header("Authorization", config.token())
                .POST(HttpRequest.BodyPublishers.ofString(encoded))
                .build();
            try {
                HttpResponse<String> response = tcpClient.send(request, HttpResponse.BodyHandlers.ofString());
                status = response.statusCode();
                responseBody = response.body();
            } catch (IOException error) {
                throw new WhoDBException(WhoDBException.Kind.TRANSPORT, "IPC request " + path + " failed: " + error.getMessage());
            } catch (InterruptedException error) {
                Thread.currentThread().interrupt();
                throw new WhoDBException(WhoDBException.Kind.TRANSPORT, "IPC request " + path + " interrupted");
            }
        } else {
            String raw = unixHttpPost(path, encoded);
            int headerEnd = raw.indexOf("\r\n\r\n");
            String statusLine = raw.substring(0, raw.indexOf("\r\n"));
            status = Integer.parseInt(statusLine.split(" ")[1]);
            responseBody = headerEnd >= 0 ? raw.substring(headerEnd + 4) : "";
            responseBody = stripChunked(raw.substring(0, headerEnd >= 0 ? headerEnd : 0), responseBody);
        }
        if (status >= 400) {
            throw new WhoDBException(WhoDBException.Kind.PLATFORM,
                "IPC request " + path + " failed with HTTP " + status, "IPC_" + status);
        }
        try {
            return JSON.readValue(responseBody, Object.class);
        } catch (Exception error) {
            throw new WhoDBException(WhoDBException.Kind.PLATFORM,
                "IPC request " + path + " returned invalid JSON", "IPC_INVALID_JSON");
        }
    }

    /**
     * Minimal HTTP/1.1 POST over a unix domain socket — java.net.http cannot
     * dial unix sockets, so this speaks the wire format directly.
     */
    private String unixHttpPost(String path, String body) {
        byte[] payload = body.getBytes(StandardCharsets.UTF_8);
        String head = "POST " + path + " HTTP/1.1\r\n"
            + "Host: whodb-ipc\r\n"
            + "Content-Type: application/json\r\n"
            + "X-Job-ID: " + config.jobId() + "\r\n"
            + "Authorization: " + config.token() + "\r\n"
            + "Content-Length: " + payload.length + "\r\n"
            + "Connection: close\r\n\r\n";
        try (SocketChannel channel = SocketChannel.open(UnixDomainSocketAddress.of(config.address()))) {
            channel.write(java.nio.ByteBuffer.wrap(head.getBytes(StandardCharsets.UTF_8)));
            channel.write(java.nio.ByteBuffer.wrap(payload));
            var buffer = java.nio.ByteBuffer.allocate(64 * 1024);
            var output = new java.io.ByteArrayOutputStream();
            while (channel.read(buffer) >= 0) {
                buffer.flip();
                output.write(buffer.array(), 0, buffer.limit());
                buffer.clear();
            }
            return output.toString(StandardCharsets.UTF_8);
        } catch (IOException error) {
            throw new WhoDBException(WhoDBException.Kind.TRANSPORT, "IPC request " + path + " failed: " + error.getMessage());
        }
    }

    /** De-chunks a Transfer-Encoding: chunked body; passes others through. */
    private static String stripChunked(String headers, String body) {
        if (!headers.toLowerCase(java.util.Locale.ROOT).contains("transfer-encoding: chunked")) {
            return body;
        }
        StringBuilder decoded = new StringBuilder();
        int cursor = 0;
        while (cursor < body.length()) {
            int lineEnd = body.indexOf("\r\n", cursor);
            if (lineEnd < 0) {
                break;
            }
            int size = Integer.parseInt(body.substring(cursor, lineEnd).trim(), 16);
            if (size == 0) {
                break;
            }
            decoded.append(body, lineEnd + 2, lineEnd + 2 + size);
            cursor = lineEnd + 2 + size + 2;
        }
        return decoded.toString();
    }

    @SuppressWarnings("unchecked")
    private synchronized List<Map<String, Object>> entityList() {
        if (entities == null) {
            Object raw = post("/entities", Map.of());
            List<Map<String, Object>> resolved = new ArrayList<>();
            if (raw instanceof List<?> entries) {
                for (Object entry : entries) {
                    if (entry instanceof Map<?, ?> entity) {
                        resolved.add((Map<String, Object>) entity);
                    }
                }
            }
            entities = resolved;
        }
        return entities;
    }

    private String entityField(String entityId, String field) {
        for (Map<String, Object> entity : entityList()) {
            if (entityId.equals(entity.get("id"))) {
                Object value = entity.get(field);
                return value instanceof String text ? text : "";
            }
        }
        throw new WhoDBException(WhoDBException.Kind.NOT_FOUND,
            "ontology entity " + entityId + " not found in this function's scope");
    }

    @SuppressWarnings("unchecked")
    private static Map<String, Object> recordInputsToData(Object values) {
        Map<String, Object> data = new LinkedHashMap<>();
        if (values instanceof List<?> entries) {
            for (Object entry : entries) {
                if (entry instanceof Map<?, ?> record) {
                    Object key = record.get("Key");
                    if (key != null) {
                        data.put(key.toString(), record.get("Value"));
                    }
                }
            }
        }
        return data;
    }

    /**
     * Maps supported operations onto IPC endpoints, reshaping results into
     * the GraphQL data shape the generated core expects.
     */
    @Override
    public Map<String, Object> execute(String operation, String document, Map<String, Object> variables) {
        Map<String, Object> data = new HashMap<>();
        data.put(operation, dispatch(operation, variables));
        return data;
    }

    @SuppressWarnings("unchecked")
    private Object dispatch(String operation, Map<String, Object> variables) {
        switch (operation) {
            case "OntologyEntities":
                return entityList();
            case "OntologyQuery": {
                Map<String, Object> input = (Map<String, Object>) variables.getOrDefault("input", Map.of());
                Map<String, Object> body = new HashMap<>();
                for (Map.Entry<String, Object> entry : input.entrySet()) {
                    if (entry.getValue() != null) {
                        body.put(entry.getKey(), entry.getValue());
                    }
                }
                Object whereJson = body.remove("whereJson");
                if (whereJson instanceof String text) {
                    try {
                        body.put("where", JSON.readValue(text, Map.class));
                    } catch (Exception ignored) {
                        // invalid filter JSON — omit, matching other SDKs
                    }
                }
                return post("/query", body);
            }
            case "OntologyDescribe":
                return post("/describe", variables.getOrDefault("input", Map.of()));
            case "OntologyAddRow": {
                String entity = entityField(String.valueOf(variables.get("entityId")), "apiName");
                post("/create", Map.of("entity", entity, "data", recordInputsToData(variables.get("values"))));
                return Map.of("Status", true);
            }
            case "OntologyAddRows": {
                String entity = entityField(String.valueOf(variables.get("entityId")), "apiName");
                List<Map<String, Object>> rows = new ArrayList<>();
                if (variables.get("rows") instanceof List<?> entries) {
                    for (Object entry : entries) {
                        if (entry instanceof Map<?, ?> row) {
                            rows.add(recordInputsToData(row.get("values")));
                        }
                    }
                }
                Map<String, Object> body = new HashMap<>();
                body.put("entity", entity);
                body.put("rows", rows);
                body.put("idempotencyKey", variables.get("idempotencyKey"));
                Object ids = post("/create_many", body);
                List<Object> idList = ids instanceof List<?> list ? new ArrayList<>(list) : List.of();
                return Map.of("inserted", idList.size(), "ids", idList);
            }
            case "OntologyUpdateRow": {
                String entityId = String.valueOf(variables.get("entityId"));
                String entity = entityField(entityId, "apiName");
                String pkKey = entityField(entityId, "primaryKey");
                Map<String, Object> data = recordInputsToData(variables.get("values"));
                String pk = "";
                if (!pkKey.isEmpty() && data.containsKey(pkKey)) {
                    pk = String.valueOf(data.remove(pkKey));
                }
                post("/update", Map.of("entity", entity, "pk", pk, "data", data));
                return Map.of("Status", true);
            }
            case "OntologyDeleteRow": {
                String entityId = String.valueOf(variables.get("entityId"));
                String entity = entityField(entityId, "apiName");
                String pkKey = entityField(entityId, "primaryKey");
                Map<String, Object> data = recordInputsToData(variables.get("values"));
                post("/delete", Map.of("entity", entity, "pk", String.valueOf(data.get(pkKey))));
                return Map.of("Status", true);
            }
            case "OntologyFollowLink": {
                String entity = entityField(String.valueOf(variables.get("entityId")), "apiName");
                Map<String, Object> body = new HashMap<>();
                body.put("entity", entity);
                body.put("pk", variables.get("pk"));
                body.put("link", variables.get("linkApiName"));
                body.put("pageSize", variables.get("pageSize"));
                body.put("offset", variables.get("pageOffset"));
                return post("/follow_link", body);
            }
            default:
                throw new WhoDBException(WhoDBException.Kind.TRANSPORT_CAPABILITY,
                    operation + " is not available inside the function runtime in v1");
        }
    }
}
