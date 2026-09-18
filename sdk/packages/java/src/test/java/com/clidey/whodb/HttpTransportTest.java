package com.clidey.whodb;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.sun.net.httpserver.Headers;
import com.sun.net.httpserver.HttpServer;
import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.atomic.AtomicInteger;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

/**
 * HTTP transport and workspace-resolution tests against a real local stub —
 * the layer the mock-transport conformance suite cannot reach (this is where
 * the HTTP/2 h2c 426 bug lived).
 */
class HttpTransportTest {
    private static final ObjectMapper JSON = new ObjectMapper();
    private static final String ORG_UUID = "11111111-1111-1111-1111-111111111111";
    private static final String PROJECT_UUID = "22222222-2222-2222-2222-222222222222";

    private record Recorded(Headers headers, String operation) {}

    private static HttpServer server;
    private static String host;
    private static final List<Recorded> requests = new ArrayList<>();
    private static final List<Map.Entry<Integer, String>> script = new ArrayList<>();

    @BeforeAll
    static void startStub() throws Exception {
        server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext("/", exchange -> {
            String raw = new String(exchange.getRequestBody().readAllBytes(), StandardCharsets.UTF_8);
            String query = String.valueOf(JSON.readValue(raw.isEmpty() ? "{}" : raw, Map.class).getOrDefault("query", ""));
            String[] words = query.split("[\\s({]+");
            String operation = words.length > 1 ? words[1] : "";
            synchronized (requests) {
                requests.add(new Recorded(exchange.getRequestHeaders(), operation));
            }
            Map.Entry<Integer, String> response;
            synchronized (script) {
                response = script.isEmpty() ? Map.entry(500, "{}") : script.remove(0);
            }
            byte[] body = response.getValue().getBytes(StandardCharsets.UTF_8);
            exchange.getResponseHeaders().set("Content-Type", "application/json");
            exchange.sendResponseHeaders(response.getKey(), body.length);
            try (OutputStream out = exchange.getResponseBody()) {
                out.write(body);
            }
        });
        server.start();
        host = "http://127.0.0.1:" + server.getAddress().getPort();
    }

    @AfterAll
    static void stopStub() {
        server.stop(0);
    }

    @BeforeEach
    void reset() {
        synchronized (requests) {
            requests.clear();
        }
        synchronized (script) {
            script.clear();
        }
    }

    private static void respond(int status, String body) {
        synchronized (script) {
            script.add(Map.entry(status, body));
        }
    }

    private static Recorded request(int index) {
        synchronized (requests) {
            return requests.get(index);
        }
    }

    private static WhoDB client(WhoDB.Config config) {
        return new WhoDB(new WhoDB.Config(config.apiKey(), config.token(), config.credentials(),
            config.org(), config.project(), host, null));
    }

    @Test
    void sendsBearerUserAgentAndWorkspaceHeaders() {
        respond(200, "{\"data\":{\"OntologyEntities\":[]}}");
        WhoDB whodb = client(new WhoDB.Config(null, "tok", null, ORG_UUID, PROJECT_UUID, null, null));
        whodb.ontologyEntities();
        Recorded recorded = request(0);
        assertEquals("Bearer tok", recorded.headers().getFirst("Authorization"));
        assertTrue(recorded.headers().getFirst("User-agent").startsWith("clidey-whodb-java/"));
        assertEquals(ORG_UUID, recorded.headers().getFirst("X-whodb-org-id"));
        assertEquals(PROJECT_UUID, recorded.headers().getFirst("X-whodb-project-id"));
    }

    @Test
    void retriesOnceOnTransient5xx() {
        respond(503, "{}");
        respond(200, "{\"data\":{\"OntologyEntities\":[]}}");
        WhoDB whodb = client(new WhoDB.Config(null, "tok", null, ORG_UUID, PROJECT_UUID, null, null));
        whodb.ontologyEntities();
        synchronized (requests) {
            assertEquals(2, requests.size());
        }
    }

    @Test
    void refreshesCredentialsOnceOn401() {
        respond(401, "{}");
        respond(200, "{\"data\":{\"OntologyEntities\":[]}}");
        AtomicInteger refreshes = new AtomicInteger();
        CredentialProvider credentials = new CredentialProvider() {
            @Override
            public String token() {
                return refreshes.get() == 0 ? "stale" : "token-" + refreshes.get();
            }

            @Override
            public void refresh() {
                refreshes.incrementAndGet();
            }
        };
        WhoDB whodb = client(new WhoDB.Config(null, null, credentials, ORG_UUID, PROJECT_UUID, null, null));
        whodb.ontologyEntities();
        assertEquals(1, refreshes.get());
        assertEquals("Bearer token-1", request(1).headers().getFirst("Authorization"));
    }

    @Test
    void persistent401IsAuthError() {
        respond(401, "{}");
        respond(401, "{}");
        WhoDB whodb = client(new WhoDB.Config(null, "bad", null, ORG_UUID, PROJECT_UUID, null, null));
        WhoDBException error = assertThrows(WhoDBException.class, whodb::ontologyEntities);
        assertEquals(WhoDBException.Kind.AUTH, error.kind());
    }

    @Test
    void graphQLErrorsMapOverRealHttp() {
        respond(200, "{\"errors\":[{\"message\":\"nope\",\"extensions\":{\"code\":\"NOT_FOUND\"}}]}");
        WhoDB whodb = client(new WhoDB.Config(null, "tok", null, ORG_UUID, PROJECT_UUID, null, null));
        WhoDBException error = assertThrows(WhoDBException.class, whodb::ontologyEntities);
        assertEquals(WhoDBException.Kind.NOT_FOUND, error.kind());
    }

    @Test
    void resolvesWorkspaceSlugsToIds() {
        respond(200, "{\"data\":{\"MyOrganizations\":[{\"id\":\"" + ORG_UUID + "\",\"slug\":\"acme\"}]}}");
        respond(200, "{\"data\":{\"Projects\":[{\"id\":\"" + PROJECT_UUID + "\",\"slug\":\"analytics\"}]}}");
        respond(200, "{\"data\":{\"OntologyEntities\":[]}}");
        WhoDB whodb = client(new WhoDB.Config(null, "tok", null, "acme", "analytics", null, null));
        whodb.ontologyEntities();
        assertEquals("MyOrganizations", request(0).operation());
        assertEquals("Projects", request(1).operation());
        assertEquals("OntologyEntities", request(2).operation());
        assertEquals(ORG_UUID, request(2).headers().getFirst("X-whodb-org-id"));
        assertEquals(PROJECT_UUID, request(2).headers().getFirst("X-whodb-project-id"));
    }

    @Test
    void apiKeyDiscoversWorkspaceViaMyWorkspace() {
        respond(200, "{\"data\":{\"MyWorkspace\":{\"orgId\":\"" + ORG_UUID + "\",\"projectId\":\"" + PROJECT_UUID + "\"}}}");
        respond(200, "{\"data\":{\"OntologyEntities\":[]}}");
        WhoDB whodb = client(new WhoDB.Config("whodb_sk_test", null, null, null, null, null, null));
        whodb.ontologyEntities();
        assertEquals("MyWorkspace", request(0).operation());
        assertEquals(PROJECT_UUID, request(1).headers().getFirst("X-whodb-project-id"));
    }

    @Test
    void multiGrantKeyWithoutProjectIsValidationError() {
        respond(200, "{\"data\":{\"MyWorkspace\":{\"orgId\":\"" + ORG_UUID + "\",\"projectId\":null}}}");
        WhoDB whodb = client(new WhoDB.Config("whodb_sk_test", null, null, null, null, null, null));
        WhoDBException error = assertThrows(WhoDBException.class, whodb::ontologyEntities);
        assertEquals(WhoDBException.Kind.VALIDATION, error.kind());
    }
}
