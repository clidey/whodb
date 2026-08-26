import { embeddedManifest } from './generated/manifest.js';
import { WhoDBVersionError } from './errors.js';
const warned = new Set();
/**
 * Emits the two-tier versioning-policy warnings for an operation, once per
 * process (SDK_DESIGN.md §2.3): deprecated → upgrade-before-sunset warning;
 * behaviorChanged → semantics-changed warning. Actually-removed operations
 * surface as WhoDBVersionError from interpretServerError below.
 */
export function warnIfFlagged(operationName) {
    const entry = embeddedManifest[operationName];
    if (!entry || warned.has(operationName))
        return;
    if (entry.deprecated) {
        warned.add(operationName);
        console.warn(`[whodb] ${operationName} is deprecated${entry.sunsetAt ? ` and will be removed after ${entry.sunsetAt}` : ''} — upgrade @clidey/whodb-sdk before then.${entry.note ? ` ${entry.note}` : ''}`);
    }
    else if (entry.behaviorChanged) {
        warned.add(operationName);
        console.warn(`[whodb] ${operationName}'s behavior changed in this platform release — results may differ from previous SDK versions.${entry.note ? ` ${entry.note}` : ''}`);
    }
}
const UNKNOWN_OPERATION_PATTERNS = [
    /Cannot query field/i,
    /Unknown field/i,
    /Unknown type/i,
    /has no field/i,
];
/**
 * Detects the server rejecting an operation this SDK was generated with —
 * the operation was removed after this SDK release. Converts the low-level
 * validation error into the actionable upgrade error.
 */
export function interpretServerError(error, sdkVersion) {
    if (error instanceof Error && UNKNOWN_OPERATION_PATTERNS.some(p => p.test(error.message))) {
        return new WhoDBVersionError(`this SDK (${sdkVersion}) was built for an older WhoDB platform API; upgrade the @clidey/whodb-sdk package`);
    }
    return error;
}
/** Test hook: clears the once-per-process warning memory. */
export function resetWarnings() {
    warned.clear();
}
