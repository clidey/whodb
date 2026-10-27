# Error Handling Reference

## Table of Contents

- [GraphQLError](#graphqlerror)
- [Error Codes](#error-codes)
- [formatError Option](#formaterror-option)
- [Production Error Handling](#production-error-handling)

## GraphQLError

Apollo Server 4 uses `GraphQLError` from the `graphql` package. Always import from `graphql`, not from Apollo Server.

### Basic Usage

```typescript
import { GraphQLError } from "graphql";

const resolvers = {
  Query: {
    user: async (_, { id }, { dataSources }) => {
      const user = await dataSources.usersAPI.getById(id);

      if (!user) {
        throw new GraphQLError("User not found", {
          extensions: {
            code: "NOT_FOUND",
            argumentName: "id",
          },
        });
      }

      return user;
    },
  },
};
```

### Error Response Format

```json
{
  "errors": [
    {
      "message": "User not found",
      "locations": [{ "line": 2, "column": 3 }],
      "path": ["user"],
      "extensions": {
        "code": "NOT_FOUND",
        "argumentName": "id"
      }
    }
  ],
  "data": {
    "user": null
  }
}
```

### GraphQLError Options

```typescript
new GraphQLError(message, {
  // Custom error code
  extensions: {
    code: "CUSTOM_CODE",
    // Any additional metadata
    http: { status: 400 },
  },

  // Original error (for stack traces)
  originalError: caughtError,

  // AST node(s) associated with the error
  nodes: fieldNode,

  // Source location
  positions: [15],

  // Path to the field that caused the error
  path: ["user", "email"],
});
```

## Error Codes

### Built-in Codes

| Code                            | Description                       | HTTP Status |
| ------------------------------- | --------------------------------- | ----------- |
| `GRAPHQL_PARSE_FAILED`          | Syntax error in query             | 400         |
| `GRAPHQL_VALIDATION_FAILED`     | Query doesn't match schema        | 400         |
| `BAD_USER_INPUT`                | Invalid argument value            | 400         |
| `UNAUTHENTICATED`               | Missing or invalid authentication | 401         |
| `FORBIDDEN`                     | Not authorized for this operation | 403         |
| `PERSISTED_QUERY_NOT_FOUND`     | APQ hash not found                | 404         |
| `PERSISTED_QUERY_NOT_SUPPORTED` | Server doesn't support APQ        | 400         |
| `OPERATION_RESOLUTION_FAILURE`  | Operation couldn't be determined  | 400         |
| `BAD_REQUEST`                   | Invalid request format            | 400         |
| `INTERNAL_SERVER_ERROR`         | Unexpected server error           | 500         |

### Custom Error Classes

```typescript
import { GraphQLError } from "graphql";

export class NotFoundError extends GraphQLError {
  constructor(resource: string, id: string) {
    super(`${resource} with id '${id}' not found`, {
      extensions: {
        code: "NOT_FOUND",
        resource,
        id,
      },
    });
  }
}

export class AuthenticationError extends GraphQLError {
  constructor(message = "Not authenticated") {
    super(message, {
      extensions: {
        code: "UNAUTHENTICATED",
        http: { status: 401 },
      },
    });
  }
}

export class ForbiddenError extends GraphQLError {
  constructor(message = "Not authorized") {
    super(message, {
      extensions: {
        code: "FORBIDDEN",
        http: { status: 403 },
      },
    });
  }
}

export class ValidationError extends GraphQLError {
  constructor(message: string, field: string) {
    super(message, {
      extensions: {
        code: "BAD_USER_INPUT",
        field,
      },
    });
  }
}

// Usage
throw new NotFoundError("User", id);
throw new AuthenticationError();
throw new ForbiddenError("Admin access required");
throw new ValidationError("Email is invalid", "email");
```

### Setting HTTP Status

```typescript
throw new GraphQLError("Not authenticated", {
  extensions: {
    code: "UNAUTHENTICATED",
    http: {
      status: 401,
      headers: new Map([["WWW-Authenticate", "Bearer"]]),
    },
  },
});
```

## formatError Option

Use `formatError` to transform errors before sending to clients.

### Basic Formatting

```typescript
const server = new ApolloServer({
  typeDefs,
  resolvers,
  formatError: (formattedError, error) => {
    // formattedError: already formatted GraphQL error
    // error: original error (may be wrapped)

    console.error("GraphQL Error:", error);

    return formattedError;
  },
});
```

### Masking Internal Errors

```typescript
const server = new ApolloServer({
  typeDefs,
  resolvers,
  formatError: (formattedError, error) => {
    // Log full error for debugging
    console.error(error);

    // Don't expose internal server errors
    if (formattedError.extensions?.code === "INTERNAL_SERVER_ERROR") {
      return {
        message: "An internal error occurred",
        extensions: {
          code: "INTERNAL_SERVER_ERROR",
        },
      };
    }

    // Remove stacktrace in production
    if (process.env.NODE_ENV === "production") {
      delete formattedError.extensions?.stacktrace;
    }

    return formattedError;
  },
});
```

### Adding Request Context

```typescript
const server = new ApolloServer({
  typeDefs,
  resolvers,
  formatError: (formattedError, error) => {
    // Add request ID for support
    return {
      ...formattedError,
      extensions: {
        ...formattedError.extensions,
        requestId: crypto.randomUUID(),
        timestamp: new Date().toISOString(),
      },
    };
  },
});
```

### Error Classification

```typescript
const server = new ApolloServer({
  typeDefs,
  resolvers,
  formatError: (formattedError, error) => {
    const code = formattedError.extensions?.code;

    // Client errors - return as-is
    const clientCodes = ["BAD_USER_INPUT", "UNAUTHENTICATED", "FORBIDDEN", "NOT_FOUND"];

    if (clientCodes.includes(code as string)) {
      return formattedError;
    }

    // Server errors - mask details
    return {
      message: "Something went wrong",
      extensions: {
        code: "INTERNAL_SERVER_ERROR",
      },
    };
  },
});
```

## Production Error Handling

### Logging Strategy

```typescript
import { ApolloServerPlugin } from "@apollo/server";

const errorLoggingPlugin: ApolloServerPlugin = {
  async requestDidStart() {
    return {
      async didEncounterErrors({ errors, request, contextValue }) {
        for (const error of errors) {
          const code = error.extensions?.code;

          // Log all errors with context
          const logEntry = {
            message: error.message,
            code,
            path: error.path,
            operationName: request.operationName,
            userId: contextValue.user?.id,
            timestamp: new Date().toISOString(),
          };

          // Different log levels based on error type
          if (code === "INTERNAL_SERVER_ERROR") {
            console.error("Server Error:", logEntry, error.originalError);
          } else if (code === "UNAUTHENTICATED" || code === "FORBIDDEN") {
            console.warn("Auth Error:", logEntry);
          } else {
            console.info("Client Error:", logEntry);
          }
        }
      },
    };
  },
};
```

### Error Reporting Service

```typescript
import * as Sentry from "@sentry/node";

const sentryPlugin: ApolloServerPlugin<MyContext> = {
  async requestDidStart({ request, contextValue }) {
    return {
      async didEncounterErrors({ errors }) {
        for (const error of errors) {
          // Only report server errors
          if (error.extensions?.code === "INTERNAL_SERVER_ERROR") {
            Sentry.withScope((scope) => {
              scope.setTag("kind", "graphql");
              scope.setExtra("query", request.query);
              scope.setExtra("variables", request.variables);
              scope.setUser({ id: contextValue.user?.id });

              Sentry.captureException(error.originalError ?? error);
            });
          }
        }
      },
    };
  },
};
```

### Partial Data with Errors

GraphQL can return partial data alongside errors:

```typescript
// Schema
type Query {
  user(id: ID!): User
  posts: [Post!]!
}

// Query
query {
  user(id: "1") { name }
  posts { title }
}

// If user resolver throws but posts succeeds:
{
  "errors": [
    {
      "message": "User not found",
      "path": ["user"]
    }
  ],
  "data": {
    "user": null,
    "posts": [
      { "title": "Post 1" },
      { "title": "Post 2" }
    ]
  }
}
```

### Error Recovery Patterns

```typescript
const resolvers = {
  Query: {
    // Return null for optional fields instead of throwing
    user: async (_, { id }, { dataSources }) => {
      try {
        return await dataSources.usersAPI.getById(id);
      } catch (e) {
        console.error("Failed to fetch user:", e);
        return null;
      }
    },

    // Return empty array for list fields
    users: async (_, __, { dataSources }) => {
      try {
        return await dataSources.usersAPI.getAll();
      } catch (e) {
        console.error("Failed to fetch users:", e);
        return [];
      }
    },
  },

  User: {
    // Handle failures in field resolvers gracefully
    posts: async (parent, _, { dataSources }) => {
      try {
        return await dataSources.postsAPI.getByAuthor(parent.id);
      } catch (e) {
        console.error("Failed to fetch posts:", e);
        return [];
      }
    },
  },
};
```

### Validation Errors

```typescript
import { z } from "zod";

const CreateUserSchema = z.object({
  email: z.string().email(),
  name: z.string().min(1).max(100),
  age: z.number().int().positive().optional(),
});

const resolvers = {
  Mutation: {
    createUser: async (_, { input }, { dataSources }) => {
      const result = CreateUserSchema.safeParse(input);

      if (!result.success) {
        const errors = result.error.issues.map((issue) => ({
          field: issue.path.join("."),
          message: issue.message,
        }));

        throw new GraphQLError("Validation failed", {
          extensions: {
            code: "BAD_USER_INPUT",
            validationErrors: errors,
          },
        });
      }

      return dataSources.usersAPI.create(result.data);
    },
  },
};
```
