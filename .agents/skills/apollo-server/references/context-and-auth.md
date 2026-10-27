# Context and Authentication Reference

## Table of Contents

- [Context Function](#context-function)
- [TypeScript Context Typing](#typescript-context-typing)
- [Authentication Patterns](#authentication-patterns)
- [Authorization Patterns](#authorization-patterns)
- [Data Sources in Context](#data-sources-in-context)
- [Security Best Practices](#security-best-practices)

## Context Function

The context function runs for every request and returns an object shared across all resolvers.

### Standalone Server

```typescript
import { ApolloServer } from "@apollo/server";
import { startStandaloneServer } from "@apollo/server/standalone";

const server = new ApolloServer({ typeDefs, resolvers });

const { url } = await startStandaloneServer(server, {
  context: async ({ req, res }) => {
    // req: IncomingMessage
    // res: ServerResponse
    return {
      token: req.headers.authorization,
    };
  },
});
```

### Express Middleware

```typescript
import { expressMiddleware } from "@as-integrations/express5";

app.use(
  "/graphql",
  cors(),
  express.json(),
  expressMiddleware(server, {
    context: async ({ req, res }) => {
      // req: express.Request
      // res: express.Response
      return {
        token: req.headers.authorization,
        ip: req.ip,
      };
    },
  }),
);
```

### Context Initialization Order

```typescript
const context = async ({ req }) => {
  // 1. Extract credentials
  const token = req.headers.authorization?.replace("Bearer ", "");

  // 2. Validate and decode (fail fast)
  let user = null;
  if (token) {
    try {
      user = await verifyToken(token);
    } catch (e) {
      // Don't throw - let resolvers handle auth
      console.warn("Invalid token:", e.message);
    }
  }

  // 3. Initialize data sources
  const dataSources = {
    usersAPI: new UsersDataSource(),
    postsAPI: new PostsDataSource(),
  };

  // 4. Return context object
  return { token, user, dataSources };
};
```

## TypeScript Context Typing

Define and use a typed context for type safety:

```typescript
import { ApolloServer } from "@apollo/server";

// Define context type
interface MyContext {
  token?: string;
  user?: {
    id: string;
    email: string;
    roles: string[];
  };
  dataSources: {
    usersAPI: UsersDataSource;
    postsAPI: PostsDataSource;
  };
}

// Pass to ApolloServer
const server = new ApolloServer<MyContext>({
  typeDefs,
  resolvers,
});

// Resolvers get typed context
const resolvers = {
  Query: {
    me: async (_, __, context: MyContext) => {
      if (!context.user) {
        throw new GraphQLError("Not authenticated");
      }
      return context.dataSources.usersAPI.getById(context.user.id);
    },
  },
};
```

## Authentication Patterns

### JWT Authentication

```typescript
import jwt from "jsonwebtoken";

interface JwtPayload {
  userId: string;
  email: string;
  roles: string[];
}

const context = async ({ req }) => {
  const token = req.headers.authorization?.replace("Bearer ", "");

  let user = null;
  if (token) {
    try {
      const decoded = jwt.verify(token, process.env.JWT_SECRET!) as JwtPayload;
      user = {
        id: decoded.userId,
        email: decoded.email,
        roles: decoded.roles,
      };
    } catch (e) {
      // Token invalid or expired - user remains null
    }
  }

  return { user };
};
```

### Session Authentication

```typescript
import session from "express-session";

// Express setup
app.use(
  session({
    secret: process.env.SESSION_SECRET!,
    resave: false,
    saveUninitialized: false,
  }),
);

app.use(
  "/graphql",
  cors(),
  express.json(),
  expressMiddleware(server, {
    context: async ({ req }) => ({
      user: req.session.user,
      session: req.session,
    }),
  }),
);

// Login mutation
const resolvers = {
  Mutation: {
    login: async (_, { email, password }, { session, dataSources }) => {
      const user = await dataSources.usersAPI.authenticate(email, password);
      if (!user) {
        throw new GraphQLError("Invalid credentials");
      }
      session.user = user;
      return user;
    },

    logout: async (_, __, { session }) => {
      return new Promise((resolve, reject) => {
        session.destroy((err) => {
          if (err) reject(err);
          else resolve(true);
        });
      });
    },
  },
};
```

### API Key Authentication

```typescript
const context = async ({ req }) => {
  const apiKey = req.headers["x-api-key"];

  let client = null;
  if (apiKey) {
    client = await db.apiKeys.findOne({ key: apiKey, active: true });
  }

  return {
    client,
    isAuthenticated: !!client,
  };
};
```

## Authorization Patterns

### Field-Level Authorization

```typescript
import { GraphQLError } from "graphql";

const resolvers = {
  User: {
    email: (parent, _, { user }) => {
      // Only return email to the user themselves or admins
      if (user?.id === parent.id || user?.roles.includes("admin")) {
        return parent.email;
      }
      return null;
    },

    privateData: (parent, _, { user }) => {
      if (!user) {
        throw new GraphQLError("Not authenticated", {
          extensions: { code: "UNAUTHENTICATED" },
        });
      }
      if (user.id !== parent.id) {
        throw new GraphQLError("Not authorized", {
          extensions: { code: "FORBIDDEN" },
        });
      }
      return parent.privateData;
    },
  },
};
```

### Role-Based Authorization

```typescript
// Helper function
function requireRole(user: User | null, roles: string[]): void {
  if (!user) {
    throw new GraphQLError("Not authenticated", {
      extensions: { code: "UNAUTHENTICATED" },
    });
  }

  const hasRole = roles.some((role) => user.roles.includes(role));
  if (!hasRole) {
    throw new GraphQLError(`Requires one of: ${roles.join(", ")}`, {
      extensions: { code: "FORBIDDEN" },
    });
  }
}

const resolvers = {
  Mutation: {
    deleteUser: async (_, { id }, { user, dataSources }) => {
      requireRole(user, ["admin"]);
      return dataSources.usersAPI.delete(id);
    },

    updatePost: async (_, { id, input }, { user, dataSources }) => {
      requireRole(user, ["admin", "editor"]);
      return dataSources.postsAPI.update(id, input);
    },
  },
};
```

### Directive-Based Authorization

```typescript
import { mapSchema, getDirective, MapperKind } from "@graphql-tools/utils";
import { defaultFieldResolver } from "graphql";

// Schema directive
const typeDefs = `#graphql
  directive @auth(requires: Role = USER) on FIELD_DEFINITION

  enum Role {
    ADMIN
    USER
    GUEST
  }

  type Query {
    users: [User!]! @auth(requires: ADMIN)
    me: User @auth
  }
`;

// Transform schema
function authDirectiveTransformer(schema) {
  return mapSchema(schema, {
    [MapperKind.OBJECT_FIELD]: (fieldConfig) => {
      const authDirective = getDirective(schema, fieldConfig, "auth")?.[0];

      if (authDirective) {
        const { requires } = authDirective;
        const { resolve = defaultFieldResolver } = fieldConfig;

        fieldConfig.resolve = async (source, args, context, info) => {
          if (!context.user) {
            throw new GraphQLError("Not authenticated");
          }

          if (requires && !context.user.roles.includes(requires)) {
            throw new GraphQLError(`Requires ${requires} role`);
          }

          return resolve(source, args, context, info);
        };
      }

      return fieldConfig;
    },
  });
}
```

## Data Sources in Context

### Creating Data Sources

```typescript
interface MyContext {
  dataSources: {
    usersAPI: UsersDataSource;
    postsAPI: PostsDataSource;
  };
}

const context = async ({ req }): Promise<MyContext> => {
  return {
    dataSources: {
      usersAPI: new UsersDataSource(),
      postsAPI: new PostsDataSource(),
    },
  };
};
```

### Passing User to Data Sources

```typescript
class AuthenticatedDataSource extends RESTDataSource {
  private user?: User;

  setUser(user: User) {
    this.user = user;
  }

  override willSendRequest(path: string, request: AugmentedRequest) {
    if (this.user) {
      request.headers["x-user-id"] = this.user.id;
    }
  }
}

const context = async ({ req }) => {
  const user = await getUser(req.headers.authorization);

  const usersAPI = new UsersDataSource();
  if (user) {
    usersAPI.setUser(user);
  }

  return { user, dataSources: { usersAPI } };
};
```

## Security Best Practices

### Never Trust Client Input

```typescript
const context = async ({ req }) => {
  // Bad - trusting client header
  const userId = req.headers["x-user-id"];

  // Good - verify token server-side
  const token = req.headers.authorization?.replace("Bearer ", "");
  const user = token ? await verifyToken(token) : null;

  return { user };
};
```

### Don't Expose Internal Errors

```typescript
const server = new ApolloServer({
  typeDefs,
  resolvers,
  formatError: (formattedError, error) => {
    // Log full error internally
    console.error(error);

    // Don't expose internal errors to clients
    if (formattedError.extensions?.code === "INTERNAL_SERVER_ERROR") {
      return {
        message: "Internal server error",
        extensions: { code: "INTERNAL_SERVER_ERROR" },
      };
    }

    return formattedError;
  },
});
```

### Rate Limiting

```typescript
import rateLimit from "express-rate-limit";

const limiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 100, // limit each IP to 100 requests per window
});

app.use("/graphql", limiter);
```

### Depth Limiting

```typescript
import depthLimit from "graphql-depth-limit";

const server = new ApolloServer({
  typeDefs,
  resolvers,
  validationRules: [depthLimit(10)],
});
```

### Query Complexity

```typescript
import { createComplexityLimitRule } from "graphql-validation-complexity";

const server = new ApolloServer({
  typeDefs,
  resolvers,
  validationRules: [
    createComplexityLimitRule(1000, {
      onCost: (cost) => console.log("Query cost:", cost),
    }),
  ],
});
```
