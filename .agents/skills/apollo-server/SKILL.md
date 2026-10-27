---
name: apollo-server
description: >
  Guide for building GraphQL servers with Apollo Server 5.x. Use this skill when:
  (1) setting up a new Apollo Server project,
  (2) writing resolvers or defining GraphQL schemas,
  (3) implementing authentication or authorization,
  (4) creating plugins or custom data sources,
  (5) troubleshooting Apollo Server errors or performance issues.
license: MIT
compatibility: Node.js v20+, TypeScript 4.7+. Works with Express v4/v5, standalone, Fastify, and serverless.
metadata:
  author: apollographql
  version: "1.0.0"
allowed-tools: Bash(npm:*) Bash(npx:*) Bash(node:*) Read Write Edit Glob Grep
---

# Apollo Server 5.x Guide

Apollo Server is an open-source GraphQL server that works with any GraphQL schema. Apollo Server 5 is framework-agnostic and runs standalone or integrates with Express, Fastify, and serverless environments.

## Quick Start

### Step 1: Install

```bash
npm install @apollo/server graphql
```

For Express integration:

```bash
npm install @apollo/server @as-integrations/express5 express graphql cors
```

### Step 2: Define Schema

```typescript
const typeDefs = `#graphql
  type Book {
    title: String
    author: String
  }

  type Query {
    books: [Book]
  }
`;
```

### Step 3: Write Resolvers

```typescript
const resolvers = {
  Query: {
    books: () => [
      { title: "The Great Gatsby", author: "F. Scott Fitzgerald" },
      { title: "1984", author: "George Orwell" },
    ],
  },
};
```

### Step 4: Start Server

**Standalone (Recommended for prototyping):**

The standalone server is great for prototyping, but for production services, we recommend integrating Apollo Server with a more fully-featured web framework such as Express, Koa, or Fastify. Swapping from the standalone server to a web framework later is straightforward.

```typescript
import { ApolloServer } from "@apollo/server";
import { startStandaloneServer } from "@apollo/server/standalone";

const server = new ApolloServer({ typeDefs, resolvers });

const { url } = await startStandaloneServer(server, {
  listen: { port: 4000 },
});

console.log(`Server ready at ${url}`);
```

**Express:**

```typescript
import { ApolloServer } from "@apollo/server";
import { expressMiddleware } from "@as-integrations/express5";
import { ApolloServerPluginDrainHttpServer } from "@apollo/server/plugin/drainHttpServer";
import express from "express";
import http from "http";
import cors from "cors";

const app = express();
const httpServer = http.createServer(app);

const server = new ApolloServer({
  typeDefs,
  resolvers,
  plugins: [ApolloServerPluginDrainHttpServer({ httpServer })],
});

await server.start();

app.use(
  "/graphql",
  cors(),
  express.json(),
  expressMiddleware(server, {
    context: async ({ req }) => ({ token: req.headers.authorization }),
  }),
);

await new Promise<void>((resolve) => httpServer.listen({ port: 4000 }, resolve));
console.log("Server ready at http://localhost:4000/graphql");
```

## Schema Definition

### Scalar Types

- `Int` - 32-bit integer
- `Float` - Double-precision floating-point
- `String` - UTF-8 string
- `Boolean` - true/false
- `ID` - Unique identifier (serialized as String)

### Type Definitions

```graphql
type User {
  id: ID!
  name: String!
  email: String
  posts: [Post!]!
}

type Post {
  id: ID!
  title: String!
  content: String
  author: User!
}

input CreatePostInput {
  title: String!
  content: String
}

type Query {
  user(id: ID!): User
  users: [User!]!
}

type Mutation {
  createPost(input: CreatePostInput!): Post!
}
```

### Enums and Interfaces

```graphql
enum Status {
  DRAFT
  PUBLISHED
  ARCHIVED
}

interface Node {
  id: ID!
}

type Article implements Node {
  id: ID!
  title: String!
}
```

## Resolvers Overview

Resolvers follow the signature: `(parent, args, contextValue, info)`

- **parent**: Result from parent resolver (root resolvers receive undefined)
- **args**: Arguments passed to the field
- **contextValue**: Shared context object (auth, dataSources, etc.)
- **info**: Field-specific info and schema details (rarely used)

```typescript
const resolvers = {
  Query: {
    user: async (_, { id }, { dataSources }) => {
      return dataSources.usersAPI.getUser(id);
    },
  },
  User: {
    posts: async (parent, _, { dataSources }) => {
      return dataSources.postsAPI.getPostsByAuthor(parent.id);
    },
  },
  Mutation: {
    createPost: async (_, { input }, { dataSources, user }) => {
      if (!user) throw new GraphQLError("Not authenticated");
      return dataSources.postsAPI.create({ ...input, authorId: user.id });
    },
  },
};
```

## Context Setup

Context is created per-request and passed to all resolvers.

```typescript
interface MyContext {
  token?: string;
  user?: User;
  dataSources: {
    usersAPI: UsersDataSource;
    postsAPI: PostsDataSource;
  };
}

const server = new ApolloServer<MyContext>({
  typeDefs,
  resolvers,
});

// Standalone
const { url } = await startStandaloneServer(server, {
  context: async ({ req }) => ({
    token: req.headers.authorization || "",
    user: await getUser(req.headers.authorization || ""),
    dataSources: {
      usersAPI: new UsersDataSource(),
      postsAPI: new PostsDataSource(),
    },
  }),
});

// Express middleware
expressMiddleware(server, {
  context: async ({ req, res }) => ({
    token: req.headers.authorization,
    user: await getUser(req.headers.authorization),
    dataSources: {
      usersAPI: new UsersDataSource(),
      postsAPI: new PostsDataSource(),
    },
  }),
});
```

## Reference Files

Detailed documentation for specific topics:

- [Resolvers](references/resolvers.md) - Resolver patterns and best practices
- [Context and Auth](references/context-and-auth.md) - Authentication and authorization
- [Plugins](references/plugins.md) - Server and request lifecycle hooks
- [Data Sources](references/data-sources.md) - RESTDataSource and DataLoader
- [Error Handling](references/error-handling.md) - GraphQLError and error formatting
- [Troubleshooting](references/troubleshooting.md) - Common issues and solutions

## Key Rules

### Schema Design

- Use **!** (non-null) for fields that always have values
- Prefer input types for mutations over inline arguments
- Use interfaces for polymorphic types
- Keep schema descriptions for documentation

### Resolver Best Practices

- Keep resolvers thin - delegate to services/data sources
- Always handle errors explicitly
- Use DataLoader for batching related queries
- Return partial data when possible (GraphQL's strength)

### Performance

- Use `@defer` and `@stream` for large responses
- Implement DataLoader to solve N+1 queries
- Consider persisted queries for production
- Use caching headers and CDN where appropriate

## Ground Rules

- ALWAYS use Apollo Server 5.x patterns (not v4 or earlier)
- ALWAYS type your context with TypeScript generics
- ALWAYS use `GraphQLError` from `graphql` package for errors
- NEVER expose stack traces in production errors
- PREFER `startStandaloneServer` for prototyping only
- USE an integration with a server framework like Express, Koa, Fastify, Next, etc. for production apps
- IMPLEMENT authentication in context, authorization in resolvers
