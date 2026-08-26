package com.clidey.whodb;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.io.IOException;
import java.time.Duration;
import java.time.OffsetDateTime;
import java.util.concurrent.TimeUnit;

/**
 * Credential provider that execs {@code whodb auth print-token --format json}
 * and caches the token until shortly before expiry — the gcloud-ADC pattern
 * for local development. Requires the whodb CLI on PATH and a prior
 * {@code whodb login}.
 */
final class CliCredentials implements CredentialProvider {
    private static final ObjectMapper JSON = new ObjectMapper();
    private static final Duration REFRESH_SKEW = Duration.ofSeconds(60);

    /** The parsed output of one CLI invocation. */
    record TokenEntry(String accessToken, String expiresAt, String host, String orgId, String projectId) {}

    private TokenEntry cached;

    private synchronized TokenEntry exec() {
        Process process;
        try {
            process = new ProcessBuilder("whodb", "auth", "print-token", "--format", "json").start();
        } catch (IOException error) {
            throw new WhoDBException(WhoDBException.Kind.CLI_CREDENTIALS,
                "whodb CLI not found — install it or set WHODB_API_KEY");
        }
        try {
            String output = new String(process.getInputStream().readAllBytes(), java.nio.charset.StandardCharsets.UTF_8);
            String stderr = new String(process.getErrorStream().readAllBytes(), java.nio.charset.StandardCharsets.UTF_8);
            if (!process.waitFor(15, TimeUnit.SECONDS) || process.exitValue() != 0) {
                process.destroyForcibly();
                throw new WhoDBException(WhoDBException.Kind.CLI_CREDENTIALS,
                    "whodb auth print-token failed: " + stderr.trim());
            }
            JsonNode parsed = JSON.readTree(output);
            return new TokenEntry(
                parsed.path("access_token").asText(""),
                parsed.path("expires_at").asText(""),
                parsed.path("host").asText(""),
                parsed.path("org_id").asText(""),
                parsed.path("project_id").asText(""));
        } catch (WhoDBException error) {
            throw error;
        } catch (InterruptedException error) {
            Thread.currentThread().interrupt();
            throw new WhoDBException(WhoDBException.Kind.CLI_CREDENTIALS, "whodb auth print-token interrupted");
        } catch (Exception error) {
            throw new WhoDBException(WhoDBException.Kind.CLI_CREDENTIALS,
                "whodb auth print-token returned invalid JSON");
        }
    }

    private boolean fresh(TokenEntry entry) {
        if (entry == null || entry.expiresAt().isEmpty()) {
            return false;
        }
        try {
            OffsetDateTime expiry = OffsetDateTime.parse(entry.expiresAt());
            return Duration.between(OffsetDateTime.now(expiry.getOffset()), expiry).compareTo(REFRESH_SKEW) > 0;
        } catch (Exception error) {
            return false;
        }
    }

    @Override
    public synchronized String token() {
        if (!fresh(cached)) {
            cached = exec();
        }
        if (cached.accessToken().isEmpty()) {
            throw new WhoDBException(WhoDBException.Kind.AUTH, "whodb CLI returned an empty access token");
        }
        return cached.accessToken();
    }

    @Override
    public synchronized void refresh() {
        cached = null;
    }

    /** Returns the CLI's saved host/org/project defaults. */
    synchronized TokenEntry defaults() {
        if (cached == null) {
            cached = exec();
        }
        return cached;
    }
}
