# Environment Variables

## Database Connection Profiles

Database connections can be configured via environment variables using two formats.

### Array format

A single variable holds a JSON array of connection profiles:

```bash
export WHODB_POSTGRES='[
  {"alias":"prod","host":"db.example.com","user":"postgres","password":"secret","database":"mydb","port":"5432"},
  {"alias":"staging","host":"staging-db.example.com","user":"postgres","password":"secret","database":"mydb","port":"5432"}
]'
```

### Numbered format

Each variable holds a single JSON object:

```bash
export WHODB_POSTGRES_1='{"alias":"prod","host":"db.example.com","user":"postgres","password":"secret","database":"mydb","port":"5432"}'
export WHODB_POSTGRES_2='{"alias":"staging","host":"staging-db.example.com","user":"postgres","password":"secret","database":"mydb","port":"5432"}'
```

### Supported prefixes

`WHODB_POSTGRES`, `WHODB_MYSQL`, `WHODB_MARIADB`, `WHODB_MONGODB`, `WHODB_REDIS`, `WHODB_CLICKHOUSE`, `WHODB_ELASTICSEARCH`, `WHODB_SQLITE3`

### JSON fields

| Field | Type | Description |
|---|---|---|
| `alias` | string | Display name for the connection |
| `host` | string | Hostname or IP address |
| `user` | string | Username |
| `password` | string | Password |
| `database` | string | Database name |
| `port` | string | Port number |
| `advanced` | object | Key-value map of advanced options (see below) |

The `advanced` field replaces the legacy `config` key. Both are accepted for backwards compatibility.

### Parsing logic

Defined in `core/src/envconfig/envconfig.go`. The array format (`WHODB_<TYPE>`) is checked first. If empty, numbered variables (`WHODB_<TYPE>_1`, `WHODB_<TYPE>_2`, ...) are read sequentially until a gap is found. The struct is `types.DatabaseCredentials` in `core/src/types/types.go`.

## Advanced Connection Options

These key-value pairs go in the `advanced` field of a connection profile. At runtime they are converted to `engine.Credentials.Advanced` (`[]engine.Record`) via `src.GetLoginCredentials()` in `core/src/src.go`.

### All databases

| Key | Default | Description |
|---|---|---|
| `Port` | varies per DB | Overrides the default port |

### SQL databases (Postgres, MySQL, MariaDB, ClickHouse)

Parsed in `core/src/plugins/gorm/db.go`.

| Key | Default | Description |
|---|---|---|
| `Port` | DB-specific | Connection port |
| `Parse Time` | `True` | Parse time values (MySQL/MariaDB) |
| `Loc` | `UTC` | Time zone location (MySQL/MariaDB) |
| `Allow clear text passwords` | `0` | Allow cleartext auth (MySQL/MariaDB) |
| `HTTP Protocol` | `disable` | Use HTTP protocol (ClickHouse) |
| `Readonly` | `disable` | Read-only mode (ClickHouse) |
| `Debug` | `disable` | Debug mode (ClickHouse) |
| `Connection Timeout` | `90` | Connection timeout in seconds |

### SSL/TLS options

Available for all database types. Parsed in `core/src/plugins/ssl/ssl.go`. See `ssl.md` for full SSL documentation.

| Key | Default | Description |
|---|---|---|
| `SSL Mode` | `disabled` | SSL mode: `disabled`, `require`, `verify-ca`, `verify-full` |
| `SSL CA Content` | `""` | CA certificate content (inline PEM) |
| `SSL CA Path` | `""` | Path to CA certificate file |
| `SSL Client Cert Content` | `""` | Client certificate content (inline PEM) |
| `SSL Client Cert Path` | `""` | Path to client certificate file |
| `SSL Client Key Content` | `""` | Client key content (inline PEM) |
| `SSL Client Key Path` | `""` | Path to client key file |
| `SSL Server Name` | hostname | Server name for certificate verification |

For each certificate, use either the `Content` or `Path` variant, not both. The `Path` variants are only available for preconfigured connection profiles (environment variables); the login form uses the `Content` variants.

### MongoDB

Parsed in `core/src/plugins/mongodb/db.go`.

| Key | Default | Description |
|---|---|---|
| `Port` | `27017` | Connection port |
| `URL Params` | `""` | Additional URL query parameters |
| `DNS Enabled` | `false` | Use `mongodb+srv://` scheme |

### Redis

Parsed in `core/src/plugins/redis/db.go`.

| Key | Default | Description |
|---|---|---|
| `Port` | `6379` | Connection port |

### Elasticsearch

Parsed in `core/src/plugins/elasticsearch/db.go`.

| Key | Default | Description |
|---|---|---|
| `Port` | `9200` | Connection port |

### Example with advanced options

```bash
export WHODB_POSTGRES_1='{
  "alias": "prod",
  "host": "db.example.com",
  "user": "postgres",
  "password": "secret",
  "database": "mydb",
  "port": "5433",
  "advanced": {
    "SSL Mode": "verify-ca",
    "SSL CA Path": "/path/to/ca.pem",
    "SSL Client Cert Path": "/path/to/client-cert.pem",
    "SSL Client Key Path": "/path/to/client-key.pem",
    "Connection Timeout": "30"
  }
}'
```

## AI Provider Variables

| Variable | Description |
|---|---|
| `WHODB_OLLAMA_HOST` | Ollama server hostname |
| `WHODB_OLLAMA_PORT` | Ollama server port |
| `WHODB_OPENAI_API_KEY` | OpenAI API key |
| `WHODB_OPENAI_ENDPOINT` | OpenAI API endpoint |
| `WHODB_ANTHROPIC_API_KEY` | Anthropic API key |
| `WHODB_ANTHROPIC_ENDPOINT` | Anthropic API endpoint |

### Generic AI providers

Configured via multiple variables per provider. See `core/src/envconfig/envconfig.go:ParseGenericProviders()`.

```bash
WHODB_AI_GENERIC_<ID>_NAME="Provider Display Name"
WHODB_AI_GENERIC_<ID>_TYPE="openai-generic"      # default if omitted
WHODB_AI_GENERIC_<ID>_BASE_URL="https://api.example.com/v1"
WHODB_AI_GENERIC_<ID>_API_KEY="sk-..."
WHODB_AI_GENERIC_<ID>_MODELS="model-1,model-2"
```

## Other Variables

| Variable | Description |
|---|---|
| `WHODB_LOG_FILE` | Redirect all logs to this file |
| `WHODB_ACCESS_LOG_FILE` | HTTP access log file path |
| `WHODB_AWS_PROVIDER` | JSON array of AWS provider configs (see `aws-integration.md`) |
