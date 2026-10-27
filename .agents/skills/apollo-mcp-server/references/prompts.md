# MCP Prompts

## Overview

Apollo MCP Server supports MCP prompts — reusable prompt templates that AI clients can discover and use. Prompts are guided workflows that help AI models use your GraphQL tools more effectively.

No configuration is required. The server automatically detects a `prompts/` directory in the working directory and advertises prompt support to MCP clients.

## Defining Prompts

Store `.md` files in a `prompts/` directory:

```
your-project/
├── config.yaml
└── prompts/
    └── check_orders.md
```

### Prompt File Format

Each file has YAML frontmatter and a template body separated by `---`:

```markdown
---
name: check_orders
description: "Look up a user's recent orders and summarize them"
arguments:
  - name: email
    description: "The customer's email address"
    required: true
  - name: limit
    description: "Maximum number of orders to return"
---

1. Use GetUserByEmail(email={{email}}) to find the user
2. Use GetUserOrders to retrieve their recent orders{{limit}}
3. Summarize each order with its status, total, and date
```

### Frontmatter Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | Yes | Unique identifier for the prompt |
| `description` | `string` | No | Human-readable description for AI clients |
| `arguments` | list | No | Arguments the prompt accepts |

### Argument Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | `string` | — | Argument name, used as `{{name}}` placeholder |
| `description` | `string` | — | Description shown to AI clients |
| `required` | boolean | `false` | Whether the argument must be provided |

## Template Placeholders

Use `{{argument_name}}` in the template body. The server replaces each placeholder with the value provided in a `prompts/get` request. Unmatched placeholders remain unchanged.

## Behavior

- If the `prompts/` directory does not exist, the server starts normally without prompt support
- Required arguments are validated; missing ones return an error
- Each prompt returns a single user-role text message
- Multi-message and non-text content types are not currently supported
