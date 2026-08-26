/**
 * Emits the two-tier versioning-policy warnings for an operation, once per
 * process (SDK_DESIGN.md §2.3): deprecated → upgrade-before-sunset warning;
 * behaviorChanged → semantics-changed warning. Actually-removed operations
 * surface as WhoDBVersionError from interpretServerError below.
 */
export declare function warnIfFlagged(operationName: string): void;
/**
 * Detects the server rejecting an operation this SDK was generated with —
 * the operation was removed after this SDK release. Converts the low-level
 * validation error into the actionable upgrade error.
 */
export declare function interpretServerError(error: unknown, sdkVersion: string): unknown;
/** Test hook: clears the once-per-process warning memory. */
export declare function resetWarnings(): void;
