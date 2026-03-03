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
| `Connection Timeout` | `90` | PostgreSQL, MySQL, MariaDB, ClickHouse. Connection timeout in seconds |

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

| Variable | Default | Description |
|---|---|---|
| `WHODB_OLLAMA_HOST` | `localhost` (`host.docker.internal` in Docker) | Ollama server hostname |
| `WHODB_OLLAMA_PORT` | `11434` | Ollama server port |
| `WHODB_OLLAMA_NAME` | unset | Display name for Ollama in the provider dropdown |
| `WHODB_OPENAI_API_KEY` | unset | OpenAI API key |
| `WHODB_OPENAI_ENDPOINT` | `https://api.openai.com/v1` | OpenAI API endpoint |
| `WHODB_OPENAI_NAME` | unset | Display name for OpenAI in the provider dropdown |
| `WHODB_ANTHROPIC_API_KEY` | unset | Anthropic API key |
| `WHODB_ANTHROPIC_ENDPOINT` | `https://api.anthropic.com/v1` | Anthropic API endpoint |
| `WHODB_ANTHROPIC_NAME` | unset | Display name for Anthropic in the provider dropdown |

### Generic AI providers

Connect any OpenAI-compatible provider. Configured via multiple variables per provider with a unique `<ID>`. See `core/src/envconfig/envconfig.go:ParseGenericProviders()`.

| Variable | Required | Default | Description |
|---|---|---|---|
| `WHODB_AI_GENERIC_<ID>_NAME` | No | `<ID>` | Display name in provider dropdown |
| `WHODB_AI_GENERIC_<ID>_TYPE` | No | `openai-generic` | Client type |
| `WHODB_AI_GENERIC_<ID>_BASE_URL` | Yes | | API base URL |
| `WHODB_AI_GENERIC_<ID>_API_KEY` | No | | API key |
| `WHODB_AI_GENERIC_<ID>_MODELS` | Yes | | Comma-separated list of model names |

## Server Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | TCP port WhoDB listens on |
| `WHODB_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error`, `none` |
| `WHODB_LOG_FORMAT` | `text` | Log format: `text` or `json` |
| `WHODB_LOG_FILE` | unset | Redirect all non-HTTP logs to a file. `default` uses `/var/log/whodb/whodb.log` |
| `WHODB_ACCESS_LOG_FILE` | unset | Redirect HTTP access logs to a file. `default` uses `/var/log/whodb/whodb.access.log` |
| `WHODB_TOKENS` | unset | Comma-separated static tokens to restrict API/UI access |
| `WHODB_ALLOWED_ORIGINS` | unset | Comma-separated CORS origins (defaults to all) |
| `WHODB_DISABLE_CREDENTIAL_FORM` | `false` | Set `true` to hide the database credential form on the login page |
| `WHODB_MAX_PAGE_SIZE` | `10000` | Maximum number of rows returned per page |
| `WHODB_DISABLE_MOCK_DATA_GENERATION` | unset | Disable mock data generation. `*` disables for all tables, or a comma-separated list of table names to disable selectively (e.g., `logs, metrics`) |

## Cloud Provider Variables

| Variable | Default | Description |
|---|---|---|
| `WHODB_ENABLE_AWS_PROVIDER` | `false` | Set `true` to enable AWS provider |
| `WHODB_AWS_PROVIDER` | unset | JSON array of AWS provider configs (see `aws-integration.md`) |
