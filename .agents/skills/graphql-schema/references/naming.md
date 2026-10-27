# Naming Conventions

This reference covers naming conventions for GraphQL schemas. Consistent naming makes APIs intuitive and self-documenting.

## Table of Contents

- [General Principles](#general-principles)
- [Types](#types)
- [Fields](#fields)
- [Arguments](#arguments)
- [Enums](#enums)
- [Mutations](#mutations)
- [Input Types](#input-types)
- [Anti-Patterns](#anti-patterns)

## General Principles

1. **Be Descriptive** - Names should clearly indicate purpose
2. **Be Consistent** - Follow the same patterns throughout
3. **Use Domain Language** - Match business terminology
4. **Avoid Abbreviations** - Prefer `createdAt` over `crtAt`
5. **Think Client-Side** - Name from consumer's perspective

## Types

### Object Types: PascalCase

Use singular nouns in PascalCase:

```graphql
# Correct
type User { ... }
type BlogPost { ... }
type ShoppingCart { ... }
type PaymentMethod { ... }

# Incorrect
type user { ... }        # lowercase
type Users { ... }       # plural
type blog_post { ... }   # snake_case
```

### Interface Types: PascalCase

Use adjectives or nouns describing capability:

```graphql
interface Node { ... }
interface Timestamped { ... }
interface Searchable { ... }
interface Commentable { ... }
```

### Union Types: PascalCase

Use nouns or compound names:

```graphql
union SearchResult = User | Post | Comment
union MediaContent = Image | Video | Audio
union NotificationTarget = User | Group | Channel
```

## Fields

### Object Fields: camelCase

Use camelCase, typically nouns or noun phrases:

```graphql
type User {
  id: ID!
  firstName: String!
  lastName: String!
  emailAddress: String!
  createdAt: DateTime!
  isActive: Boolean!
}
```

### Boolean Fields

Prefix with `is`, `has`, `can`, or `should`:

```graphql
type User {
  isActive: Boolean!
  isVerified: Boolean!
  hasSubscription: Boolean!
  canEdit: Boolean!
  shouldNotify: Boolean!
}

type Post {
  isPublished: Boolean!
  isArchived: Boolean!
  hasFeaturedImage: Boolean!
}
```

### Collection Fields

Use plural nouns:

```graphql
type User {
  posts: [Post!]!
  followers: [User!]!
  notifications: [Notification!]!
}

type Query {
  users: [User!]!
  allPosts: [Post!]!
}
```

### Relationship Fields

Name based on the relationship:

```graphql
type Post {
  author: User!         # Not: user, createdBy
  comments: [Comment!]!
  tags: [Tag!]!
}

type Comment {
  post: Post!           # Parent reference
  author: User!
  replies: [Comment!]!  # Child reference
}
```

### Computed Fields

Name by what they return, not how they're computed:

```graphql
type User {
  fullName: String!        # Not: getFullName, computedName
  postCount: Int!          # Not: calculatePostCount
  recentActivity: [Activity!]!
}
```

## Arguments

### Argument Names: camelCase

```graphql
type Query {
  user(id: ID!): User
  users(first: Int, after: String): UserConnection!
  search(query: String!, filters: SearchFilters): [SearchResult!]!
}
```

### Common Argument Patterns

```graphql
# Single item lookup
user(id: ID!): User
post(slug: String!): Post

# Filtering
users(role: Role, isActive: Boolean): [User!]!

# Pagination
posts(first: Int, after: String): PostConnection!
posts(last: Int, before: String): PostConnection!

# Sorting
posts(orderBy: PostOrderBy): [Post!]!

# Search
search(query: String!): [SearchResult!]!
```

### Avoid Generic Names

```graphql
# Avoid
posts(filter: JSON)
users(options: Options)

# Prefer
posts(status: PostStatus, authorId: ID)
users(role: Role, createdAfter: DateTime)
```

## Enums

### Enum Names: PascalCase

```graphql
enum UserRole { ... }
enum OrderStatus { ... }
enum SortDirection { ... }
```

### Enum Values: SCREAMING_SNAKE_CASE

```graphql
enum UserRole {
  ADMIN
  MODERATOR
  MEMBER
  GUEST
}

enum OrderStatus {
  PENDING_PAYMENT
  PAYMENT_RECEIVED
  PROCESSING
  SHIPPED
  DELIVERED
  CANCELLED
  REFUNDED
}

enum SortDirection {
  ASC
  DESC
}
```

## Mutations

### Mutation Names: verbNoun

Use action verbs followed by the subject:

```graphql
type Mutation {
  # Create operations
  createUser(input: CreateUserInput!): User!
  createPost(input: CreatePostInput!): Post!

  # Update operations
  updateUser(input: UpdateUserInput!): User!
  updatePost(input: UpdatePostInput!): Post!

  # Delete operations
  deleteUser(id: ID!): DeleteUserPayload!
  deletePost(id: ID!): DeletePostPayload!

  # Domain-specific operations
  publishPost(id: ID!): Post!
  archivePost(id: ID!): Post!

  sendMessage(input: SendMessageInput!): Message!

  addItemToCart(input: AddItemInput!): Cart!
  removeItemFromCart(itemId: ID!): Cart!

  followUser(userId: ID!): FollowPayload!
  unfollowUser(userId: ID!): UnfollowPayload!
}
```

### Common Verb Patterns

| Operation | Verbs |
|-----------|-------|
| Create | `create`, `add`, `register`, `submit` |
| Read | `get`, `fetch`, `load` (avoid in mutations) |
| Update | `update`, `edit`, `modify`, `set` |
| Delete | `delete`, `remove`, `archive` |
| State change | `publish`, `approve`, `reject`, `cancel` |
| Relationships | `add`, `remove`, `link`, `unlink` |
| Actions | `send`, `invite`, `follow`, `like` |

## Input Types

### Input Type Names

Use the mutation name + `Input`:

```graphql
input CreateUserInput {
  email: String!
  name: String!
}

input UpdateUserInput {
  id: ID!
  email: String
  name: String
}

input SendMessageInput {
  recipientId: ID!
  body: String!
}
```

### Payload Types

Use the mutation name + `Payload` or `Result`:

```graphql
type DeleteUserPayload {
  success: Boolean!
  deletedUserId: ID
}

type FollowUserPayload {
  follower: User!
  followee: User!
}
```

## Anti-Patterns

### Don't Use Hungarian Notation

```graphql
# Avoid
type TUser { ... }
type UserType { ... }
strName: String!
intAge: Int!

# Prefer
type User { ... }
name: String!
age: Int!
```

### Don't Use Redundant Prefixes

```graphql
# Avoid
type User {
  userId: ID!
  userName: String!
  userEmail: String!
}

# Prefer
type User {
  id: ID!
  name: String!
  email: String!
}
```

### Don't Expose Implementation Details

```graphql
# Avoid
type User {
  mysql_id: Int!
  redis_cache_key: String!
  getDerivedStateFromProps: JSON!
}

# Prefer
type User {
  id: ID!
  # Internal details should not appear in schema
}
```

### Don't Use Vague Names

```graphql
# Avoid
type Query {
  getData: JSON
  getInfo(type: String): JSON
  fetch(params: JSON): JSON
}

# Prefer
type Query {
  userProfile(userId: ID!): UserProfile
  orderHistory(first: Int): OrderConnection!
  searchProducts(query: String!): [Product!]!
}
```

### Don't Mix Conventions

```graphql
# Avoid: inconsistent naming
type User {
  firstName: String!    # camelCase
  last_name: String!    # snake_case
  EmailAddress: String! # PascalCase
}

# Prefer: consistent camelCase
type User {
  firstName: String!
  lastName: String!
  emailAddress: String!
}
```
