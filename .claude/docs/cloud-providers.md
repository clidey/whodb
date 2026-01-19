# Cloud Provider Architecture

This document describes the generic cloud provider architecture for WhoDB. Cloud providers auto-discover database resources from cloud accounts (AWS, GCP, Azure) and connect to them using existing database plugins.

## Architecture Pattern: Interface + Union

The cloud provider system uses the **Interface + Union** pattern in GraphQL, which provides:

1. **Generic operations** that work across all providers
2. **Type-safe provider-specific inputs** for add/update operations
3. **O(1) complexity** when adding new providers (no existing code changes)

```
┌─────────────────────────────────────────────────────────────────┐
│                     GraphQL Schema                               │
├─────────────────────────────────────────────────────────────────┤
│  CloudProvider Interface         Provider-Specific Types         │
│  ┌──────────────────────┐       ┌──────────────────────┐        │
│  │ Id, Name, Region     │       │ AWSProvider          │        │
│  │ Status, Error        │◄──────│ + AuthMethod         │        │
│  │ DiscoveredCount      │       │ + ProfileName        │        │
│  │ ProviderType         │       │ + DiscoverRDS        │        │
│  └──────────────────────┘       └──────────────────────┘        │
│           ▲                              │                       │
│           │                     ┌────────┴────────┐              │
│           │                     │ GCPProvider     │  (future)    │
│           └─────────────────────│ + ProjectID     │              │
│                                 │ + DiscoverSQL   │              │
│                                 └─────────────────┘              │
└─────────────────────────────────────────────────────────────────┘
```

## Key Types

### CloudProviderType Enum

```graphql
enum CloudProviderType {
  AWS
  # GCP - future
  # Azure - future
}
```

### CloudProviderStatus Enum

```graphql
enum CloudProviderStatus {
  Connected
  Discovering
  Error
  Disconnected
  CredentialsRequired
}
```

### CloudProvider Interface

Common fields that all providers implement:

```graphql
interface CloudProvider {
  Id: ID!
  ProviderType: CloudProviderType!
  Name: String!
  Region: String!
  Status: CloudProviderStatus!
  LastDiscoveryAt: String
  DiscoveredCount: Int!
  Error: String
}
```

### DiscoveredConnection Type

Represents a database found by any provider:

```graphql
type DiscoveredConnection {
  Id: ID!
  ProviderType: CloudProviderType!
  ProviderID: String!
  Name: String!
  DatabaseType: String!
  Region: String!
  Status: String!
  Metadata: [Record!]!
}
```

## Generic vs Provider-Specific Operations

### Generic Operations (work with all providers)

| Operation | Description |
|-----------|-------------|
| `CloudProviders` | List all configured providers |
| `CloudProvider(id)` | Get a specific provider |
| `RemoveCloudProvider(id)` | Remove any provider |
| `TestCloudProvider(id)` | Test any provider connection |
| `RefreshCloudProvider(id)` | Refresh discovery for any provider |
| `DiscoveredConnections` | List all discovered connections |

### Provider-Specific Operations (type-safe inputs)

| Operation | Description |
|-----------|-------------|
| `AddAWSProvider(input)` | Add AWS provider with AWS-specific config |
| `UpdateAWSProvider(id, input)` | Update AWS provider |
| `AddGCPProvider(input)` | Add GCP provider (future) |

## Adding a New Cloud Provider

Follow these steps to add support for a new cloud provider (e.g., GCP):

### 1. Update GraphQL Schema (`core/graph/schema.graphqls`)

```graphql
# Add to CloudProviderType enum
enum CloudProviderType {
  AWS
  GCP  # Add new provider
}

# Create provider-specific type implementing CloudProvider
type GCPProvider implements CloudProvider {
  # Interface fields (required)
  Id: ID!
  ProviderType: CloudProviderType!
  Name: String!
  Region: String!
  Status: CloudProviderStatus!
  LastDiscoveryAt: String
  DiscoveredCount: Int!
  Error: String

  # GCP-specific fields
  ProjectID: String!
  ServiceAccountEmail: String
  DiscoverCloudSQL: Boolean!
  DiscoverMemorystore: Boolean!
  DiscoverBigtable: Boolean!
}

# Create provider-specific input
input GCPProviderInput {
  Name: String!
  Region: String!
  ProjectID: String!
  ServiceAccountKey: String  # JSON key file content
  DiscoverCloudSQL: Boolean
  DiscoverMemorystore: Boolean
  DiscoverBigtable: Boolean
}

# Add mutations (in Mutation type)
AddGCPProvider(input: GCPProviderInput!): GCPProvider
UpdateGCPProvider(id: ID!, input: GCPProviderInput!): GCPProvider
```

### 2. Regenerate Go Models

```bash
cd core && go generate ./...
```

### 3. Create Provider Package (`core/src/providers/gcp/`)

```
core/src/providers/gcp/
├── provider.go      # Main GCP provider implementation
├── cloudsql.go      # Cloud SQL discovery
├── memorystore.go   # Memorystore (Redis) discovery
└── provider_test.go # Tests
```

**provider.go:**

```go
package gcp

import (
    "context"
    "github.com/clidey/whodb/core/src/providers"
)

type Config struct {
    ID                 string
    Name               string
    Region             string
    ProjectID          string
    ServiceAccountKey  string
    DiscoverCloudSQL   bool
    DiscoverMemorystore bool
}

type Provider struct {
    config *Config
    // GCP clients
}

func New(config *Config) (*Provider, error) {
    // Initialize GCP clients
    return &Provider{config: config}, nil
}

func (p *Provider) Type() providers.ProviderType {
    return providers.ProviderTypeGCP
}

func (p *Provider) ID() string {
    return p.config.ID
}

func (p *Provider) DiscoverConnections(ctx context.Context) ([]providers.DiscoveredConnection, error) {
    var connections []providers.DiscoveredConnection

    if p.config.DiscoverCloudSQL {
        sqlConns, err := p.discoverCloudSQL(ctx)
        if err != nil {
            return nil, err
        }
        connections = append(connections, sqlConns...)
    }

    // Add other discovery methods...
    return connections, nil
}

// Implement remaining ConnectionProvider interface methods...
```

### 4. Update Resolvers (`core/graph/schema.resolvers.go`)

```go
// Add GCP-specific mutations
func (r *mutationResolver) AddGCPProvider(ctx context.Context, input model.GCPProviderInput) (*model.GCPProvider, error) {
    // Create and register GCP provider
}

func (r *mutationResolver) UpdateGCPProvider(ctx context.Context, id string, input model.GCPProviderInput) (*model.GCPProvider, error) {
    // Update GCP provider
}

// Update generic resolvers to handle GCP
func (r *queryResolver) CloudProviders(ctx context.Context) ([]model.CloudProvider, error) {
    // Return both AWS and GCP providers
}
```

### 5. Update Settings State (`core/src/settings/providers.go`)

Add GCP to the settings state management if needed.

### 6. Update Frontend

**Add GraphQL operations** (`frontend/src/pages/settings/gcp-providers.graphql`):

```graphql
mutation AddGCPProvider($input: GCPProviderInput!) {
  AddGCPProvider(input: $input) {
    Id
    ProviderType
    Name
    Region
    ProjectID
    # ... other fields
  }
}
```

**Create UI components** (`frontend/src/components/gcp/`):

```
frontend/src/components/gcp/
├── gcp-provider-modal.tsx  # Add/edit GCP provider
├── gcp-connection-picker.tsx
└── index.ts
```

**Update Redux store** - the generic `cloudProviders` array already supports multiple provider types via the `ProviderType` discriminator.

### 7. Add Localization

Create YAML files for GCP-specific translations:

```yaml
# frontend/src/locales/components/gcp-provider-modal.yaml
en:
  projectId: "GCP Project ID"
  serviceAccountKey: "Service Account Key"
  discoverCloudSQL: "Discover Cloud SQL"
```

### 8. Add Tests

- Unit tests for GCP provider package
- Integration tests for discovery
- E2E tests for UI flows

## File Locations

| Component | Location |
|-----------|----------|
| GraphQL Schema | `core/graph/schema.graphqls` |
| Go Models | `core/graph/model/models_gen.go` (generated) |
| Resolvers | `core/graph/schema.resolvers.go` |
| Provider Interface | `core/src/providers/provider.go` |
| Provider Registry | `core/src/providers/registry.go` |
| AWS Provider | `core/src/providers/aws/` |
| Settings State | `core/src/settings/providers.go` |
| Frontend Types | `frontend/src/generated/graphql.tsx` (generated) |
| Redux Store | `frontend/src/store/providers.ts` |
| AWS Components | `frontend/src/components/aws/` |

## Design Principles

1. **Generic where possible** - Use `CloudProvider` interface for shared operations
2. **Specific where needed** - Use provider-specific types for inputs requiring validation
3. **No switch statements** - Provider logic stays in provider packages, not shared code
4. **Discriminated unions** - Use `ProviderType` to identify provider at runtime
5. **Backwards compatible** - Adding providers doesn't change existing GraphQL queries

## Connection Prefill Rules

When a user selects a discovered cloud connection, the frontend prefills the login form with connection settings (hostname, port, SSL/TLS). These settings vary by database type.

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                  cloud-connection-prefill.ts                     │
├─────────────────────────────────────────────────────────────────┤
│  basePrefillRules (CE)                                           │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ DatabaseTypeA → { "Setting": "value" }                      │ │
│  │ DatabaseTypeB → { "TLS": "true" }   (conditional on meta)   │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                         ▼                                        │
│              merge at runtime (if EE loaded)                     │
│                         ▼                                        │
│  eePrefillRules (@ee/utils/cloud-prefill-rules.ts)               │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ EEDatabaseTypeA → { "EE Setting": "value" }                 │ │
│  │ EEDatabaseTypeB → { ... }                                   │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                         ▼                                        │
│              prefillRules (combined)                             │
│                         ▼                                        │
│              buildConnectionPrefill(conn)                        │
└─────────────────────────────────────────────────────────────────┘
```

### File Locations

| Component | Location |
|-----------|----------|
| CE Prefill Logic | `frontend/src/utils/cloud-connection-prefill.ts` |
| EE Prefill Rules | `ee/frontend/src/utils/cloud-prefill-rules.ts` |
| Picker Component | `frontend/src/components/aws/aws-connection-picker.tsx` |

### Adding a New Prefill Rule (CE)

Edit `frontend/src/utils/cloud-connection-prefill.ts`:

```typescript
const basePrefillRules: Record<string, PrefillRule> = {
    // Existing rules...

    // Simple rule - always apply these settings
    NewDatabaseType: () => ({ "Setting Name": "value" }),

    // Conditional rule - check metadata from discovery
    AnotherDatabaseType: (_, meta) =>
        meta("someMetadataKey") === "true" ? { "Conditional Setting": "value" } : {},
};
```

### Adding a New Prefill Rule (EE)

Edit `ee/frontend/src/utils/cloud-prefill-rules.ts`:

```typescript
export const eePrefillRules: Record<string, PrefillRule> = {
    // EE-only database types
    EEDatabaseType: () => ({ "EE Setting": "value" }),

    // Conditional based on metadata
    AnotherEEType: (_, meta) => {
        const customValue = meta("customMetadataKey");
        return customValue ? { "Custom Setting": customValue } : {};
    },
};
```

### How It Works

1. **Backend Discovery** - Provider discovers connections and populates `Metadata` map
2. **GraphQL Response** - `DiscoveredConnection.Metadata` contains allowed keys (endpoint, port, etc.)
3. **Frontend Prefill** - `buildConnectionPrefill(conn)` applies rules based on `DatabaseType`
4. **Login Form** - Receives prefill data and populates hostname, port, and advanced settings

### PrefillRule Function Signature

```typescript
type PrefillRule = (
    conn: LocalDiscoveredConnection,  // Full connection object
    meta: (key: string) => string | undefined  // Helper to read metadata
) => Record<string, string>;  // Advanced settings to apply
```

### Available Metadata Keys

The backend exposes these metadata keys for prefill decisions:

| Key | Description |
|-----|-------------|
| `endpoint` | Database hostname/endpoint |
| `port` | Database port |
| `transitEncryption` | Whether TLS is enabled (for cache services) |
| `serverless` | Whether instance is serverless |
| `iamAuthEnabled` | Whether IAM auth is available |
| `authTokenEnabled` | Whether auth token is enabled |

### Design Principles

1. **CE defines base rules** - Common database types in `basePrefillRules`
2. **EE extends, never modifies** - EE rules merge with CE, don't replace
3. **Rules are per-DatabaseType** - Each database type has explicit rules
4. **Metadata-driven** - Rules can inspect discovery metadata for conditional logic
5. **No provider knowledge in rules** - Rules don't know about AWS/GCP, only DatabaseType

## Related Documentation

- [AWS Integration](./aws-integration.md) - AWS-specific implementation details
- [Plugin Architecture](./plugin-architecture.md) - Database plugin pattern (similar concept)
