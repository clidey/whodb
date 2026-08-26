/** Base class for all errors thrown by the WhoDB SDK. */
export class WhoDBError extends Error {
    constructor(message) {
        super(message);
        this.name = new.target.name;
    }
}
/** Authentication failed: missing, invalid, expired, or revoked credentials. */
export class AuthError extends WhoDBError {
}
/** The requested resource does not exist or the caller cannot see it. */
export class NotFoundError extends WhoDBError {
}
/** The request was rejected as invalid before execution. */
export class ValidationError extends WhoDBError {
}
/**
 * This SDK release was generated against an older platform API and the
 * operation no longer exists. The only fix is upgrading the package.
 */
export class WhoDBVersionError extends WhoDBError {
}
/** The whodb CLI credential helper is unavailable or not logged in. */
export class CliCredentialsError extends WhoDBError {
}
/** An operation is not available over the current transport (e.g. IPC). */
export class TransportCapabilityError extends WhoDBError {
}
/** Any other platform-reported error, carrying the GraphQL error code. */
export class PlatformError extends WhoDBError {
    code;
    constructor(message, code) {
        super(message);
        this.code = code;
    }
}
/**
 * Maps a GraphQL errors array to the SDK error taxonomy. The first error
 * decides the type; its code is preserved on PlatformError for callers that
 * need to branch on specifics.
 */
export function mapGraphQLErrors(errors) {
    const first = errors[0] ?? { message: 'unknown platform error' };
    const code = first.extensions?.code ?? '';
    const message = first.message;
    switch (code) {
        case 'UNAUTHENTICATED':
            return new AuthError(message);
        case 'FORBIDDEN':
            return new AuthError(message);
        case 'NOT_FOUND':
            return new NotFoundError(message);
        case 'BAD_USER_INPUT':
        case 'GRAPHQL_VALIDATION_FAILED':
            return new ValidationError(message);
        default:
            return new PlatformError(message, code);
    }
}
