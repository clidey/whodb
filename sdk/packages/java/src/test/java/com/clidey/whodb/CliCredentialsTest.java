package com.clidey.whodb;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.attribute.PosixFilePermissions;
import java.time.OffsetDateTime;
import java.time.ZoneOffset;
import java.time.format.DateTimeFormatter;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;

/** CLI credential-helper tests using a fake whodb binary. */
class CliCredentialsTest {

    @TempDir
    Path dir;

    /** Writes a fake whodb CLI script; appends to a counter file per run. */
    private Path fakeCli(String body) throws Exception {
        Path command = dir.resolve("whodb");
        Files.writeString(command, "#!/bin/sh\necho x >> \"" + dir.resolve("count") + "\"\n" + body + "\n");
        Files.setPosixFilePermissions(command, PosixFilePermissions.fromString("rwxr-xr-x"));
        return command;
    }

    private long execCount() throws Exception {
        Path countFile = dir.resolve("count");
        return Files.exists(countFile) ? Files.readString(countFile).length() / 2 : 0;
    }

    private static String futureExpiry(long seconds) {
        return OffsetDateTime.now(ZoneOffset.UTC).plusSeconds(seconds)
            .format(DateTimeFormatter.ISO_OFFSET_DATE_TIME);
    }

    @Test
    void missingBinaryIsCliCredentialsError() {
        CliCredentials credentials = new CliCredentials(dir.resolve("no-such-binary").toString());
        WhoDBException error = assertThrows(WhoDBException.class, credentials::token);
        assertEquals(WhoDBException.Kind.CLI_CREDENTIALS, error.kind());
    }

    @Test
    void invalidJsonIsCliCredentialsError() throws Exception {
        CliCredentials credentials = new CliCredentials(fakeCli("echo not-json").toString());
        WhoDBException error = assertThrows(WhoDBException.class, credentials::token);
        assertEquals(WhoDBException.Kind.CLI_CREDENTIALS, error.kind());
    }

    @Test
    void nonZeroExitIsCliCredentialsError() throws Exception {
        CliCredentials credentials =
            new CliCredentials(fakeCli("echo 'run: whodb login' >&2; exit 1").toString());
        WhoDBException error = assertThrows(WhoDBException.class, credentials::token);
        assertEquals(WhoDBException.Kind.CLI_CREDENTIALS, error.kind());
    }

    @Test
    void freshTokensAreCached() throws Exception {
        CliCredentials credentials = new CliCredentials(fakeCli(
            "echo '{\"access_token\":\"tok-1\",\"expires_at\":\"" + futureExpiry(3600) + "\"}'").toString());
        for (int i = 0; i < 3; i++) {
            assertEquals("tok-1", credentials.token());
        }
        assertEquals(1, execCount());
    }

    @Test
    void nearExpiryTokensReExec() throws Exception {
        // 30s ahead: inside the 60-second refresh skew.
        CliCredentials credentials = new CliCredentials(fakeCli(
            "echo '{\"access_token\":\"tok-1\",\"expires_at\":\"" + futureExpiry(30) + "\"}'").toString());
        credentials.token();
        credentials.token();
        assertEquals(2, execCount());
    }

    @Test
    void refreshDropsTheCache() throws Exception {
        CliCredentials credentials = new CliCredentials(fakeCli(
            "echo '{\"access_token\":\"tok-1\",\"expires_at\":\"" + futureExpiry(3600) + "\"}'").toString());
        credentials.token();
        credentials.refresh();
        credentials.token();
        assertEquals(2, execCount());
    }
}
