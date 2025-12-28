---
name: schema-designer
description: Help design database schemas, create tables, and plan data models. Activates when users ask to create tables, design schemas, or model data relationships.
---

# Schema Designer

Help users design database schemas, create tables, and model data relationships.

## When to Use

Activate when user asks:
- "Create a table for storing orders"
- "Design a schema for a blog"
- "Add a column to track user preferences"
- "How should I model this relationship?"

## Workflow

### 1. Understand Requirements
Ask clarifying questions:
- What data needs to be stored?
- What are the relationships between entities?
- What queries will be common?
- What's the expected data volume?

### 2. Check Existing Schema
```
whodb_tables() → See what already exists
whodb_columns(table="related_table") → Understand existing structure
```

### 3. Design the Schema
Follow database design principles:
- Normalize to reduce redundancy
- Use appropriate data types
- Define primary keys
- Establish foreign key relationships
- Add indexes for common queries

### 4. Generate DDL
Provide CREATE TABLE statements with explanations.

## Data Type Guidelines

### Identifiers
| Use Case | PostgreSQL | MySQL | SQLite |
|----------|------------|-------|--------|
| Auto-increment ID | `SERIAL` / `BIGSERIAL` | `INT AUTO_INCREMENT` | `INTEGER PRIMARY KEY` |
| UUID | `UUID` | `CHAR(36)` | `TEXT` |

### Text
| Use Case | PostgreSQL | MySQL | SQLite |
|----------|------------|-------|--------|
| Short text (<255) | `VARCHAR(n)` | `VARCHAR(n)` | `TEXT` |
| Long text | `TEXT` | `TEXT` | `TEXT` |
| Fixed length | `CHAR(n)` | `CHAR(n)` | `TEXT` |

### Numbers
| Use Case | PostgreSQL | MySQL | SQLite |
|----------|------------|-------|--------|
| Integer | `INTEGER` | `INT` | `INTEGER` |
| Big integer | `BIGINT` | `BIGINT` | `INTEGER` |
| Decimal (money) | `NUMERIC(10,2)` | `DECIMAL(10,2)` | `REAL` |
| Float | `REAL` | `FLOAT` | `REAL` |

### Dates
| Use Case | PostgreSQL | MySQL | SQLite |
|----------|------------|-------|--------|
| Date only | `DATE` | `DATE` | `TEXT` |
| Timestamp | `TIMESTAMP` | `DATETIME` | `TEXT` |
| With timezone | `TIMESTAMPTZ` | `TIMESTAMP` | `TEXT` |

### Boolean
| PostgreSQL | MySQL | SQLite |
|------------|-------|--------|
| `BOOLEAN` | `TINYINT(1)` | `INTEGER` |

## Common Patterns

### Users Table
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
```

### One-to-Many (Orders → Order Items)
```sql
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    status VARCHAR(20) DEFAULT 'pending',
    total NUMERIC(10,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id INTEGER NOT NULL REFERENCES products(id),
    quantity INTEGER NOT NULL DEFAULT 1,
    unit_price NUMERIC(10,2) NOT NULL
);

CREATE INDEX idx_order_items_order ON order_items(order_id);
```

### Many-to-Many (Users ↔ Roles)
```sql
CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE
);

CREATE TABLE user_roles (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);
```

### Soft Delete Pattern
```sql
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    deleted_at TIMESTAMP NULL,  -- NULL = not deleted
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Query active posts
SELECT * FROM posts WHERE deleted_at IS NULL;
```

### Audit Trail Pattern
```sql
CREATE TABLE audit_log (
    id SERIAL PRIMARY KEY,
    table_name VARCHAR(50) NOT NULL,
    record_id INTEGER NOT NULL,
    action VARCHAR(10) NOT NULL,  -- INSERT, UPDATE, DELETE
    old_values JSONB,
    new_values JSONB,
    user_id INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_table_record ON audit_log(table_name, record_id);
```

## Best Practices

1. **Always define PRIMARY KEY** - Every table needs one
2. **Use foreign keys** - Enforce referential integrity
3. **Add NOT NULL** - Unless the column is truly optional
4. **Create indexes** - On foreign keys and frequently queried columns
5. **Use appropriate types** - Don't store numbers as strings
6. **Add timestamps** - `created_at` and `updated_at` are almost always useful
7. **Name consistently** - `user_id` not `userId` or `UserID`
8. **Avoid reserved words** - Don't name columns `order`, `user`, `group`

## Migration Safety

When modifying existing tables:

```sql
-- Safe: Adding nullable column
ALTER TABLE users ADD COLUMN phone VARCHAR(20);

-- Safe: Adding column with default
ALTER TABLE users ADD COLUMN active BOOLEAN DEFAULT true;

-- Caution: Adding NOT NULL (requires default or backfill)
ALTER TABLE users ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'active';

-- Caution: Dropping column (data loss)
ALTER TABLE users DROP COLUMN old_column;

-- Caution: Changing type (may fail on existing data)
ALTER TABLE users ALTER COLUMN age TYPE INTEGER;
```
