# Resolvers Reference

## Table of Contents

- [Resolver Signature](#resolver-signature)
- [Resolver Map Structure](#resolver-map-structure)
- [Async Resolvers](#async-resolvers)
- [Field Resolvers](#field-resolvers)
- [Default Resolvers](#default-resolvers)
- [N+1 Problem](#n1-problem)
- [Best Practices](#best-practices)

## Resolver Signature

Every resolver receives four positional arguments:

```typescript
(parent, args, contextValue, info) => result;
```

### Arguments

| Argument       | Description                                                                               |
| -------------- | ----------------------------------------------------------------------------------------- |
| `parent`       | Return value of the parent resolver. For root types (Query, Mutation), this is undefined. |
| `args`         | Object containing all arguments passed to the field.                                      |
| `contextValue` | Shared object for all resolvers in a request. Contains auth, dataSources, etc.            |
| `info`         | Contains field name, path, schema, and AST information. Rarely needed.                    |

### TypeScript Typing

```typescript
import { GraphQLResolveInfo } from "graphql";

type Resolver<TParent, TArgs, TContext, TResult> = (
  parent: TParent,
  args: TArgs,
  context: TContext,
  info: GraphQLResolveInfo,
) => TResult | Promise<TResult>;

// Example
const userResolver: Resolver<undefined, { id: string }, MyContext, User | null> = async (
  _,
  { id },
  { dataSources },
) => {
  return dataSources.usersAPI.getUser(id);
};
```

## Resolver Map Structure

Resolvers are organized by type and field name:

```typescript
const resolvers = {
  Query: {
    users: (_, __, { dataSources }) => dataSources.usersAPI.getAll(),
    user: (_, { id }, { dataSources }) => dataSources.usersAPI.getById(id),
  },

  Mutation: {
    createUser: (_, { input }, { dataSources }) => dataSources.usersAPI.create(input),
    updateUser: (_, { id, input }, { dataSources }) => dataSources.usersAPI.update(id, input),
    deleteUser: (_, { id }, { dataSources }) => dataSources.usersAPI.delete(id),
  },

  Subscription: {
    userCreated: {
      subscribe: (_, __, { pubsub }) => pubsub.asyncIterator(["USER_CREATED"]),
    },
  },

  User: {
    posts: (parent, _, { dataSources }) => dataSources.postsAPI.getByAuthor(parent.id),
    fullName: (parent) => `${parent.firstName} ${parent.lastName}`,
  },

  // Custom scalar
  DateTime: new GraphQLScalarType({
    name: "DateTime",
    serialize: (value) => value.toISOString(),
    parseValue: (value) => new Date(value),
  }),

  // Enum mapping (when values differ from schema)
  Status: {
    DRAFT: 0,
    PUBLISHED: 1,
    ARCHIVED: 2,
  },
};
```

## Async Resolvers

Resolvers can return Promises. Apollo Server awaits them automatically.

```typescript
const resolvers = {
  Query: {
    // Async/await pattern (recommended)
    user: async (_, { id }, { dataSources }) => {
      const user = await dataSources.usersAPI.getById(id);
      if (!user) {
        throw new GraphQLError("User not found", {
          extensions: { code: "NOT_FOUND" },
        });
      }
      return user;
    },

    // Promise pattern
    users: (_, __, { dataSources }) => {
      return dataSources.usersAPI.getAll();
    },

    // Parallel fetching
    dashboard: async (_, __, { dataSources }) => {
      const [users, posts, stats] = await Promise.all([
        dataSources.usersAPI.getAll(),
        dataSources.postsAPI.getRecent(),
        dataSources.analyticsAPI.getStats(),
      ]);
      return { users, posts, stats };
    },
  },
};
```

## Field Resolvers

Field resolvers compute derived values or fetch related data:

```typescript
const resolvers = {
  User: {
    // Computed field
    fullName: (parent) => `${parent.firstName} ${parent.lastName}`,

    // Related data (one-to-many)
    posts: async (parent, _, { dataSources }) => {
      return dataSources.postsAPI.getByAuthorId(parent.id);
    },

    // With arguments
    posts: async (parent, { limit, offset }, { dataSources }) => {
      return dataSources.postsAPI.getByAuthorId(parent.id, { limit, offset });
    },

    // Conditional fetching
    privateData: async (parent, _, { user }) => {
      if (user?.id !== parent.id) {
        return null;
      }
      return parent.privateData;
    },
  },

  Post: {
    // Related data (many-to-one)
    author: async (parent, _, { dataSources }) => {
      return dataSources.usersAPI.getById(parent.authorId);
    },
  },
};
```

## Default Resolvers

Apollo Server provides default resolvers that return `parent[fieldName]`:

```typescript
// Schema
type User {
  id: ID!
  name: String!
  email: String
}

// Resolvers - you don't need to write these
const resolvers = {
  User: {
    id: (parent) => parent.id,      // Default resolver handles this
    name: (parent) => parent.name,  // Default resolver handles this
    email: (parent) => parent.email, // Default resolver handles this
  },
};

// Only define resolvers when:
// 1. Field name differs from data property
// 2. Data needs transformation
// 3. Field requires fetching related data
const resolvers = {
  User: {
    fullName: (parent) => `${parent.first_name} ${parent.last_name}`,
  },
};
```

## N+1 Problem

The N+1 problem occurs when fetching related data triggers separate queries for each item.

### Problem Example

```typescript
// Query: { users { posts { title } } }
// This causes:
// 1 query for users
// N queries for posts (one per user)

const resolvers = {
  Query: {
    users: () => db.query("SELECT * FROM users"), // 1 query
  },
  User: {
    posts: (parent) => db.query("SELECT * FROM posts WHERE author_id = ?", [parent.id]), // N queries
  },
};
```

### Solution: DataLoader

```typescript
import DataLoader from "dataloader";

// Create loader per request (in context)
const context = async () => ({
  loaders: {
    postsByAuthor: new DataLoader(async (authorIds) => {
      const posts = await db.query("SELECT * FROM posts WHERE author_id IN (?)", [authorIds]);
      // Return posts grouped by author_id in same order as input
      return authorIds.map((id) => posts.filter((p) => p.author_id === id));
    }),
  },
});

// Use loader in resolver
const resolvers = {
  User: {
    posts: (parent, _, { loaders }) => loaders.postsByAuthor.load(parent.id),
  },
};
```

## Best Practices

### Keep Resolvers Thin

```typescript
// Bad - business logic in resolver
const resolvers = {
  Mutation: {
    createOrder: async (_, { input }, { dataSources, user }) => {
      const items = await dataSources.inventory.checkStock(input.items);
      const total = items.reduce((sum, item) => sum + item.price * item.qty, 0);
      const discount = user.isPremium ? total * 0.1 : 0;
      // ... more logic
    },
  },
};

// Good - delegate to service
const resolvers = {
  Mutation: {
    createOrder: async (_, { input }, { services, user }) => {
      return services.orders.create(input, user);
    },
  },
};
```

### Handle Null Appropriately

```typescript
const resolvers = {
  Query: {
    user: async (_, { id }, { dataSources }) => {
      // Return null for not found (matches nullable schema)
      return dataSources.usersAPI.getById(id);
    },

    userRequired: async (_, { id }, { dataSources }) => {
      const user = await dataSources.usersAPI.getById(id);
      if (!user) {
        throw new GraphQLError("User not found");
      }
      return user;
    },
  },
};
```

### Avoid Over-fetching in Parent

```typescript
// Bad - fetching all data upfront
const resolvers = {
  Query: {
    user: async (_, { id }) => {
      const user = await db.user.findById(id);
      const posts = await db.posts.findByAuthor(id);
      const followers = await db.followers.findByUser(id);
      return { ...user, posts, followers };
    },
  },
};

// Good - fetch only when requested
const resolvers = {
  Query: {
    user: (_, { id }) => db.user.findById(id),
  },
  User: {
    posts: (parent) => db.posts.findByAuthor(parent.id),
    followers: (parent) => db.followers.findByUser(parent.id),
  },
};
```
