# Data Sources Reference

## Table of Contents

- [RESTDataSource](#restdatasource)
- [DataLoader](#dataloader)
- [Custom Data Sources](#custom-data-sources)
- [Best Practices](#best-practices)

## RESTDataSource

`@apollo/datasource-rest` provides a class for fetching data from REST APIs with built-in caching and request deduplication.

### Installation

```bash
npm install @apollo/datasource-rest
```

### Basic Usage

```typescript
import { RESTDataSource } from "@apollo/datasource-rest";

class UsersAPI extends RESTDataSource {
  override baseURL = "https://api.example.com/";

  async getUser(id: string): Promise<User> {
    return this.get<User>(`users/${id}`);
  }

  async getUsers(): Promise<User[]> {
    return this.get<User[]>("users");
  }

  async createUser(input: CreateUserInput): Promise<User> {
    return this.post<User>("users", { body: input });
  }

  async updateUser(id: string, input: UpdateUserInput): Promise<User> {
    return this.put<User>(`users/${id}`, { body: input });
  }

  async deleteUser(id: string): Promise<void> {
    return this.delete(`users/${id}`);
  }
}
```

### Context Integration

```typescript
interface MyContext {
  dataSources: {
    usersAPI: UsersAPI;
    postsAPI: PostsAPI;
  };
}

const server = new ApolloServer<MyContext>({ typeDefs, resolvers });

const { url } = await startStandaloneServer(server, {
  context: async () => ({
    dataSources: {
      usersAPI: new UsersAPI(),
      postsAPI: new PostsAPI(),
    },
  }),
});
```

### Request Customization

```typescript
class AuthenticatedAPI extends RESTDataSource {
  override baseURL = "https://api.example.com/";

  private token: string;

  constructor(token: string) {
    super();
    this.token = token;
  }

  // Add headers to every request
  override willSendRequest(path: string, request: AugmentedRequest) {
    request.headers["authorization"] = `Bearer ${this.token}`;
    request.headers["x-request-id"] = crypto.randomUUID();
  }

  // Transform response data
  override async didReceiveResponse<T>(response: Response, request: Request): Promise<T> {
    const body = await super.didReceiveResponse<T>(response, request);
    // Add metadata or transform
    return body;
  }
}
```

### Caching

RESTDataSource supports caching based on HTTP cache headers:

```typescript
class CachedAPI extends RESTDataSource {
  override baseURL = "https://api.example.com/";

  // Override cache options per request
  async getUser(id: string): Promise<User> {
    return this.get<User>(`users/${id}`, {
      cacheOptions: { ttl: 300 }, // 5 minutes
    });
  }

  // Disable caching for specific requests
  async getUserFresh(id: string): Promise<User> {
    return this.get<User>(`users/${id}`, {
      cacheOptions: { ttl: 0 },
    });
  }
}
```

### Error Handling

```typescript
import { GraphQLError } from "graphql";

class UsersAPI extends RESTDataSource {
  override baseURL = "https://api.example.com/";

  async getUser(id: string): Promise<User> {
    try {
      return await this.get<User>(`users/${id}`);
    } catch (error) {
      if (error.extensions?.response?.status === 404) {
        throw new GraphQLError("User not found", {
          extensions: { code: "NOT_FOUND" },
        });
      }
      throw error;
    }
  }
}
```

## DataLoader

DataLoader batches and caches individual loads within a single request, solving the N+1 problem.

### Installation

```bash
npm install dataloader
```

### Basic Usage

```typescript
import DataLoader from "dataloader";

// Batch function receives array of keys, returns array of results in same order
const userLoader = new DataLoader<string, User>(async (ids) => {
  const users = await db.query("SELECT * FROM users WHERE id IN (?)", [ids]);

  // IMPORTANT: Return results in same order as input ids
  const userMap = new Map(users.map((u) => [u.id, u]));
  return ids.map((id) => userMap.get(id) ?? new Error(`User ${id} not found`));
});

// Usage
const user1 = await userLoader.load("1");
const user2 = await userLoader.load("2");

// Both loads are batched into single query
```

### Context Integration

Create new DataLoader instances per request to prevent caching across requests:

```typescript
import DataLoader from "dataloader";

interface MyContext {
  loaders: {
    userLoader: DataLoader<string, User>;
    postsByAuthorLoader: DataLoader<string, Post[]>;
  };
}

const context = async (): Promise<MyContext> => ({
  loaders: {
    userLoader: new DataLoader(async (ids) => {
      const users = await db.users.findByIds(ids);
      const userMap = new Map(users.map((u) => [u.id, u]));
      return ids.map((id) => userMap.get(id) ?? null);
    }),

    postsByAuthorLoader: new DataLoader(async (authorIds) => {
      const posts = await db.posts.findByAuthorIds(authorIds);
      return authorIds.map((id) => posts.filter((p) => p.authorId === id));
    }),
  },
});

// Resolvers
const resolvers = {
  Post: {
    author: (post, _, { loaders }) => loaders.userLoader.load(post.authorId),
  },
  User: {
    posts: (user, _, { loaders }) => loaders.postsByAuthorLoader.load(user.id),
  },
};
```

### Options

```typescript
const loader = new DataLoader(batchFn, {
  // Maximum batch size (default: Infinity)
  maxBatchSize: 100,

  // Use object keys (default: false)
  cacheKeyFn: (key) => JSON.stringify(key),

  // Disable caching (default: true)
  cache: false,

  // Custom cache implementation
  cacheMap: new Map(),

  // Batch scheduling function
  batchScheduleFn: (callback) => setTimeout(callback, 10),
});
```

### Priming the Cache

```typescript
// Pre-populate cache after batch fetch
const users = await db.users.findAll();
users.forEach((user) => userLoader.prime(user.id, user));

// Clear specific key
userLoader.clear("1");

// Clear all
userLoader.clearAll();
```

## Custom Data Sources

Create custom data sources for databases, queues, or other services:

```typescript
class DatabaseSource {
  private db: Database;

  constructor(db: Database) {
    this.db = db;
  }

  async getUser(id: string): Promise<User | null> {
    return this.db.users.findUnique({ where: { id } });
  }

  async createUser(input: CreateUserInput): Promise<User> {
    return this.db.users.create({ data: input });
  }

  async getUsersWithPosts(ids: string[]): Promise<UserWithPosts[]> {
    return this.db.users.findMany({
      where: { id: { in: ids } },
      include: { posts: true },
    });
  }
}

// Context
const context = async () => ({
  dataSources: {
    db: new DatabaseSource(prisma),
  },
});
```

### With Connection Pooling

```typescript
class PooledDataSource {
  private pool: Pool;

  constructor(pool: Pool) {
    this.pool = pool;
  }

  async query<T>(sql: string, params: unknown[]): Promise<T[]> {
    const client = await this.pool.connect();
    try {
      const result = await client.query(sql, params);
      return result.rows;
    } finally {
      client.release();
    }
  }

  async getUser(id: string): Promise<User | null> {
    const rows = await this.query<User>("SELECT * FROM users WHERE id = $1", [id]);
    return rows[0] ?? null;
  }
}
```

## Best Practices

### Create Data Sources Per Request

```typescript
// Good - new instance per request
context: async () => ({
  dataSources: {
    usersAPI: new UsersAPI(),
  },
});

// Bad - shared instance across requests
const usersAPI = new UsersAPI();
context: async () => ({
  dataSources: { usersAPI },
});
```

### Combine DataLoader with RESTDataSource

```typescript
import DataLoader from "dataloader";

class UsersAPI extends RESTDataSource {
  override baseURL = "https://api.example.com/";

  private batchLoader = new DataLoader<string, User>(async (ids) => {
    const users = await this.get<User[]>("users", {
      params: { ids: ids.join(",") },
    });
    const userMap = new Map(users.map((u) => [u.id, u]));
    return ids.map((id) => userMap.get(id) ?? new Error(`Not found: ${id}`));
  });

  async getUser(id: string): Promise<User> {
    return this.batchLoader.load(id);
  }

  async getUsers(ids: string[]): Promise<User[]> {
    return this.batchLoader.loadMany(ids);
  }
}
```

### Handle Partial Failures

```typescript
const userLoader = new DataLoader<string, User>(async (ids) => {
  const users = await fetchUsers(ids);
  const userMap = new Map(users.map((u) => [u.id, u]));

  // Return Error for missing items, not null
  return ids.map((id) => userMap.get(id) ?? new Error(`User ${id} not found`));
});

// In resolver, handle the error
const resolvers = {
  Post: {
    author: async (post, _, { loaders }) => {
      try {
        return await loaders.userLoader.load(post.authorId);
      } catch (e) {
        // Return null for nullable field
        return null;
      }
    },
  },
};
```

### Avoid Over-fetching

```typescript
// Bad - always fetches all relations
async getUser(id: string): Promise<UserWithRelations> {
  return this.get(`users/${id}?include=posts,followers,following`);
}

// Good - separate methods for different needs
async getUser(id: string): Promise<User> {
  return this.get(`users/${id}`);
}

async getUserWithPosts(id: string): Promise<UserWithPosts> {
  return this.get(`users/${id}?include=posts`);
}
```
