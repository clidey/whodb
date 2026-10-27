# Error Design Patterns

This reference covers error handling patterns in GraphQL schema design.

## Table of Contents

- [GraphQL Error Model](#graphql-error-model)
- [When to Use Each Pattern](#when-to-use-each-pattern)
- [Union-Based Error Pattern](#union-based-error-pattern)
- [Interface-Based Errors](#interface-based-errors)
- [Error Codes](#error-codes)
- [Partial Success](#partial-success)

## GraphQL Error Model

GraphQL has a built-in error system with errors in the response:

```json
{
  "data": { "user": null },
  "errors": [
    {
      "message": "User not found",
      "path": ["user"],
      "extensions": {
        "code": "NOT_FOUND"
      }
    }
  ]
}
```

### Built-in Errors: Good For

- Unexpected errors (bugs, infrastructure issues)
- Authentication failures (401-level)
- Authorization failures (403-level)
- Validation of query itself

### Built-in Errors: Bad For

- Expected business errors (item out of stock)
- Multiple error types for one operation
- Errors that need rich data

## When to Use Each Pattern

| Scenario | Pattern |
|----------|---------|
| Unexpected server error | Built-in errors |
| Authentication required | Built-in errors |
| User input validation | Union result types |
| Business rule violation | Union result types |
| Partial success possible | Union or nullable fields |
| Multiple error types | Union result types |

## Union-Based Error Pattern

### Basic Result Type

```graphql
type Mutation {
  createUser(input: CreateUserInput!): CreateUserResult!
}

union CreateUserResult = CreateUserSuccess | ValidationError

type CreateUserSuccess {
  user: User!
}

type ValidationError {
  message: String!
  field: String
}
```

### Multiple Error Types

```graphql
union CreateOrderResult =
  | CreateOrderSuccess
  | ValidationError
  | InsufficientInventory
  | PaymentFailed

type CreateOrderSuccess {
  order: Order!
}

type ValidationError {
  message: String!
  field: String
}

type InsufficientInventory {
  message: String!
  unavailableItems: [OrderItem!]!
}

type PaymentFailed {
  message: String!
  reason: PaymentFailureReason!
  retryable: Boolean!
}

enum PaymentFailureReason {
  CARD_DECLINED
  INSUFFICIENT_FUNDS
  EXPIRED_CARD
  FRAUD_SUSPECTED
}
```

### Client Usage

```graphql
mutation CreateOrder($input: CreateOrderInput!) {
  createOrder(input: $input) {
    ... on CreateOrderSuccess {
      order {
        id
        total
      }
    }
    ... on ValidationError {
      message
      field
    }
    ... on InsufficientInventory {
      message
      unavailableItems {
        productId
        requestedQuantity
        availableQuantity
      }
    }
    ... on PaymentFailed {
      message
      reason
      retryable
    }
  }
}
```

### Benefits of Union Pattern

1. **Type safety** - Clients know all possible outcomes
2. **Rich error data** - Each error type has specific fields
3. **Self-documenting** - Schema shows what can go wrong
4. **Forced handling** - Clients must handle each case

## Interface-Based Errors

### Error Interface

```graphql
interface Error {
  message: String!
}

type ValidationError implements Error {
  message: String!
  field: String!
}

type NotFoundError implements Error {
  message: String!
  resourceType: String!
  resourceId: ID!
}

type PermissionError implements Error {
  message: String!
  requiredPermission: String!
}

union UpdateUserResult = User | ValidationError | NotFoundError | PermissionError
```

### Using Interface for Queries

```graphql
type Query {
  user(id: ID!): UserResult!
}

union UserResult = User | NotFoundError | PermissionError

# Client query:
query GetUser($id: ID!) {
  user(id: $id) {
    ... on User {
      id
      name
    }
    ... on Error {
      message
    }
  }
}
```

## Error Codes

### Standardize Error Codes

```graphql
enum ErrorCode {
  # Validation errors
  VALIDATION_FAILED
  INVALID_INPUT
  REQUIRED_FIELD_MISSING

  # Authentication/Authorization
  UNAUTHENTICATED
  UNAUTHORIZED
  TOKEN_EXPIRED

  # Resource errors
  NOT_FOUND
  ALREADY_EXISTS
  CONFLICT

  # Business logic
  INSUFFICIENT_FUNDS
  LIMIT_EXCEEDED
  OPERATION_NOT_ALLOWED

  # System errors
  INTERNAL_ERROR
  SERVICE_UNAVAILABLE
  RATE_LIMITED
}

type MutationError {
  code: ErrorCode!
  message: String!
  field: String
  details: JSON
}
```

### Error with Code Pattern

```graphql
type ValidationError {
  code: ErrorCode!
  message: String!
  field: String
}

type CreateUserSuccess {
  user: User!
}

union CreateUserResult = CreateUserSuccess | ValidationError

# Usage enables consistent error handling:
# if (result.__typename === 'ValidationError') {
#   switch (result.code) {
#     case 'ALREADY_EXISTS': ...
#     case 'INVALID_INPUT': ...
#   }
# }
```

## Partial Success

### Batch Operations

For operations on multiple items:

```graphql
input BulkUpdateInput {
  items: [UpdateItemInput!]!
}

type BulkUpdateResult {
  successful: [Item!]!
  failed: [BulkUpdateError!]!
}

type BulkUpdateError {
  index: Int!
  itemId: ID
  error: UpdateError!
}

union UpdateError = ValidationError | NotFoundError | PermissionError

type Mutation {
  bulkUpdateItems(input: BulkUpdateInput!): BulkUpdateResult!
}
```

### Client Usage for Batch

```graphql
mutation BulkUpdate($input: BulkUpdateInput!) {
  bulkUpdateItems(input: $input) {
    successful {
      id
      name
    }
    failed {
      index
      itemId
      error {
        ... on ValidationError {
          message
          field
        }
        ... on NotFoundError {
          message
          resourceId
        }
      }
    }
  }
}
```

### Warnings Pattern

Return success with warnings:

```graphql
type ImportResult {
  imported: [Record!]!
  skipped: [SkippedRecord!]!
  warnings: [ImportWarning!]!
}

type SkippedRecord {
  row: Int!
  reason: String!
  data: JSON
}

type ImportWarning {
  row: Int
  message: String!
  severity: WarningSeverity!
}

enum WarningSeverity {
  INFO
  WARNING
  ERROR
}
```

### Nullable Fields for Partial Data

```graphql
type UserWithExternalData {
  id: ID!
  name: String!
  # These might fail independently
  profileImage: Image           # External service
  socialConnections: [Social]   # External service
  # Errors for each
  profileImageError: String
  socialConnectionsError: String
}
```

Alternative with explicit result types:

```graphql
type UserWithExternalData {
  id: ID!
  name: String!
  profileImage: ImageResult!
  socialConnections: SocialConnectionsResult!
}

union ImageResult = Image | FetchError
union SocialConnectionsResult = SocialConnectionList | FetchError

type FetchError {
  message: String!
  service: String!
  retryable: Boolean!
}
```
