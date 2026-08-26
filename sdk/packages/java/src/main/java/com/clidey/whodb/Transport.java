package com.clidey.whodb;

import java.util.Map;

/**
 * Executes one platform operation and returns the GraphQL data object.
 * Implementations: the default HTTP transport and {@link IpcTransport}
 * (inside the WhoDB Functions runtime). The generated core and facades never
 * speak HTTP directly.
 */
public interface Transport {
    /** Executes one operation, returning the GraphQL {@code data} object. */
    Map<String, Object> execute(String operation, String document, Map<String, Object> variables);
}
