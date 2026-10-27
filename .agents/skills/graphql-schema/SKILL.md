---
name: graphql-schema
description: >
  Guide for designing and changing GraphQL schemas following industry best practices. Use this skill
  whenever you create or edit GraphQL SDL, such as a `schema.graphql` file or `typeDefs`, including when:
  (1) designing a new GraphQL schema or API,
  (2) adding or changing types, fields, arguments, mutations, or descriptions in an existing schema,
  (3) reviewing existing schema for improvements,
  (4) deciding on type structures or nullability,
  (5) implementing pagination or error patterns,
  (6) ensuring security in schema design.
license: MIT
compatibility: Any GraphQL implementation (Apollo Server, graphql-js, Yoga, etc.)
metadata:
  author: apollographql
  version: "1.0.2"
allowed-tools: Bash(npm:*) Bash(npx:*) Read Write Edit Glob Grep
---

# GraphQL Schema Design Guide

This guide covers best practices for designing GraphQL schemas that are intuitive, performant, and maintainable. Schema design is primarily a server-side concern that directly impacts API usability.

## Schema Design Principles

### 1. Design for Client Needs

- Think about what queries clients will write
- Organize types around use cases, not database tables
- Expose capabilities, not implementation details

### 2. Be Explicit

- Use clear, descriptive names
- Make nullability intentional
- Document with descriptions

### 3. Design for Evolution

- Plan for backwards compatibility
- Use deprecation before removal
- Avoid breaking changes
- Give every new argument or input field a default value, or make it nullable. Existing clients don't send it, so a required field without a default makes their requests fail validation.

## Quick Reference

### Type Definition Syntax

```graphql
"""
A user in the system.
"""
type User {
  id: ID!
  email: String!
  name: String
  posts(first: Int = 10, after: String): PostConnection!
  createdAt: DateTime!
}
```

### Nullability Rules

| Pattern | Meaning |
|---------|---------|
| String | Nullable - may be null |
| String! | Non-null - always has value |
| [String] | Nullable list, nullable items |
| [String!] | Nullable list, non-null items |
| [String]! | Non-null list, nullable items |
| [String!]! | Non-null list, non-null items |

**Best Practice:** Use **[Type!]!** for lists - empty list over null, no null items.

### Input vs Output Types

```graphql
# Output type - what clients receive
type User {
  id: ID!
  email: String!
  createdAt: DateTime!
}

# Input type - what clients send
input CreateUserInput {
  email: String!
  name: String
}

# Mutation using input type
type Mutation {
  createUser(input: CreateUserInput!): User!
}
```

### Interface Pattern

```graphql
interface Node {
  id: ID!
}

type User implements Node {
  id: ID!
  email: String!
}

type Post implements Node {
  id: ID!
  title: String!
}
```

### Union Pattern

```graphql
union SearchResult = User | Post | Comment

type Query {
  search(query: String!): [SearchResult!]!
}
```

## Reference Files

Detailed documentation for specific topics:

- [Types](references/types.md) - Type design patterns, interfaces, unions, and custom scalars
- [Naming](references/naming.md) - Naming conventions for types, fields, and arguments
- [Pagination](references/pagination.md) - Connection pattern and cursor-based pagination
- [Errors](references/errors.md) - Error modeling and result types
- [Security](references/security.md) - Security best practices for schema design

## Key Rules

### Type Design

- Define types based on domain concepts, not data storage
- Use interfaces for shared fields across types
- Use unions for mutually exclusive types
- Keep types focused (single responsibility)
- Avoid deep nesting - flatten when possible

### Field Design

- Fields should be named from client's perspective
- Return the most specific type possible
- Make expensive fields explicit (consider arguments)
- Use arguments for filtering, sorting, pagination

### Mutation Design

- Use single input argument pattern: `mutation(input: InputType!)`
- Return affected objects in mutation responses
- Model mutations around business operations, not CRUD
- Consider returning a union of success/error types

### ID Strategy

- Use globally unique IDs when possible
- Implement `Node` interface for refetchability
- Base64-encode compound IDs if needed

## Ground Rules

- ALWAYS add descriptions to types and fields
- ALWAYS use non-null (**!**) for fields that cannot be null
- ALWAYS use **[Type!]!** pattern for lists
- ALWAYS paginate lists that can grow without limit, such as `Query.users` or `Post.comments`: return a connection and give `first` a default page size
- NEVER expose database internals in schema
- NEVER break backwards compatibility without deprecation
- NEVER add a required argument or input field without a default value
- PREFER dedicated input types over many arguments
- PREFER enums over arbitrary strings for fixed values
- USE `ID` type for identifiers, not `String` or `Int`
- USE custom scalars for domain-specific values (DateTime, Email, URL)
