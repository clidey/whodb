---
name: query-builder
description: Convert natural language questions into SQL queries. Activates when users ask data questions in plain English like "show me users who signed up last week" or "find orders over $100".
---

# Query Builder

Convert natural language questions into SQL queries using the database schema.

## When to Use

Activate when user asks questions like:
- "Show me all users who signed up last month"
- "Find orders greater than $100"
- "Which products have low inventory?"
- "Get the top 10 customers by total spend"

## Workflow

### 1. Understand the Schema
Before generating SQL, always check the table structure:

```
whodb_tables(connection="...") → Get available tables
whodb_columns(table="relevant_table") → Get column names and types
```

### 2. Identify Intent
Parse the natural language request:
- **Subject**: What entity? (users, orders, products)
- **Filter**: What conditions? (last month, > $100, active)
- **Aggregation**: Count, sum, average, max, min?
- **Grouping**: By what dimension?
- **Ordering**: Sort by what? Ascending/descending?
- **Limit**: How many results?

### 3. Map to Schema
- Match entities to table names
- Match attributes to column names
- Identify foreign key joins needed

### 4. Generate SQL
Build the query following SQL best practices:

```sql
SELECT columns
FROM table
[JOIN other_table ON condition]
WHERE filters
[GROUP BY columns]
[HAVING aggregate_condition]
ORDER BY column [ASC|DESC]
LIMIT n;
```

### 5. Execute and Present
```
whodb_query(query="generated SQL")
```

## Translation Patterns

| Natural Language | SQL Pattern |
|------------------|-------------|
| "last week/month/year" | `WHERE date_col >= DATE_SUB(NOW(), INTERVAL 1 WEEK)` |
| "more than X" / "greater than X" | `WHERE col > X` |
| "top N" | `ORDER BY col DESC LIMIT N` |
| "how many" | `SELECT COUNT(*)` |
| "total" / "sum of" | `SELECT SUM(col)` |
| "average" | `SELECT AVG(col)` |
| "for each" / "by" | `GROUP BY col` |
| "between X and Y" | `WHERE col BETWEEN X AND Y` |
| "contains" / "like" | `WHERE col LIKE '%term%'` |
| "starts with" | `WHERE col LIKE 'term%'` |
| "is empty" / "is null" | `WHERE col IS NULL` |
| "is not empty" | `WHERE col IS NOT NULL` |

## Date Handling by Database

### PostgreSQL
```sql
WHERE created_at >= NOW() - INTERVAL '7 days'
WHERE created_at >= DATE_TRUNC('month', CURRENT_DATE)
```

### MySQL
```sql
WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
WHERE created_at >= DATE_FORMAT(NOW(), '%Y-%m-01')
```

### SQLite
```sql
WHERE created_at >= DATE('now', '-7 days')
WHERE created_at >= DATE('now', 'start of month')
```

## Examples

### "Show me users who signed up this month"
```sql
SELECT * FROM users
WHERE created_at >= DATE_TRUNC('month', CURRENT_DATE)
ORDER BY created_at DESC;
```

### "Find the top 5 products by sales"
```sql
SELECT p.name, SUM(oi.quantity) as total_sold
FROM products p
JOIN order_items oi ON p.id = oi.product_id
GROUP BY p.id, p.name
ORDER BY total_sold DESC
LIMIT 5;
```

### "How many orders per customer?"
```sql
SELECT customer_id, COUNT(*) as order_count
FROM orders
GROUP BY customer_id
ORDER BY order_count DESC;
```

## Safety Rules

- Always use LIMIT for exploratory queries (default: 100)
- Never generate DELETE, UPDATE, or DROP unless explicitly requested
- Warn if query might return large result sets
- Use table aliases for readability in JOINs
