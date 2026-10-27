# Pagination Design

This reference covers pagination patterns for GraphQL schemas, with focus on the cursor-based Connection pattern.

## Table of Contents

- [Pagination Approaches](#pagination-approaches)
- [Offset vs Cursor](#offset-vs-cursor)
- [Connection Pattern](#connection-pattern)
- [Building Connections](#building-connections)
- [Sorting and Filtering](#sorting-and-filtering)
- [Performance Considerations](#performance-considerations)

## Pagination Approaches

### Simple List (No Pagination)

Only use for small, bounded collections:

```graphql
type User {
  # OK: Users typically have few roles
  roles: [Role!]!

  # OK: Limited enum values
  permissions: [Permission!]!
}
```

### Offset-Based

Straightforward approach with limitations:

```graphql
type Query {
  posts(offset: Int = 0, limit: Int = 20): [Post!]!
}
```

### Cursor-Based (Connection Pattern)

Recommended for most cases:

```graphql
type Query {
  posts(first: Int, after: String): PostConnection!
}
```

## Offset vs Cursor

### Offset-Based Pagination

```graphql
type Query {
  posts(offset: Int = 0, limit: Int = 20): PostsPage!
}

type PostsPage {
  items: [Post!]!
  totalCount: Int!
  hasMore: Boolean!
}
```

**Pros:**
- Straightforward to build
- Allows jumping to specific page
- Familiar to REST developers

**Cons:**
- Inconsistent with real-time data (items shift)
- Poor performance on large offsets
- Duplicate or missing items when data changes

### Cursor-Based Pagination

```graphql
type Query {
  posts(first: Int, after: String): PostConnection!
}
```

**Pros:**
- Stable pagination (cursor points to specific item)
- Efficient for large datasets
- Works well with real-time updates
- Industry standard (Relay specification)

**Cons:**
- Can't jump to arbitrary page
- Requires more code to build
- Opaque cursors require explanation

## Connection Pattern

### Relay Connection Specification

The Connection pattern is defined by the Relay specification and widely adopted:

```graphql
type Query {
  posts(
    first: Int
    after: String
    last: Int
    before: String
  ): PostConnection!
}

type PostConnection {
  edges: [PostEdge!]!
  pageInfo: PageInfo!
  totalCount: Int
}

type PostEdge {
  node: Post!
  cursor: String!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}
```

### Connection Arguments

| Argument | Purpose |
|----------|---------|
| `first` | Number of items from the start |
| `after` | Cursor to start after (forward pagination) |
| `last` | Number of items from the end |
| `before` | Cursor to start before (backward pagination) |

**Usage patterns:**
- Forward: `first` + `after`
- Backward: `last` + `before`
- Don't mix forward and backward in same request

### Edge Type

Edges contain:
- `node`: The actual item
- `cursor`: Opaque cursor for this item
- Additional edge-specific data (optional)

```graphql
type PostEdge {
  node: Post!
  cursor: String!
  # Edge-specific metadata
  addedAt: DateTime
  addedBy: User
}
```

### PageInfo Type

```graphql
type PageInfo {
  hasNextPage: Boolean!      # More items forward?
  hasPreviousPage: Boolean!  # More items backward?
  startCursor: String        # Cursor of first item
  endCursor: String          # Cursor of last item
}
```

## Building Connections

### Basic Connection Query

```graphql
type Query {
  # Simple connection
  posts(first: Int, after: String): PostConnection!

  # Connection with filters
  userPosts(
    userId: ID!
    first: Int
    after: String
    status: PostStatus
  ): PostConnection!
}
```

### Connection for Relationships

```graphql
type User {
  id: ID!
  name: String!

  # Paginated relationship
  posts(first: Int, after: String): PostConnection!
  followers(first: Int, after: String): UserConnection!
  following(first: Int, after: String): UserConnection!
}
```

### Cursor Design

Cursors should be:
- **Opaque**: Clients shouldn't parse them
- **Stable**: Same cursor = same position
- **Serializable**: Usually base64-encoded

```typescript
// Common cursor strategies:

// 1. Encoded ID (simple)
const cursor = base64(`id:${post.id}`);

// 2. Encoded timestamp + ID (for sorted lists)
const cursor = base64(`${post.createdAt}:${post.id}`);

// 3. Encoded offset (simpler, but less stable)
const cursor = base64(`offset:${index}`);
```

### Default Page Size

Always set sensible defaults and limits:

```graphql
type Query {
  posts(
    first: Int = 20  # Default page size
    after: String
  ): PostConnection!
}
```

In resolver, enforce maximum:
```typescript
const resolvers = {
  Query: {
    posts: (_, { first = 20, after }) => {
      const limit = Math.min(first, 100); // Cap at 100
      // ...
    }
  }
};
```

## Sorting and Filtering

### Sorting Arguments

```graphql
enum PostOrderField {
  CREATED_AT
  UPDATED_AT
  TITLE
  POPULARITY
}

input PostOrder {
  field: PostOrderField!
  direction: OrderDirection!
}

enum OrderDirection {
  ASC
  DESC
}

type Query {
  posts(
    first: Int
    after: String
    orderBy: PostOrder = { field: CREATED_AT, direction: DESC }
  ): PostConnection!
}
```

### Filtering Arguments

```graphql
input PostFilter {
  status: PostStatus
  authorId: ID
  createdAfter: DateTime
  createdBefore: DateTime
  tags: [String!]
}

type Query {
  posts(
    first: Int
    after: String
    filter: PostFilter
    orderBy: PostOrder
  ): PostConnection!
}
```

### Combined Example

```graphql
type Query {
  posts(
    first: Int = 20
    after: String
    last: Int
    before: String
    filter: PostFilter
    orderBy: PostOrder
  ): PostConnection!
}

# Usage:
# query {
#   posts(
#     first: 10
#     filter: { status: PUBLISHED, tags: ["graphql"] }
#     orderBy: { field: CREATED_AT, direction: DESC }
#   ) {
#     edges {
#       node { id title }
#       cursor
#     }
#     pageInfo {
#       hasNextPage
#       endCursor
#     }
#   }
# }
```

## Performance Considerations

### Avoid COUNT(*) for totalCount

`totalCount` can be expensive. Options:

1. Make it nullable and skip when expensive
2. Return an estimate
3. Use a separate query for count

```graphql
type PostConnection {
  edges: [PostEdge!]!
  pageInfo: PageInfo!
  totalCount: Int  # Nullable - may not always be computed
}
```

### Use Efficient Cursors

Cursor-based pagination should use indexed columns:

```sql
-- Efficient: Uses index on created_at
SELECT * FROM posts
WHERE created_at < $cursor_timestamp
ORDER BY created_at DESC
LIMIT 20;

-- Inefficient: Full table scan
SELECT * FROM posts
LIMIT 20 OFFSET 10000;
```

### Limit Maximum Page Size

Prevent clients from requesting too many items:

```typescript
const MAX_PAGE_SIZE = 100;

function resolveConnection(first: number | null) {
  const limit = Math.min(first ?? 20, MAX_PAGE_SIZE);
  // ...
}
```

### Index Cursor Columns

Ensure database indexes exist for cursor columns:

```sql
CREATE INDEX idx_posts_created_at ON posts(created_at DESC);
CREATE INDEX idx_posts_author_created ON posts(author_id, created_at DESC);
```

### Consider Denormalization

For very large datasets, consider:
- Materialized views
- Denormalized count columns
- Cached aggregations
