package com.clidey.whodb;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.sun.net.httpserver.HttpServer;
import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.net.StandardProtocolFamily;
import java.net.UnixDomainSocketAddress;
import java.nio.ByteBuffer;
import java.nio.channels.ServerSocketChannel;
import java.nio.channels.SocketChannel;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

/** IPC dispatch tests over TCP, plus the raw unix-socket HTTP path. */
class IpcTransportTest {
    private static final ObjectMapper JSON = new ObjectMapper();

    private static HttpServer server;
    private static String address;
    private static final List<Map.Entry<String, Map<String, Object>>> calls = new ArrayList<>();

    @BeforeAll
    @SuppressWarnings("unchecked")
    static void startStub() throws Exception {
        server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext("/", exchange -> {
            String raw = new String(exchange.getRequestBody().readAllBytes(), StandardCharsets.UTF_8);
            Map<String, Object> body = JSON.readValue(raw.isEmpty() ? "{}" : raw, Map.class);
            String path = exchange.getRequestURI().getPath();
            synchronized (calls) {
                calls.add(Map.entry(path, body));
            }
            boolean authorized = "ipc-token".equals(exchange.getRequestHeaders().getFirst("Authorization"))
                && "job-1".equals(exchange.getRequestHeaders().getFirst("X-Job-ID"));
            String payload = switch (path) {
                case "/entities" -> "[{\"id\":\"ent-1\",\"apiName\":\"User\",\"primaryKey\":\"id\"}]";
                case "/query" -> "{\"columns\":[\"id\"],\"rows\":[[\"u1\"]],\"total\":1}";
                case "/create", "/update", "/delete" -> "{}";
                case "/create_many" -> "[\"id-1\",\"id-2\"]";
                default -> "{}";
            };
            byte[] bytes = payload.getBytes(StandardCharsets.UTF_8);
            exchange.getResponseHeaders().set("Content-Type", "application/json");
            exchange.sendResponseHeaders(authorized ? 200 : 401, bytes.length);
            try (OutputStream out = exchange.getResponseBody()) {
                out.write(bytes);
            }
        });
        server.start();
        address = "127.0.0.1:" + server.getAddress().getPort();
    }

    @AfterAll
    static void stopStub() {
        server.stop(0);
    }

    @BeforeEach
    void reset() {
        synchronized (calls) {
            calls.clear();
        }
    }

    private static IpcTransport transport() {
        return new IpcTransport(new IpcTransport.Config(address, "job-1", "ipc-token"));
    }

    private static Map.Entry<String, Map<String, Object>> call(int index) {
        synchronized (calls) {
            return calls.get(index);
        }
    }

    @Test
    void queryConvertsWhereJsonAndDropsNulls() {
        Map<String, Object> input = new java.util.HashMap<>();
        input.put("entity", "User");
        input.put("whereJson", "{\"id\":{\"eq\":\"u1\"}}");
        input.put("pageSize", 1);
        input.put("sort", null);
        Map<String, Object> data = transport().execute("OntologyQuery", "", Map.of("input", input));
        assertTrue(data.get("OntologyQuery") instanceof Map);
        Map.Entry<String, Map<String, Object>> recorded = call(0);
        assertEquals("/query", recorded.getKey());
        assertEquals(Map.of("id", Map.of("eq", "u1")), recorded.getValue().get("where"));
        assertTrue(!recorded.getValue().containsKey("whereJson"));
        assertTrue(!recorded.getValue().containsKey("sort"));
    }

    @Test
    @SuppressWarnings("unchecked")
    void addRowResolvesEntityAndConvertsRecordInputs() {
        Map<String, Object> data = transport().execute("OntologyAddRow", "", Map.of(
            "entityId", "ent-1",
            "values", List.of(Map.of("Key", "name", "Value", "Ada"))));
        assertEquals(true, ((Map<String, Object>) data.get("OntologyAddRow")).get("Status"));
        Map.Entry<String, Map<String, Object>> create = call(1); // call 0 is /entities
        assertEquals("/create", create.getKey());
        assertEquals("User", create.getValue().get("entity"));
        assertEquals(Map.of("name", "Ada"), create.getValue().get("data"));
    }

    @Test
    void updateRowSplitsPrimaryKey() {
        transport().execute("OntologyUpdateRow", "", Map.of(
            "entityId", "ent-1",
            "values", List.of(Map.of("Key", "id", "Value", "u1"), Map.of("Key", "plan", "Value", "pro"))));
        Map.Entry<String, Map<String, Object>> update = call(1);
        assertEquals("/update", update.getKey());
        assertEquals("u1", update.getValue().get("pk"));
        assertEquals(Map.of("plan", "pro"), update.getValue().get("data"));
    }

    @Test
    @SuppressWarnings("unchecked")
    void createManyReturnsInsertedCount() {
        Map<String, Object> data = transport().execute("OntologyAddRows", "", Map.of(
            "entityId", "ent-1",
            "rows", List.of(Map.of("values", List.of(Map.of("Key", "name", "Value", "Ada")))),
            "idempotencyKey", "batch-1"));
        assertEquals(2, ((Map<String, Object>) data.get("OntologyAddRows")).get("inserted"));
        assertEquals("batch-1", call(1).getValue().get("idempotencyKey"));
    }

    @Test
    void unknownEntityIsNotFound() {
        WhoDBException error = assertThrows(WhoDBException.class, () ->
            transport().execute("OntologyAddRow", "", Map.of("entityId", "nope", "values", List.of())));
        assertEquals(WhoDBException.Kind.NOT_FOUND, error.kind());
    }

    @Test
    void unsupportedOperationIsCapabilityError() {
        WhoDBException error = assertThrows(WhoDBException.class, () ->
            transport().execute("RunSourceQuery", "", Map.of()));
        assertEquals(WhoDBException.Kind.TRANSPORT_CAPABILITY, error.kind());
    }

    /** Serves one scripted raw HTTP response over a unix domain socket. */
    private static Path serveUnixOnce(String rawResponse) throws Exception {
        Path socketPath = Files.createTempDirectory("whodb-ipc-test").resolve("ipc.sock");
        ServerSocketChannel listener = ServerSocketChannel.open(StandardProtocolFamily.UNIX);
        listener.bind(UnixDomainSocketAddress.of(socketPath));
        Thread thread = new Thread(() -> {
            try (listener; SocketChannel channel = listener.accept()) {
                ByteBuffer buffer = ByteBuffer.allocate(64 * 1024);
                // Read until the request body arrives (headers + JSON body).
                StringBuilder seen = new StringBuilder();
                while (channel.read(buffer) >= 0) {
                    buffer.flip();
                    seen.append(StandardCharsets.UTF_8.decode(buffer));
                    buffer.clear();
                    int headerEnd = seen.indexOf("\r\n\r\n");
                    if (headerEnd >= 0 && seen.substring(headerEnd + 4).contains("}")) {
                        break;
                    }
                }
                channel.write(ByteBuffer.wrap(rawResponse.getBytes(StandardCharsets.UTF_8)));
            } catch (Exception ignored) {
                // test thread — failures surface as client-side errors
            }
        });
        thread.setDaemon(true);
        thread.start();
        return socketPath;
    }

    @Test
    void unixSocketPostParsesContentLengthResponse() throws Exception {
        String body = "{\"columns\":[\"id\"],\"rows\":[[\"u1\"]],\"total\":1}";
        Path socketPath = serveUnixOnce("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n"
            + "Content-Length: " + body.getBytes(StandardCharsets.UTF_8).length + "\r\n"
            + "Connection: close\r\n\r\n" + body);
        IpcTransport unixTransport = new IpcTransport(
            new IpcTransport.Config(socketPath.toString(), "job-1", "ipc-token"));
        Map<String, Object> data = unixTransport.execute("OntologyDescribe", "",
            Map.of("input", Map.of("entities", List.of("User"))));
        assertTrue(data.get("OntologyDescribe") instanceof Map);
    }

    @Test
    void unixSocketPostDecodesChunkedResponse() throws Exception {
        String chunkOne = "{\"columns\":[\"id\"],";
        String chunkTwo = "\"rows\":[[\"u1\"]],\"total\":1}";
        String raw = "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n"
            + "Transfer-Encoding: chunked\r\nConnection: close\r\n\r\n"
            + Integer.toHexString(chunkOne.length()) + "\r\n" + chunkOne + "\r\n"
            + Integer.toHexString(chunkTwo.length()) + "\r\n" + chunkTwo + "\r\n"
            + "0\r\n\r\n";
        Path socketPath = serveUnixOnce(raw);
        IpcTransport unixTransport = new IpcTransport(
            new IpcTransport.Config(socketPath.toString(), "job-1", "ipc-token"));
        Map<String, Object> data = unixTransport.execute("OntologyDescribe", "",
            Map.of("input", Map.of("entities", List.of("User"))));
        assertTrue(data.get("OntologyDescribe") instanceof Map);
    }
}
