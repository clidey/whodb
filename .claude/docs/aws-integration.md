# AWS Integration

This document describes the AWS-specific implementation details for WhoDB. For the generic cloud provider architecture and how to add new providers (GCP, Azure), see [Cloud Providers](./cloud-providers.md).

The AWS integration allows WhoDB to auto-discover source resources from AWS
accounts and connect to them using the existing source catalog and plugin layer.

## Architecture Overview

AWS is implemented as a **Connection Provider**, not a separate source type. This means:

1. **AWS discovers source resources** (RDS, ElastiCache, DocumentDB, plus registered extensions such as S3)
2. **AWS generates credentials** that existing connectors/plugins understand
3. **The existing source/session flow connects** using those credentials

The MySQL plugin doesn't know or care if it's connecting to AWS RDS MySQL or a self-hosted MySQL - it just receives standard credentials.

```
┌─────────────────────────────────────────────────────────────────┐
│                        Connection Sources                        │
├─────────────────────────────────────────────────────────────────┤
│  Manual Entry (current)     │     AWS Provider (new)            │
│  - User enters hostname     │     - Auto-discovers resources    │
│  - User enters credentials  │     - Builds credentials for:     │
│                             │       • RDS → MySQL/Postgres/MariaDB│
│                             │       • ElastiCache → Redis        │
│                             │       • DocumentDB → MongoDB       │
│                             │       • Extensions → source types  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Existing Database Plugins                    │
│  MySQL │ PostgreSQL │ MariaDB │ Redis │ MongoDB │ ...           │
│  (unchanged - they receive standard engine.Credentials)         │
└─────────────────────────────────────────────────────────────────┘
```

## Package Structure

```
core/src/
├── aws/                      # AWS infrastructure layer
│   ├── config.go            # AWS SDK configuration
│   ├── credentials.go       # Credential parsing and provider building
│   ├── errors.go            # AWS error handling
│   ├── profiles.go          # Local AWS profile discovery
│   ├── regions.go           # AWS region definitions
│   └── *_test.go            # Tests
│
└── providers/               # Connection provider abstraction
    ├── provider.go         # ConnectionProvider interface
    ├── registry.go         # Provider registry
    └── aws/               # AWS provider implementation
        ├── provider.go    # Main AWS provider
        ├── rds.go        # RDS instance discovery
        ├── elasticache.go # ElastiCache cluster discovery
        ├── documentdb.go  # DocumentDB cluster discovery
        └── *_test.go      # Tests
```

## Key Interfaces

### ConnectionProvider

All connection sources implement this interface:

```go
type ConnectionProvider interface {
    Type() ProviderType
    ID() string
    Name() string
    DiscoverConnections(ctx context.Context) ([]DiscoveredConnection, error)
    TestConnection(ctx context.Context) error
    RefreshConnection(ctx context.Context, connectionID string) (bool, error)
    Close(ctx context.Context) error
}
```

### DiscoveredConnection

Represents a source discovered by a provider. Internally the provider layer
still stores an `engine.DatabaseType`; the GraphQL layer translates that into
the public `SourceType` string.

```go
type DiscoveredConnection struct {
    ID           string                // "providerID/resourceID"
    ProviderType ProviderType          // "aws", "manual"
    ProviderID   string                // Provider instance ID
    Name         string                // Display name
    DatabaseType engine.DatabaseType   // MySQL, Postgres, Redis, etc.
    Region       string                // Geographic location
    Status       ConnectionStatus      // available, starting, stopped, etc.
    Metadata     map[string]string     // Provider-specific details
}
```

## AWS Provider Configuration

```go
config := &aws.Config{
    ID:                  "aws-us-west-2",
    Name:                "Production AWS",
    Region:              "us-west-2",
    AuthMethod:          awsinfra.AuthMethodDefault, // or profile
    DiscoverRDS:         true,
    DiscoverElastiCache: true,
    DiscoverDocumentDB:  true,
    DiscoverS3:          true, // used when an S3 discovery extension is registered
}

provider, err := aws.New(config)
```

### Authentication Methods

| Method | Description | Use Case |
|--------|-------------|----------|
| `default` | AWS SDK credential chain | Recommended for most cases |
| `profile` | Named AWS profile | Multiple AWS accounts |

No credentials are stored in config.json. For explicit access keys, set `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` env vars (picked up by the `default` chain).

Discovery flags default to enabled for newly configured providers. Persisted
provider configs that predate a newer discovery flag should be treated as if
the new flag is enabled unless the user explicitly disables it later.

## Usage Example

```go
// Create provider
provider, _ := aws.New(&aws.Config{
    ID:     "aws-prod",
    Name:   "Production",
    Region: "us-west-2",
})

// Register with the global registry
registry := providers.GetDefaultRegistry()
registry.Register(provider)

// Discover all connections
ctx := context.Background()
connections, _ := registry.DiscoverAll(ctx)

// Connection metadata (endpoint, port, TLS) is exposed to frontend.
// Users select a connection in the UI, which prefills the login form.
// Users can then modify values (e.g., hostname to localhost for tunneling)
// before submitting via the standard LoginSource mutation.

// The source login resolver maps source types to connectors/plugins:
//   - ElastiCache → Redis plugin
//   - DocumentDB → MongoDB plugin
//   - S3 extension → S3 source connector
```

## Supported AWS Services

### RDS (Relational Database Service)

Discovers MySQL, PostgreSQL, and MariaDB instances including Aurora variants.

**Mapped to WhoDB types:**
- `mysql` → `MySQL`
- `mariadb` → `MariaDB`
- `postgres` → `Postgres`
- `aurora-mysql` → `MySQL`
- `aurora-postgresql` → `Postgres`

**IAM Authentication:** Supported when enabled on the RDS instance. The provider generates short-lived auth tokens (15-min validity).

### ElastiCache

Discovers Redis clusters (both standalone and replication groups).

**Note:** Memcached is not supported because WhoDB doesn't have a Memcached plugin.

**Supports:**
- Cluster mode enabled/disabled
- AUTH token authentication
- TLS encryption

### DocumentDB

Discovers MongoDB-compatible DocumentDB clusters.

**Note:** DocumentDB requires TLS and doesn't support all MongoDB features (e.g., no retryWrites).

## Error Handling

The `aws` package maps AWS SDK errors to user-friendly messages:

```go
var (
    ErrAccessDenied       // IAM permission issues
    ErrInvalidCredentials // Bad access key/secret
    ErrExpiredCredentials // Session token expired
    ErrResourceNotFound   // Resource doesn't exist
    ErrThrottling         // Rate limited
    ErrServiceUnavailable // AWS service issues
    ErrConnectionFailed   // Network issues
)

// Usage
if aws.IsRetryable(err) {
    // Safe to retry
}
```

## Testing

Tests are designed to run without AWS credentials:

```bash
# Run AWS infrastructure tests
go test ./src/aws/...

# Run provider tests
go test ./src/providers/...
```

## Security Considerations

1. **No credentials stored** - config.json contains only region, profile name, and discovery flags
2. **Use IAM roles** in production - Avoid static credentials on AWS infrastructure
3. **TLS required** - DocumentDB and optionally ElastiCache require TLS

## Environment Variables

Configure AWS providers via environment:

```bash
# Single provider with default auth
WHODB_AWS_PROVIDER='[{
  "name": "Production",
  "region": "us-west-2"
}]'

# Multiple regions with a named profile
WHODB_AWS_PROVIDER='[
  {"name": "US", "region": "us-west-2"},
  {"name": "EU", "region": "eu-west-1", "profileName": "eu-account"}
]'
```

## Future Extensions

These are planned but not yet implemented:

1. **Secrets Manager integration** - Retrieve database passwords from AWS Secrets Manager
2. **S3 export** - Export query results to S3
3. **CloudWatch metrics** - Show database performance metrics
