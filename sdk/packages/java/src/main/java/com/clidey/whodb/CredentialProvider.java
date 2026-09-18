package com.clidey.whodb;

/**
 * Yields a bearer credential for platform requests. {@link #refresh()} is
 * invoked once after a 401 before the request is retried.
 */
public interface CredentialProvider {
    /** Returns the bearer token to send. */
    String token();

    /** Drops any cached credential so the next {@link #token()} re-fetches. */
    default void refresh() {}
}
