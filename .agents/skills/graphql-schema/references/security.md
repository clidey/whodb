# Security Best Practices

This reference covers security considerations for GraphQL schema design.

## Table of Contents

- [Introspection](#introspection)
- [Query Complexity](#query-complexity)
- [Depth Limiting](#depth-limiting)
- [Rate Limiting](#rate-limiting)
- [Field-Level Authorization](#field-level-authorization)
- [Input Validation](#input-validation)
- [Persisted Queries](#persisted-queries)
- [Information Disclosure](#information-disclosure)

## Introspection

### Disable in Production

Introspection reveals your entire schema. Disable it in production:

```typescript
// Apollo Server
const server = new ApolloServer({
  typeDefs,
  resolvers,
  introspection: process.env.NODE_ENV !== 'production',
});
```

### Allow for Development

Keep introspection enabled for:
- Development environments
- Internal tools
- Authorized clients (with authentication)

```typescript
const server = new ApolloServer({
  typeDefs,
  resolvers,
  introspection: process.env.NODE_ENV === 'development' ||
                 process.env.ENABLE_INTROSPECTION === 'true',
});
```

## Query Complexity

### Why Limit Complexity

A single query can request enormous amounts of data:

```graphql
# Potentially very expensive
query {
  users(first: 1000) {
    posts(first: 1000) {
      comments(first: 1000) {
        author {
          posts(first: 1000) {
            # ... could go deeper
          }
        }
      }
    }
  }
}
```

### Complexity Calculation

Assign costs to fields and limit total cost:

```graphql
type Query {
  users(first: Int): [User!]! @cost(complexity: 10, multipliers: ["first"])
}

type User {
  posts(first: Int): [Post!]! @cost(complexity: 5, multipliers: ["first"])
}
```

### Implementation with graphql-validation-complexity

```typescript
import { createComplexityLimitRule } from 'graphql-validation-complexity';

const complexityLimitRule = createComplexityLimitRule(1000, {
  scalarCost: 1,
  objectCost: 10,
  listFactor: 10,
});

const server = new ApolloServer({
  typeDefs,
  resolvers,
  validationRules: [complexityLimitRule],
});
```

### Cost Estimation in Schema

Document expected costs:

```graphql
"""
Returns user's posts.
Cost: Base 5 + (first * 2)
"""
type User {
  posts(
    first: Int = 20 @cost(weight: 2)
  ): PostConnection! @cost(complexity: 5)
}
```

## Depth Limiting

### Why Limit Depth

Prevent deeply nested queries:

```graphql
# Depth: 10+ levels deep
query {
  user {
    friends {
      friends {
        friends {
          friends {
            # ...
          }
        }
      }
    }
  }
}
```

### Implementation

```typescript
import depthLimit from 'graphql-depth-limit';

const server = new ApolloServer({
  typeDefs,
  resolvers,
  validationRules: [depthLimit(10)],
});
```

### Recommended Limits

| Application Type | Max Depth |
|-----------------|-----------|
| Simple API | 5-7 |
| Complex API | 7-10 |
| Internal tools | 10-15 |

## Rate Limiting

### Query-Based Rate Limiting

Limit queries per time window:

```typescript
// Example with express-rate-limit
const limiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 1000, // limit each IP to 1000 requests per window
});

app.use('/graphql', limiter);
```

### Complexity-Based Rate Limiting

Limit based on query cost, not just count:

```typescript
// Track complexity per user
const userComplexityBudget = new Map();

const complexityPlugin = {
  requestDidStart() {
    return {
      didResolveOperation({ request, document, context }) {
        const complexity = calculateComplexity(document);
        const userId = context.user?.id || request.http?.headers.get('x-forwarded-for');

        const current = userComplexityBudget.get(userId) || 0;
        if (current + complexity > MAX_COMPLEXITY_PER_MINUTE) {
          throw new GraphQLError('Rate limit exceeded');
        }
        userComplexityBudget.set(userId, current + complexity);
      }
    };
  }
};
```

### Field-Specific Rate Limiting

Limit expensive fields specifically:

```graphql
type Query {
  """
  Rate limited to 10 requests per minute
  """
  expensiveAnalytics: Analytics! @rateLimit(max: 10, window: "1m")
}
```

## Field-Level Authorization

### Schema Design for Authorization

Don't expose unauthorized fields in schema:

```graphql
# User-facing schema
type User {
  id: ID!
  name: String!
  publicProfile: PublicProfile!
}

# Admin-only schema (separate schema or schema stitching)
type User {
  id: ID!
  name: String!
  email: String!        # Admin only
  internalNotes: String # Admin only
}
```

### Resolver-Level Authorization

Check permissions in resolvers:

```typescript
const resolvers = {
  User: {
    email: (user, args, context) => {
      if (!context.user || context.user.id !== user.id) {
        if (!context.user?.isAdmin) {
          return null; // or throw error
        }
      }
      return user.email;
    },
    internalNotes: (user, args, context) => {
      if (!context.user?.isAdmin) {
        throw new GraphQLError('Not authorized', {
          extensions: { code: 'UNAUTHORIZED' }
        });
      }
      return user.internalNotes;
    }
  }
};
```

### Directive-Based Authorization

```graphql
directive @auth(requires: Role!) on FIELD_DEFINITION

enum Role {
  USER
  ADMIN
  SUPER_ADMIN
}

type User {
  id: ID!
  name: String!
  email: String! @auth(requires: USER)  # Own data or admin
  ssn: String @auth(requires: SUPER_ADMIN)
}
```

## Input Validation

### Schema-Level Validation

Use custom scalars for validation:

```graphql
scalar Email      # Validates email format
scalar URL        # Validates URL format
scalar DateTime   # Validates ISO 8601 format

type Mutation {
  createUser(
    email: Email!
    website: URL
    birthDate: DateTime!
  ): User!
}
```

### Input Constraints

Document and enforce constraints:

```graphql
input CreatePostInput {
  """
  Title of the post.
  Min length: 1
  Max length: 200
  """
  title: String!

  """
  Post content.
  Max length: 50000
  """
  content: String!

  """
  Tags for the post.
  Max items: 10
  """
  tags: [String!]
}
```

### Resolver Validation

Always validate in resolvers:

```typescript
const resolvers = {
  Mutation: {
    createPost: (_, { input }) => {
      if (input.title.length > 200) {
        throw new GraphQLError('Title too long', {
          extensions: { code: 'VALIDATION_ERROR', field: 'title' }
        });
      }
      if (input.content.length > 50000) {
        throw new GraphQLError('Content too long', {
          extensions: { code: 'VALIDATION_ERROR', field: 'content' }
        });
      }
      if (input.tags?.length > 10) {
        throw new GraphQLError('Too many tags', {
          extensions: { code: 'VALIDATION_ERROR', field: 'tags' }
        });
      }
      // ... create post
    }
  }
};
```

## Persisted Queries

### What Are Persisted Queries?

Map query hashes to pre-approved queries:

```
Client sends: { "extensions": { "persistedQuery": { "sha256Hash": "abc123..." }}}
Server looks up: abc123... → "query GetUser($id: ID!) { user(id: $id) { id name }}"
```

### Benefits

1. **Security**: Only allow approved queries
2. **Performance**: No parsing overhead
3. **Bandwidth**: Smaller payloads
4. **CDN**: Queries can be cached

### Implementation

```typescript
// Apollo Server with automatic persisted queries
import { ApolloServerPluginLandingPageDisabled } from '@apollo/server/plugin/disabled';

const server = new ApolloServer({
  typeDefs,
  resolvers,
  persistedQueries: {
    cache: new KeyValueCache(), // Your cache implementation
  },
});
```

### Strict Mode

In production, reject non-persisted queries:

```typescript
const server = new ApolloServer({
  typeDefs,
  resolvers,
  persistedQueries: {
    cache: persistedQueryCache,
  },
  allowBatchedHttpRequests: false,
  plugins: [
    {
      async requestDidStart() {
        return {
          async didResolveOperation({ request }) {
            if (!request.extensions?.persistedQuery) {
              throw new GraphQLError('Only persisted queries allowed');
            }
          }
        };
      }
    }
  ]
});
```

## Information Disclosure

### Error Messages

Don't leak internal details:

```typescript
// Bad: Exposes internal implementation
throw new Error(`Database error: SQLSTATE[23000]: duplicate key 'users_email_unique'`);

// Good: User-friendly message
throw new GraphQLError('Email already registered', {
  extensions: { code: 'EMAIL_EXISTS' }
});
```

### Stack Traces

Disable in production:

```typescript
const server = new ApolloServer({
  typeDefs,
  resolvers,
  includeStacktraceInErrorResponses: process.env.NODE_ENV === 'development',
});
```

### Schema Descriptions

Don't expose sensitive information in descriptions:

```graphql
# Bad: Exposes internal details
"""
User entity. Stored in PostgreSQL users table.
Synced with Salesforce via nightly cron job.
"""
type User { ... }

# Good: Public-facing description
"""
A user account in the system.
"""
type User { ... }
```

### Field Nullability and Errors

Consider what null vs error reveals:

```graphql
type User {
  # Returns null if no permission - reveals existence
  secretData: String

  # Returns error if no permission - might reveal existence
  secretData: String!
}

# Better: Same behavior whether exists or not
# Query returns null/error whether user exists or not
```
