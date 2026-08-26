/** Base class for all errors thrown by the WhoDB SDK. */
export declare class WhoDBError extends Error {
    constructor(message: string);
}
/** Authentication failed: missing, invalid, expired, or revoked credentials. */
export declare class AuthError extends WhoDBError {
}
/** The requested resource does not exist or the caller cannot see it. */
export declare class NotFoundError extends WhoDBError {
}
/** The request was rejected as invalid before execution. */
export declare class ValidationError extends WhoDBError {
}
/**
 * This SDK release was generated against an older platform API and the
 * operation no longer exists. The only fix is upgrading the package.
 */
export declare class WhoDBVersionError extends WhoDBError {
}
/** The whodb CLI credential helper is unavailable or not logged in. */
export declare class CliCredentialsError extends WhoDBError {
}
/** An operation is not available over the current transport (e.g. IPC). */
export declare class TransportCapabilityError extends WhoDBError {
}
/** Any other platform-reported error, carrying the GraphQL error code. */
export declare class PlatformError extends WhoDBError {
    readonly code: string;
    constructor(message: string, code: string);
}
interface GraphQLErrorShape {
    message: string;
    extensions?: {
        code?: string;
    };
}
/**
 * Maps a GraphQL errors array to the SDK error taxonomy. The first error
 * decides the type; its code is preserved on PlatformError for callers that
 * need to branch on specifics.
 */
export declare function mapGraphQLErrors(errors: GraphQLErrorShape[]): WhoDBError;
export {};
