// Package whodb is the official Go SDK for the WhoDB hosted platform —
// the ontology, datasets, and sources as in-code function APIs.
//
//	client, err := whodb.New(whodb.Config{APIKey: os.Getenv("WHODB_API_KEY")})
//	user, err := client.Ontology("User").Get(ctx, "u_123")
package whodb

import (
	"errors"
	"fmt"
	"regexp"
)

// Sentinel errors forming the SDK error taxonomy, mirrored across languages.
// Match with errors.Is; PlatformError additionally carries the GraphQL code.
var (
	// ErrAuth: authentication failed — missing, invalid, expired, or revoked credentials.
	ErrAuth = errors.New("whodb: authentication failed")
	// ErrNotFound: the resource does not exist or the caller cannot see it.
	ErrNotFound = errors.New("whodb: not found")
	// ErrValidation: the request was rejected as invalid before execution.
	ErrValidation = errors.New("whodb: invalid request")
	// ErrVersion: this SDK targets an older platform API; upgrade the module.
	ErrVersion = errors.New("whodb: sdk outdated for platform API")
	// ErrCliCredentials: the whodb CLI credential helper is unavailable or not logged in.
	ErrCliCredentials = errors.New("whodb: cli credentials unavailable")
	// ErrTransportCapability: the operation is unavailable over the current transport.
	ErrTransportCapability = errors.New("whodb: operation not available over this transport")
)

// PlatformError is any other platform-reported error, carrying the GraphQL code.
type PlatformError struct {
	Message string
	Code    string
}

// Error implements error.
func (e *PlatformError) Error() string {
	return fmt.Sprintf("whodb: platform error %s: %s", e.Code, e.Message)
}

// graphQLError is one entry of a GraphQL errors array.
type graphQLError struct {
	Message    string `json:"message"`
	Extensions struct {
		Code string `json:"code"`
	} `json:"extensions"`
}

// mapGraphQLErrors maps a GraphQL errors array to the SDK error taxonomy.
// The first error decides the type.
func mapGraphQLErrors(errs []graphQLError) error {
	if len(errs) == 0 {
		return &PlatformError{Message: "unknown platform error", Code: ""}
	}
	first := errs[0]
	switch first.Extensions.Code {
	case "UNAUTHENTICATED", "FORBIDDEN":
		return fmt.Errorf("%w: %s", ErrAuth, first.Message)
	case "NOT_FOUND":
		return fmt.Errorf("%w: %s", ErrNotFound, first.Message)
	case "BAD_USER_INPUT", "GRAPHQL_VALIDATION_FAILED":
		return fmt.Errorf("%w: %s", ErrValidation, first.Message)
	default:
		return &PlatformError{Message: first.Message, Code: first.Extensions.Code}
	}
}

var unknownOperationPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)Cannot query field`),
	regexp.MustCompile(`(?i)Unknown field`),
	regexp.MustCompile(`(?i)Unknown type`),
	regexp.MustCompile(`(?i)has no field`),
}

// interpretServerError converts "unknown operation" server rejections — the
// operation was removed after this SDK release — into the actionable
// upgrade error.
func interpretServerError(err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	for _, pattern := range unknownOperationPatterns {
		if pattern.MatchString(message) {
			return fmt.Errorf("%w: this SDK (%s) was built for an older WhoDB platform API; upgrade github.com/clidey/whodb/sdk/packages/go", ErrVersion, SDKVersion)
		}
	}
	return err
}
