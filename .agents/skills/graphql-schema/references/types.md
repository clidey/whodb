# Type Design Patterns

This reference covers type design patterns for building well-structured GraphQL schemas.

## Table of Contents

- [Schema-First Design](#schema-first-design)
- [Object Types](#object-types)
- [Nullability Strategy](#nullability-strategy)
- [ID Design](#id-design)
- [Interfaces](#interfaces)
- [Unions](#unions)
- [Input Types](#input-types)
- [Enums](#enums)
- [Custom Scalars](#custom-scalars)

## Schema-First Design

Design your schema before implementing resolvers. This ensures:

- API design focuses on client needs
- Implementation details don't leak into schema
- Team agreement on API contract

```graphql
# Start with what clients need
type Query {
  # Get a user's profile with recent activity
  userProfile(id: ID!): UserProfile

  # Search for content across the platform
  search(query: String!, type: SearchType): SearchResults!
}
```

## Object Types

### Single Responsibility

Each type should represent one clear concept:

```graphql
# Good: Focused types
type User {
  id: ID!
  email: String!
  profile: UserProfile!
}

type UserProfile {
  displayName: String!
  bio: String
  avatarUrl: String
}

# Avoid: Overloaded type
type User {
  id: ID!
  email: String!
  displayName: String!
  bio: String
  avatarUrl: String
  # ... mixing identity and profile concerns
}
```

### Field Cohesion

Group related fields. If fields always appear together, they belong together:

```graphql
type Address {
  street: String!
  city: String!
  state: String!
  postalCode: String!
  country: String!
}

type Order {
  id: ID!
  shippingAddress: Address!
  billingAddress: Address!
}
```

### Computed vs Stored Fields

Both computed and stored data should be fields. Clients don't care about storage:

```graphql
type Product {
  id: ID!
  name: String!
  priceInCents: Int!        # Stored
  formattedPrice: String!   # Computed
  inStock: Boolean!         # Computed from inventory
}
```

## Nullability Strategy

### Non-Null by Default

Make fields non-null unless there's a reason for null:

```graphql
type User {
  id: ID!           # Always exists
  email: String!    # Required field
  name: String      # Optional - user might not set
  deletedAt: DateTime  # Null means not deleted
}
```

### Valid Reasons for Nullable Fields

1. **Optional data** - User hasn't provided it
2. **Partial failure** - Resolver might fail for this field
3. **Permission-based** - Hidden if no access
4. **Semantic meaning** - Null means something (e.g., "not set")

### List Nullability

Always use **[Type!]!** for lists:

```graphql
type User {
  # Correct: non-null list, non-null items
  posts: [Post!]!

  # Return empty list, not null
  # Never return [null, post, null]
}
```

### Nullable for External Dependencies

If a field depends on an external service that might fail:

```graphql
type User {
  id: ID!
  email: String!

  # Might fail if recommendation service is down
  # Better to return null than fail entire query
  recommendedPosts: [Post!]
}
```

## ID Design

### Global IDs

Use globally unique IDs for entity identification:

```graphql
interface Node {
  "Globally unique identifier"
  id: ID!
}

type User implements Node {
  id: ID!  # e.g., "User:123" or base64("User:123")
}
```

### Base64 Encoding Pattern

Encode type and database ID together:

```
# Format: base64(TypeName:databaseId)
User:123 → VXNlcjoxMjM=
Post:456 → UG9zdDo0NTY=
```

Benefits:
- Globally unique across types
- Can determine type from ID
- Opaque to clients (discourages assumptions)

### Expose Database IDs Separately

If clients need the original ID:

```graphql
type User implements Node {
  id: ID!           # Global ID: "VXNlcjoxMjM="
  databaseId: Int!  # Original: 123
}
```

## Interfaces

### When to Use Interfaces

Use interfaces when types share common fields and behavior:

```graphql
interface Timestamped {
  createdAt: DateTime!
  updatedAt: DateTime!
}

type User implements Timestamped {
  id: ID!
  email: String!
  createdAt: DateTime!
  updatedAt: DateTime!
}

type Post implements Timestamped {
  id: ID!
  title: String!
  createdAt: DateTime!
  updatedAt: DateTime!
}
```

### Node Interface for Refetching

Implement `Node` for any type that can be fetched by ID:

```graphql
interface Node {
  id: ID!
}

type Query {
  node(id: ID!): Node
  nodes(ids: [ID!]!): [Node]!
}
```

### Multiple Interfaces

Types can implement multiple interfaces:

```graphql
interface Node {
  id: ID!
}

interface Timestamped {
  createdAt: DateTime!
  updatedAt: DateTime!
}

type Comment implements Node & Timestamped {
  id: ID!
  body: String!
  createdAt: DateTime!
  updatedAt: DateTime!
}
```

## Unions

### When to Use Unions

Use unions for mutually exclusive types that don't share fields:

```graphql
union SearchResult = User | Post | Comment

type Query {
  search(query: String!): [SearchResult!]!
}
```

### Unions vs Interfaces

| Use Case | Choice |
|----------|--------|
| Types share common fields | Interface |
| Types are mutually exclusive | Union |
| Polymorphic field return | Either (depends on shared fields) |
| Error handling patterns | Union (Result type) |

### Result Type Pattern

Use unions for operation results:

```graphql
type CreateUserSuccess {
  user: User!
}

type ValidationError {
  field: String!
  message: String!
}

type EmailAlreadyExists {
  existingUserId: ID!
}

union CreateUserResult = CreateUserSuccess | ValidationError | EmailAlreadyExists

type Mutation {
  createUser(input: CreateUserInput!): CreateUserResult!
}
```

## Input Types

### Structure Input Types

Group related inputs:

```graphql
input CreatePostInput {
  title: String!
  body: String!
  tags: [String!]
  publishAt: DateTime
}

input UpdatePostInput {
  id: ID!
  title: String
  body: String
  tags: [String!]
}

type Mutation {
  createPost(input: CreatePostInput!): Post!
  updatePost(input: UpdatePostInput!): Post!
}
```

### Optional Fields in Updates

Make update input fields nullable to allow partial updates:

```graphql
input UpdateUserInput {
  id: ID!            # Which user to update
  name: String       # Pass to change, omit to keep
  email: String      # Pass to change, omit to keep
  bio: String        # Pass to change, omit to keep
}
```

### Nested Input Types

Create nested inputs for complex structures:

```graphql
input AddressInput {
  street: String!
  city: String!
  state: String!
  postalCode: String!
  country: String!
}

input CreateOrderInput {
  items: [OrderItemInput!]!
  shippingAddress: AddressInput!
  billingAddress: AddressInput
}
```

## Enums

### Use Enums for Fixed Values

```graphql
enum OrderStatus {
  PENDING
  CONFIRMED
  SHIPPED
  DELIVERED
  CANCELLED
}

enum SortDirection {
  ASC
  DESC
}
```

### Document Enum Values

```graphql
enum Role {
  "Regular user with limited permissions"
  USER

  "Moderator with content management permissions"
  MODERATOR

  "Administrator with full system access"
  ADMIN
}
```

## Custom Scalars

### Common Custom Scalars

```graphql
scalar DateTime    # ISO 8601 date-time
scalar Date        # ISO 8601 date
scalar Time        # ISO 8601 time
scalar URL         # Valid URL string
scalar Email       # Valid email address
scalar JSON        # Arbitrary JSON (use sparingly)
scalar UUID        # UUID string
scalar BigInt      # Large integers beyond Int range
```

### When to Use Custom Scalars

Use custom scalars when:
- Built-in scalars don't capture the domain concept
- Validation at the schema level is valuable
- Serialization format matters (e.g., dates)

```graphql
type User {
  email: Email!       # Validated email format
  website: URL        # Validated URL format
  createdAt: DateTime!  # Consistent date format
}
```

### Don't Overuse JSON Scalar

Avoid `JSON` scalar except for truly dynamic data:

```graphql
# Avoid: Loses type safety
type Config {
  settings: JSON!
}

# Better: Define the structure
type Config {
  theme: Theme!
  notifications: NotificationSettings!
  privacy: PrivacySettings!
}
```
