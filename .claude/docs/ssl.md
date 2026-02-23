# SSL/TLS Configuration

This document covers SSL/TLS support in WhoDB, including architecture, configuration, and how to add SSL support to new database plugins.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         User Input                                   │
│  (Frontend form, Profile config, Environment variable)               │
│                              │                                       │
│                              ▼                                       │
│                    ┌─────────────────┐                              │
│                    │ NormalizeSSLMode │  ← Converts native names    │
│                    │ ("require" → "required")                       │
│                    └────────┬────────┘                              │
│                              │                                       │
│                              ▼                                       │
│                    ┌─────────────────┐                              │
│                    │ ValidateSSLMode │  ← Checks mode valid for DB  │
│                    └────────┬────────┘                              │
│                              │                                       │
│                              ▼                                       │
│                    ┌─────────────────┐                              │
│                    │   SSLConfig     │  ← Unified config struct     │
│                    │ Mode, CACert,   │                              │
│                    │ ClientCert, etc │                              │
│                    └────────┬────────┘                              │
│                              │                                       │
│                              ▼                                       │
│                    ┌─────────────────┐                              │
│                    │ BuildTLSConfig  │  ← Creates Go *tls.Config    │
│                    └────────┬────────┘                              │
│                              │                                       │
│                              ▼                                       │
│                    ┌─────────────────┐                              │
│                    │ Database Driver │  ← pgx, mysql, mongo, etc.   │
│                    └─────────────────┘                              │
└─────────────────────────────────────────────────────────────────────┘
```

## Key Files

| File | Purpose |
|------|---------|
| `core/src/plugins/ssl/ssl.go` | Mode definitions, validation, aliasing, registry |
| `core/src/plugins/ssl/tls.go` | `BuildTLSConfig()`, certificate loading |
| `core/src/plugins/gorm/db.go` | `parseSSLConfig()` for Gorm-based plugins |
| `core/src/engine/plugin.go` | `SSLStatus` type, `GetSSLStatus()` interface |
| `frontend/src/config/database-types.ts` | SSL modes per database (static, duplicated from backend) |
| `frontend/src/components/ssl-config.tsx` | SSL configuration UI component |

## SSL Modes

### Unified Mode Names

WhoDB uses unified SSL mode names across all databases:

| Mode | Constant | Description |
|------|----------|-------------|
| `disabled` | `SSLModeDisabled` | No SSL/TLS encryption |
| `preferred` | `SSLModePreferred` | Use TLS if server supports it (MySQL only) |
| `required` | `SSLModeRequired` | Require TLS, skip certificate verification |
| `verify-ca` | `SSLModeVerifyCA` | Verify server certificate against CA |
| `verify-identity` | `SSLModeVerifyIdentity` | Verify CA + server hostname |
| `enabled` | `SSLModeEnabled` | Simple TLS toggle with verification |
| `insecure` | `SSLModeInsecure` | TLS enabled, skip all verification |

### Database-Specific Mode Support

| Database | Supported Modes |
|----------|-----------------|
| PostgreSQL | `disabled`, `required`, `verify-ca`, `verify-identity` |
| MySQL/MariaDB | `disabled`, `preferred`, `required`, `verify-ca`, `verify-identity` |
| ClickHouse | `disabled`, `enabled`, `insecure` |
| MongoDB | `disabled`, `enabled`, `insecure` |
| Redis | `disabled`, `enabled`, `insecure` |
| Elasticsearch | `disabled`, `enabled`, `insecure` |

> **Note**: EE databases (Oracle, MSSQL, etc.) have SSL mode documentation in `ee/CLAUDE.md`.

## Mode Aliasing

Different databases use different native terminology. WhoDB normalizes these:

```go
// PostgreSQL native → WhoDB unified
"disable"     → "disabled"
"require"     → "required"
"verify-full" → "verify-identity"

// MySQL native → WhoDB unified
"DISABLED"        → "disabled"
"REQUIRED"        → "required"
"VERIFY_IDENTITY" → "verify-identity"
```

Aliases are defined in `ssl.go`:

```go
var sslModeAliases = map[engine.DatabaseType]map[string]SSLMode{
    engine.DatabaseType_Postgres: {
        "disable":     SSLModeDisabled,
        "require":     SSLModeRequired,
        "verify-full": SSLModeVerifyIdentity,
    },
    // ...
}
```

## Certificate Loading

Certificates can be provided two ways:

1. **Content** (frontend/API): Certificate PEM content sent directly
2. **Path** (profiles only): Server reads file from path

```go
type CertificateInput struct {
    Content string // PEM content (frontend sends this)
    Path    string // File path (profile-based only, admin-controlled)
}
```

**Security**: Path-based loading is restricted to profile connections to prevent path traversal attacks from untrusted frontend input.

## Adding SSL to a New Database Plugin

### Step 1: Register SSL Modes

**For CE databases** - Add to `databaseSSLModes` in `core/src/plugins/ssl/ssl.go`:

```go
var databaseSSLModes = map[engine.DatabaseType][]SSLModeInfo{
    // ... existing entries ...
    engine.DatabaseType_NewDB: {
        {SSLModeDisabled, "Disabled", "No SSL/TLS encryption"},
        {SSLModeEnabled, "Enabled", "Enable TLS with certificate verification"},
        {SSLModeInsecure, "Insecure", "Enable TLS, skip certificate verification"},
        // Add verify-ca, verify-identity if the database supports them
    },
}
```

**For EE databases** - Call in `ee/core/src/plugins/init.go`:

```go
func registerEESSLModes() {
    ssl.RegisterDatabaseSSLModes(ee_engine.DatabaseType_NewDB, []ssl.SSLModeInfo{
        {ssl.SSLModeDisabled, "Disabled", "No SSL/TLS encryption"},
        {ssl.SSLModeEnabled, "Enabled", "Enable TLS with certificate verification"},
        {ssl.SSLModeInsecure, "Insecure", "Enable TLS, skip certificate verification"},
    })
}
```

### Step 2: Register Aliases (if needed)

Only needed if the database has well-known native mode names that users might use:

**For CE** - Add to `sslModeAliases` in `ssl.go`:

```go
var sslModeAliases = map[engine.DatabaseType]map[string]SSLMode{
    // ... existing entries ...
    engine.DatabaseType_NewDB: {
        "native-off":    SSLModeDisabled,
        "native-on":     SSLModeEnabled,
    },
}
```

**For EE** - Call in init:

```go
ssl.RegisterSSLModeAliases(ee_engine.DatabaseType_NewDB, map[string]ssl.SSLMode{
    "native-off": ssl.SSLModeDisabled,
})
```

### Step 3: Parse SSL Config

**Option A: Gorm-based plugin** (extends `GormPlugin`)

The base class handles everything. Just use `connectionInput.SSLConfig`:

```go
func (p *NewDBPlugin) DB(config *engine.PluginConfig) (*gorm.DB, error) {
    connectionInput, err := p.ParseConnectionConfig(config)
    if err != nil {
        return nil, err
    }

    // connectionInput.SSLConfig is already parsed
    if connectionInput.SSLConfig != nil && connectionInput.SSLConfig.IsEnabled() {
        // Apply SSL...
    }
}
```

**Option B: Non-Gorm plugin** - Create a local `parseSSLConfig`:

```go
// parseSSLConfig extracts SSL configuration from advanced options.
// isProfile: if true, allows path-based certificate loading.
func parseSSLConfig(advanced []engine.Record, hostname string, isProfile bool) *ssl.SSLConfig {
    modeStr := common.GetRecordValueOrDefault(advanced, ssl.KeySSLMode, string(ssl.SSLModeDisabled))

    // Normalize database-native mode names
    mode := ssl.NormalizeSSLMode(engine.DatabaseType_NewDB, modeStr)

    // Validate the mode
    if !ssl.ValidateSSLMode(engine.DatabaseType_NewDB, mode) {
        log.Warnf("Invalid SSL mode '%s' for NewDB", modeStr)
        return nil
    }

    if mode == ssl.SSLModeDisabled {
        return nil
    }

    config := &ssl.SSLConfig{
        Mode: mode,
        CACert: ssl.CertificateInput{
            Content: common.GetRecordValueOrDefault(advanced, ssl.KeySSLCACertContent, ""),
        },
        ClientCert: ssl.CertificateInput{
            Content: common.GetRecordValueOrDefault(advanced, ssl.KeySSLClientCertContent, ""),
        },
        ClientKey: ssl.CertificateInput{
            Content: common.GetRecordValueOrDefault(advanced, ssl.KeySSLClientKeyContent, ""),
        },
        ServerName: common.GetRecordValueOrDefault(advanced, ssl.KeySSLServerName, ""),
    }

    // Path-based loading only for profiles (admin-controlled)
    if isProfile {
        config.CACert.Path = common.GetRecordValueOrDefault(advanced, ssl.KeySSLCACertPath, "")
        config.ClientCert.Path = common.GetRecordValueOrDefault(advanced, ssl.KeySSLClientCertPath, "")
        config.ClientKey.Path = common.GetRecordValueOrDefault(advanced, ssl.KeySSLClientKeyPath, "")
    }

    return config
}
```

### Step 4: Build and Apply TLS Config

In your `DB()` function:

```go
func (p *NewDBPlugin) DB(config *engine.PluginConfig) (*gorm.DB, error) {
    // ... parse connection config ...

    sslConfig := parseSSLConfig(config.Credentials.Advanced,
        config.Credentials.Hostname, config.Credentials.IsProfile)

    if sslConfig != nil && sslConfig.IsEnabled() {
        tlsConfig, err := ssl.BuildTLSConfig(sslConfig, config.Credentials.Hostname)
        if err != nil {
            log.WithError(err).Error("Failed to build TLS configuration")
            return nil, err
        }

        // Apply to your driver (varies by database):
        // pgx:      pgxConfig.TLSConfig = tlsConfig
        // mysql:    mysqldriver.RegisterTLSConfig(name, tlsConfig); cfg.TLSConfig = name
        // mongodb:  clientOptions.SetTLSConfig(tlsConfig)
        // redis:    opts.TLSConfig = tlsConfig
        // url-based: query.Set("encrypt", "true")
    }
}
```

### Step 5: Implement GetSSLStatus

Create `plugins/newdb/ssl_status.go`:

```go
package newdb

import (
    "github.com/clidey/whodb/core/src/engine"
    "github.com/clidey/whodb/core/src/log"
    "github.com/clidey/whodb/core/src/plugins/ssl"
)

func (p *NewDBPlugin) GetSSLStatus(config *engine.PluginConfig) (*engine.SSLStatus, error) {
    log.Debug("[SSL] NewDBPlugin.GetSSLStatus: checking SSL mode")

    sslConfig := parseSSLConfig(config.Credentials.Advanced,
        config.Credentials.Hostname, config.Credentials.IsProfile)

    if sslConfig == nil || !sslConfig.IsEnabled() {
        return &engine.SSLStatus{
            IsEnabled: false,
            Mode:      string(ssl.SSLModeDisabled),
        }, nil
    }

    return &engine.SSLStatus{
        IsEnabled: true,
        Mode:      string(sslConfig.Mode),
    }, nil
}
```

**Advanced**: For databases that can report actual SSL status (like PostgreSQL), query the connection:

```go
func (p *PostgresPlugin) GetSSLStatus(config *engine.PluginConfig) (*engine.SSLStatus, error) {
    return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (*engine.SSLStatus, error) {
        var sslInUse bool
        err := db.Raw("SELECT ssl FROM pg_stat_ssl WHERE pid = pg_backend_pid()").Scan(&sslInUse).Error
        // ...
    })
}
```

## Testing SSL

### Local Development

Use the SSL-enabled containers in `dev/docker-compose.yml`:

```bash
cd dev
docker-compose --profile ssl up -d e2e_postgres_ssl
```

SSL containers run on different ports:
- PostgreSQL SSL: 5433
- MySQL SSL: 3309
- MariaDB SSL: 3310
- MongoDB SSL: 27018
- Redis SSL: 6380
- ClickHouse SSL: 8443/9440
- Elasticsearch SSL: 9201

### Test Certificates

Development certificates are in `dev/certs/`:
- `ca/<db>/ca.pem` - CA certificate
- `server/<db>/server-cert.pem` - Server certificate
- `server/<db>/server-key.pem` - Server private key
- `client/<db>/client-cert.pem` - Client certificate (for mTLS)
- `client/<db>/client-key.pem` - Client private key (for mTLS)

### Verify SSL Connection

After connecting, check the SSL status indicator in the sidebar or query:

```sql
-- PostgreSQL
SELECT ssl, version FROM pg_stat_ssl WHERE pid = pg_backend_pid();

-- MySQL/MariaDB
SHOW SESSION STATUS LIKE 'Ssl_cipher';
```

## Checklist for New Database SSL Support

- [ ] SSL modes registered in backend (`ssl.go` or `RegisterDatabaseSSLModes`)
- [ ] SSL modes added to frontend (`database-types.ts` sslModes array)
- [ ] Aliases registered if database has native mode names (both backend and frontend)
- [ ] `parseSSLConfig()` implemented or using GormPlugin base
- [ ] `ssl.BuildTLSConfig()` called in DB connection
- [ ] TLS config applied to database driver correctly
- [ ] `GetSSLStatus()` implemented in `ssl_status.go`
- [ ] Logging added for SSL operations
- [ ] Backend builds: `cd core && go build .`
- [ ] Frontend type checks: `cd frontend && ./node_modules/.bin/tsc --noEmit`
- [ ] EE builds (if applicable): `go build -tags ee ./core`

## Frontend Integration

The `SSLConfig` component (`frontend/src/components/ssl-config.tsx`) automatically:
- Receives SSL modes via props from `database-types.ts`
- Handles mode aliasing (shows correct selection for aliased values)
- Shows/hides certificate inputs based on mode requirements
- Supports file picker and paste modes for certificates

### Intentional Duplication

**SSL modes are duplicated between frontend and backend.** This is intentional:

- **Backend** (`core/src/plugins/ssl/ssl.go`): Source of truth for validation during connection
- **Frontend** (`frontend/src/config/database-types.ts`): Display data for the login form dropdown

**Why duplicate?** SSL modes are defined by database protocols and rarely change. Duplicating them:
- Eliminates a network request on every login page load
- Avoids authentication complexity (modes needed before user logs in)
- Removes loading states and race conditions
- Simplifies the frontend component

**When adding a new database**, update both:
1. Backend: `ssl.go` mode registry (for validation)
2. Frontend: `database-types.ts` sslModes array (for display)
