# Troubleshooting Reference

## Table of Contents

- [Setup Issues](#setup-issues)
- [Schema Issues](#schema-issues)
- [Runtime Errors](#runtime-errors)
- [Performance Issues](#performance-issues)
- [Integration Issues](#integration-issues)
- [Debugging Tips](#debugging-tips)

## Setup Issues

### Module Not Found Errors

**Error:** `Cannot find module '@apollo/server'`

```bash
# Ensure correct packages are installed
npm install @apollo/server graphql

# For Express integration
npm install @apollo/server express graphql cors

# Clear node_modules and reinstall if issues persist
rm -rf node_modules package-lock.json
npm install
```

**Error:** `Cannot find module '@apollo/server/standalone'`

This is a subpath export. Ensure:

- Node.js v18+ (for native ESM subpath exports)
- TypeScript `moduleResolution` is `bundler`, `node16`, or `nodenext`

### TypeScript Configuration

**Error:** `Cannot use import statement outside a module`

```json
// package.json
{
  "type": "module"
}

// tsconfig.json
{
  "compilerOptions": {
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "esModuleInterop": true,
    "target": "ES2022"
  }
}
```

**Error:** `Property 'xxx' does not exist on type 'BaseContext'`

```typescript
// Define and use typed context
interface MyContext {
  user?: User;
  dataSources: DataSources;
}

const server = new ApolloServer<MyContext>({ typeDefs, resolvers });
```

### CommonJS Compatibility

Apollo Server 4 is ESM-first. For CommonJS projects:

```typescript
// Use dynamic import
const { ApolloServer } = await import('@apollo/server');

// Or configure tsconfig for interop
{
  "compilerOptions": {
    "module": "CommonJS",
    "esModuleInterop": true,
    "allowSyntheticDefaultImports": true
  }
}
```

## Schema Issues

### Unknown Type Error

**Error:** `Unknown type "User". Did you mean...`

```typescript
// Ensure all types are defined in typeDefs
const typeDefs = `#graphql
  type Query {
    user(id: ID!): User  # User must be defined
  }

  type User {  # Define the type
    id: ID!
    name: String!
  }
`;
```

### Missing Resolver

**Error:** `Cannot return null for non-nullable field Query.user`

```typescript
// Schema declares non-null
type Query {
  user(id: ID!): User!  # ! means non-null
}

// Resolver must return a value
const resolvers = {
  Query: {
    user: async (_, { id }, { dataSources }) => {
      const user = await dataSources.usersAPI.getById(id);
      if (!user) {
        throw new GraphQLError('User not found');  // Throw, don't return null
      }
      return user;
    },
  },
};
```

### Enum Mismatch

**Error:** `Enum "Status" cannot represent value: "draft"`

```typescript
// Schema defines uppercase
enum Status {
  DRAFT
  PUBLISHED
}

// But database returns lowercase
// Solution: Map enum values
const resolvers = {
  Status: {
    DRAFT: 'draft',
    PUBLISHED: 'published',
  },
  // Or transform in resolver
  Post: {
    status: (parent) => parent.status.toUpperCase(),
  },
};
```

### Input Type Errors

**Error:** `Variable "$input" got invalid value`

```graphql
# Schema
input CreateUserInput {
  email: String!
  name: String!
}

# Query - ensure variable matches input type exactly
mutation CreateUser($input: CreateUserInput!) {
  createUser(input: $input) { id }
}

# Variables - must match schema structure
{
  "input": {
    "email": "test@example.com",
    "name": "Test User"
  }
}
```

## Runtime Errors

### Context Undefined

**Error:** `Cannot read properties of undefined (reading 'user')`

```typescript
// Ensure context function returns complete object
const context = async ({ req }) => {
  // Always return all expected properties
  return {
    user: (await getUser(req.headers.authorization)) ?? null,
    dataSources: {
      usersAPI: new UsersAPI(),
    },
  };
};
```

### Async/Await Issues

**Error:** `[object Promise]` returned instead of actual data

```typescript
// Bad - missing await
const resolvers = {
  Query: {
    user: (_, { id }, { dataSources }) => {
      dataSources.usersAPI.getById(id); // Missing return/await
    },
  },
};

// Good - return promise or use async/await
const resolvers = {
  Query: {
    user: (_, { id }, { dataSources }) => {
      return dataSources.usersAPI.getById(id); // Return promise
    },
    // OR
    user: async (_, { id }, { dataSources }) => {
      return await dataSources.usersAPI.getById(id); // Async/await
    },
  },
};
```

### Circular Reference

**Error:** `Converting circular structure to JSON`

```typescript
// Avoid returning raw ORM objects with circular refs
const resolvers = {
  Query: {
    user: async (_, { id }) => {
      const user = await prisma.user.findUnique({
        where: { id },
        include: { posts: { include: { author: true } } }, // Circular!
      });

      // Transform to plain object
      return {
        id: user.id,
        name: user.name,
        posts: user.posts.map((p) => ({ id: p.id, title: p.title })),
      };
    },
  },
};
```

## Performance Issues

### N+1 Queries

**Symptom:** Slow queries, many database calls

```typescript
// Problem: Each user triggers separate posts query
const resolvers = {
  User: {
    posts: (parent) => db.posts.findByAuthor(parent.id), // N queries
  },
};

// Solution: Use DataLoader
import DataLoader from "dataloader";

const context = async () => ({
  loaders: {
    postsByAuthor: new DataLoader(async (authorIds) => {
      const posts = await db.posts.findByAuthorIds(authorIds); // 1 query
      return authorIds.map((id) => posts.filter((p) => p.authorId === id));
    }),
  },
});

const resolvers = {
  User: {
    posts: (parent, _, { loaders }) => loaders.postsByAuthor.load(parent.id),
  },
};
```

### Memory Leaks

**Symptom:** Memory usage grows over time

```typescript
// Problem: Shared data source instances
const sharedAPI = new UsersAPI(); // Wrong!
const context = async () => ({ dataSources: { usersAPI: sharedAPI } });

// Solution: Create per-request instances
const context = async () => ({
  dataSources: {
    usersAPI: new UsersAPI(), // New instance per request
  },
});

// Problem: DataLoader created once
const loader = new DataLoader(batchFn); // Wrong - caches forever!

// Solution: Create per-request DataLoaders
const context = async () => ({
  loaders: {
    userLoader: new DataLoader(batchFn), // New instance per request
  },
});
```

### Large Response Handling

**Symptom:** Timeout or memory errors on large queries

```typescript
// Add pagination
type Query {
  users(limit: Int = 10, offset: Int = 0): [User!]!
}

// Limit query depth
import depthLimit from 'graphql-depth-limit';

const server = new ApolloServer({
  typeDefs,
  resolvers,
  validationRules: [depthLimit(5)],
});

// Add query complexity limit
import { createComplexityLimitRule } from 'graphql-validation-complexity';

const server = new ApolloServer({
  typeDefs,
  resolvers,
  validationRules: [createComplexityLimitRule(1000)],
});
```

## Integration Issues

### CORS Errors

**Error:** `Access-Control-Allow-Origin` header missing

```typescript
import cors from "cors";

// Express integration - add cors before middleware
app.use(
  "/graphql",
  cors({
    origin: ["http://localhost:3000", "https://myapp.com"],
    credentials: true,
  }),
  express.json(),
  expressMiddleware(server),
);

// Standalone - configure cors option
const { url } = await startStandaloneServer(server, {
  listen: { port: 4000 },
  context: async ({ req }) => ({
    /* ... */
  }),
  // Standalone has basic CORS enabled by default
});
```

### Body Parser Issues

**Error:** `req.body` is undefined or empty

```typescript
// Express - ensure json middleware is before Apollo
app.use(express.json()); // Must come before expressMiddleware

app.use(
  "/graphql",
  cors(),
  express.json(), // JSON parser is required
  expressMiddleware(server),
);
```

### WebSocket / Subscriptions

**Error:** Subscriptions not working

```typescript
// Apollo Server 4 doesn't include subscription support by default
// Use graphql-ws package
import { WebSocketServer } from "ws";
import { useServer } from "graphql-ws/lib/use/ws";
import { makeExecutableSchema } from "@graphql-tools/schema";

const schema = makeExecutableSchema({ typeDefs, resolvers });

const wsServer = new WebSocketServer({
  server: httpServer,
  path: "/graphql",
});

useServer({ schema }, wsServer);
```

## Debugging Tips

### Enable Debug Logging

```typescript
const server = new ApolloServer({
  typeDefs,
  resolvers,
  plugins: [
    {
      async requestDidStart({ request }) {
        console.log("Query:", request.query);
        console.log("Variables:", JSON.stringify(request.variables, null, 2));

        return {
          async willSendResponse({ response }) {
            console.log("Response:", JSON.stringify(response.body, null, 2));
          },
        };
      },
    },
  ],
});
```

### Inspect Context

```typescript
const resolvers = {
  Query: {
    debug: (_, __, context) => {
      console.log("Context keys:", Object.keys(context));
      console.log("User:", context.user);
      return "Check server logs";
    },
  },
};
```

### Test Resolvers Directly

```typescript
// Unit test resolvers without server
import { resolvers } from "./resolvers";

describe("Query.user", () => {
  it("returns user by id", async () => {
    const mockDataSources = {
      usersAPI: {
        getById: jest.fn().mockResolvedValue({ id: "1", name: "Test" }),
      },
    };

    const result = await resolvers.Query.user(undefined, { id: "1" }, { dataSources: mockDataSources });

    expect(result).toEqual({ id: "1", name: "Test" });
  });
});
```

### Check Schema

```typescript
import { printSchema } from "graphql";
import { makeExecutableSchema } from "@graphql-tools/schema";

const schema = makeExecutableSchema({ typeDefs, resolvers });
console.log(printSchema(schema));
```

### Apollo Sandbox

Enable Apollo Sandbox for interactive debugging:

```typescript
import { ApolloServerPluginLandingPageLocalDefault } from "@apollo/server/plugin/landingPage/default";

const server = new ApolloServer({
  typeDefs,
  resolvers,
  plugins: [ApolloServerPluginLandingPageLocalDefault({ embed: true })],
});
```

Open `http://localhost:4000/graphql` in browser to access Sandbox.
