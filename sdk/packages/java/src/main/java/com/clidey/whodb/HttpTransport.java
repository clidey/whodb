package com.clidey.whodb;

import com.fasterxml.jackson.databind.ObjectMapper;
import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.util.List;
import java.util.Map;
import java.util.Set;

/** The default GraphQL-over-HTTP transport (POST {@code /api/query}). */
final class HttpTransport implements Transport {
    private static final ObjectMapper JSON = new ObjectMapper();
    private static final Set<Integer> RETRYABLE_STATUS = Set.of(502, 503, 504);

    private final URI endpoint;
    private final CredentialProvider credentials;
    // HTTP/1.1: the default HTTP/2 h2c upgrade is rejected (426) by the platform.
    private final HttpClient client = HttpClient.newBuilder().version(HttpClient.Version.HTTP_1_1).build();

    private volatile String orgId = "";
    private volatile String projectId = "";

    HttpTransport(String host, CredentialProvider credentials) {
        String stripped = host;
        while (stripped.endsWith("/")) {
            stripped = stripped.substring(0, stripped.length() - 1);
        }
        this.endpoint = URI.create(stripped + "/api/query");
        this.credentials = credentials;
    }

    /** Sets the workspace scope headers used on subsequent requests. */
    void setWorkspace(String orgId, String projectId) {
        this.orgId = orgId;
        this.projectId = projectId;
    }

    private HttpResponse<String> post(String body) {
        HttpRequest.Builder builder = HttpRequest.newBuilder(endpoint)
            .header("Content-Type", "application/json")
            .header("Authorization", "Bearer " + credentials.token())
            .header("User-Agent", "clidey-whodb-java/" + WhoDB.SDK_VERSION)
            .POST(HttpRequest.BodyPublishers.ofString(body));
        if (!orgId.isEmpty()) {
            builder.header("X-Whodb-Org-Id", orgId);
        }
        if (!projectId.isEmpty()) {
            builder.header("X-Whodb-Project-Id", projectId);
        }
        try {
            return client.send(builder.build(), HttpResponse.BodyHandlers.ofString());
        } catch (IOException error) {
            throw new WhoDBException(WhoDBException.Kind.TRANSPORT, "platform request failed: " + error.getMessage());
        } catch (InterruptedException error) {
            Thread.currentThread().interrupt();
            throw new WhoDBException(WhoDBException.Kind.TRANSPORT, "platform request interrupted");
        }
    }

    /** Performs one operation with a single 401-refresh and 5xx retry. */
    @Override
    @SuppressWarnings("unchecked")
    public Map<String, Object> execute(String operation, String document, Map<String, Object> variables) {
        String body;
        try {
            body = JSON.writeValueAsString(Map.of("query", document, "variables", variables));
        } catch (Exception error) {
            throw new WhoDBException(WhoDBException.Kind.TRANSPORT, "failed to serialize request for " + operation);
        }
        HttpResponse<String> response = post(body);
        if (response.statusCode() == 401) {
            credentials.refresh();
            response = post(body);
        } else if (RETRYABLE_STATUS.contains(response.statusCode())) {
            response = post(body);
        }
        if (response.statusCode() == 401) {
            throw new WhoDBException(WhoDBException.Kind.AUTH,
                "authentication failed — check your API key or run: whodb login");
        }
        if (response.statusCode() >= 400) {
            throw new WhoDBException(WhoDBException.Kind.PLATFORM,
                "platform request failed with HTTP " + response.statusCode(), "HTTP_" + response.statusCode());
        }
        Map<String, Object> payload;
        try {
            payload = JSON.readValue(response.body(), Map.class);
        } catch (Exception error) {
            throw new WhoDBException(WhoDBException.Kind.PLATFORM, "invalid response for " + operation, "INVALID_RESPONSE");
        }
        Object errors = payload.get("errors");
        if (errors instanceof List<?> errorList && !errorList.isEmpty()) {
            throw WhoDBException.fromGraphQLErrors((List<Object>) errors);
        }
        Object data = payload.get("data");
        if (!(data instanceof Map<?, ?>)) {
            throw new WhoDBException(WhoDBException.Kind.PLATFORM, "empty response for " + operation, "EMPTY_RESPONSE");
        }
        return (Map<String, Object>) data;
    }
}
