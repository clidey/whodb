package com.clidey.whodb;

import java.util.List;
import java.util.Map;

/**
 * Error taxonomy for the WhoDB SDK, mirrored across all SDK languages.
 * {@link Kind} identifies the failure class; {@link #code()} carries the
 * platform's GraphQL error code for {@code Kind.PLATFORM}.
 */
public class WhoDBException extends RuntimeException {

    /** The failure class, shared across all SDK languages. */
    public enum Kind {
        /** Authentication failed: missing, invalid, expired, or revoked credentials. */
        AUTH,
        /** The requested resource does not exist or the caller cannot see it. */
        NOT_FOUND,
        /** The request was rejected as invalid before execution. */
        VALIDATION,
        /** This SDK release targets an older platform API; upgrade the artifact. */
        VERSION,
        /** The whodb CLI credential helper is unavailable or not logged in. */
        CLI_CREDENTIALS,
        /** An operation is not available over the current transport (e.g. IPC). */
        TRANSPORT_CAPABILITY,
        /** Any other platform-reported error; {@link #code()} carries the GraphQL code. */
        PLATFORM,
        /** Transport-level failure (network, serialization). */
        TRANSPORT
    }

    private final Kind kind;
    private final String code;

    /** Creates an exception of the given kind. */
    public WhoDBException(Kind kind, String message) {
        this(kind, message, "");
    }

    /** Creates an exception carrying a platform error code. */
    public WhoDBException(Kind kind, String message, String code) {
        super(message);
        this.kind = kind;
        this.code = code;
    }

    /** The failure class. */
    public Kind kind() {
        return kind;
    }

    /** The platform's GraphQL error code (empty unless kind is PLATFORM). */
    public String code() {
        return code;
    }

    /** Maps a GraphQL errors array to the SDK taxonomy; the first error decides. */
    @SuppressWarnings("unchecked")
    public static WhoDBException fromGraphQLErrors(List<Object> errors) {
        if (errors == null || errors.isEmpty()) {
            return new WhoDBException(Kind.PLATFORM, "unknown platform error");
        }
        Map<String, Object> first = (Map<String, Object>) errors.get(0);
        String message = String.valueOf(first.getOrDefault("message", "unknown platform error"));
        String code = "";
        Object extensions = first.get("extensions");
        if (extensions instanceof Map<?, ?> extensionsMap) {
            Object rawCode = extensionsMap.get("code");
            code = rawCode == null ? "" : rawCode.toString();
        }
        return switch (code) {
            case "UNAUTHENTICATED", "FORBIDDEN" -> new WhoDBException(Kind.AUTH, message);
            case "NOT_FOUND" -> new WhoDBException(Kind.NOT_FOUND, message);
            case "BAD_USER_INPUT", "GRAPHQL_VALIDATION_FAILED" -> new WhoDBException(Kind.VALIDATION, message);
            default -> new WhoDBException(Kind.PLATFORM, message, code);
        };
    }

    private static final String[] UNKNOWN_OPERATION_MARKERS = {
        "cannot query field", "unknown field", "unknown type", "has no field",
    };

    /** Converts "unknown operation" server rejections into the upgrade error. */
    static WhoDBException interpretServerError(WhoDBException error) {
        String message = error.getMessage() == null ? "" : error.getMessage().toLowerCase(java.util.Locale.ROOT);
        for (String marker : UNKNOWN_OPERATION_MARKERS) {
            if (message.contains(marker)) {
                return new WhoDBException(Kind.VERSION,
                    "this SDK (" + WhoDB.SDK_VERSION + ") was built for an older WhoDB platform API; "
                        + "upgrade the com.clidey:whodb-sdk artifact");
            }
        }
        return error;
    }
}
