package com.clidey.whodb;

import com.clidey.whodb.gen.Manifest;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Once-per-process deprecation/behavior-change warnings for operations (SDK
 * versioning policy §2.3). Actually-removed operations surface as
 * {@code Kind.VERSION} from {@code interpretServerError}.
 */
final class ManifestCheck {
    private static final Set<String> WARNED = ConcurrentHashMap.newKeySet();

    private ManifestCheck() {}

    /** Emits the once-per-process warning when an operation is flagged. */
    static void warnIfFlagged(String operation) {
        Manifest.OperationPolicy policy = Manifest.EMBEDDED.get(operation);
        if (policy == null) {
            return;
        }
        if (policy.deprecated()) {
            if (WARNED.add(operation)) {
                String suffix = policy.sunsetAt().isEmpty() ? "" : " and will be removed after " + policy.sunsetAt();
                System.err.println("[whodb] " + operation + " is deprecated" + suffix
                    + " — upgrade the com.clidey:whodb-sdk artifact before then. " + policy.note());
            }
        } else if (policy.behaviorChanged()) {
            if (WARNED.add(operation)) {
                System.err.println("[whodb] " + operation
                    + "'s behavior changed in this platform release — results may differ from previous SDK versions. "
                    + policy.note());
            }
        }
    }

    /** Test hook clearing the once-per-process warning memory. */
    static void resetWarnings() {
        WARNED.clear();
    }
}
