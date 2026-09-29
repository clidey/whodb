# Protected SQL execution

Automatic SQL execution uses the same classifier in CLI/MCP and chat. A query
must fit the supported read grammar **and** have a connector with enforced
read-only execution. Unknown syntax requires approval; read-only mode rejects it.
Unsupported connectors return an error instead of executing normally. Explicitly
writable query/editor flows keep their existing permissions.

## Supported grammar and compatibility

The shared classifier accepts a deliberately bounded SQL subset: SELECT,
read-only CTEs and subqueries, explicit `AS` aliases, joins, ordinary comparisons,
grouping, ordering, limits, set operations, VALUES, TABLE, simple EXPLAIN forms,
a few standalone SHOW forms, and SQLite metadata PRAGMAs. Standard unqualified
COUNT/SUM/AVG/MIN/MAX aggregates are supported. Every statement in a batch must
pass. The parser is not intended to validate every dialect's complete grammar.
The database can still reject a syntactically unsupported read.

Dialect extensions, implicit aliases, casts, most function calls, qualified or
quoted routine calls, table functions, backslash-dependent quoting, executable
comments and unrecognized syntax require approval. So do SELECT INTO,
OUTFILE/DUMPFILE, sequence NEXTVAL, explicit locking, configuration commands,
procedural execution, DDL/DML, and native commands outside the SQL grammar.
Adding syntax support requires positive and adversarial regression tests; adding
a leading verb or keyword exception is insufficient.

The contract assumes trusted database schema objects, built-in routine
resolution, and appropriate database credentials. It does not sandbox a hostile
database server, administrator-installed routines/operators/views, or external
storage hidden behind a table. Use least-privilege database accounts and trusted
schema/search-path configuration. It does not promise zero engine-internal
writes (such as caches or temporary query processing), resource isolation, or
that MCP confirmation proves a human clicked a button.

## Enforcement

PostgreSQL-family and MySQL-family connectors use native read-only transactions.
SQLite uses a read-only file connection plus query_only. ClickHouse uses
readonly=1. Protected GORM requests own their connections, close them after the
request, and cannot join an existing transaction; ordinary connection pooling is
unchanged. This costs an extra connection establishment for protected requests.

DuckDB protection is disabled because the current driver rejects read-only
transactions. Sources without a verified enforcement implementation also reject
protected execution. A source catalog capability is a requirement for execution,
not permission to silently drop the read-only flag.

MCP default confirmation mode uses protected execution even for queries that do
not need approval. Approval retains SQL, parameters and a fingerprint of the
resolved connection. Changed targets require new approval. Read-only and current
security restrictions still apply. Approval is consumed before dispatch: after
an error or timeout the caller must inspect the outcome and obtain new approval
before another attempt. This provides at-most-one dispatch per token, not
exactly-once database effects.

## Local security verification

These commands start **disposable localhost databases**, with dedicated names
and ports. Do not substitute production connections. They do not use the shared
development database stack.

```sh
docker run -d --name whodb-readguard-pg -p 127.0.0.1:55439:5432 -e POSTGRES_USER=guard -e POSTGRES_PASSWORD=guard-local -e POSTGRES_DB=guard postgres:18
docker run -d --name whodb-readguard-mysql -p 127.0.0.1:53319:3306 -e MYSQL_ROOT_PASSWORD=guard-local -e MYSQL_DATABASE=guard mysql:8.4 --secure-file-priv=/tmp
docker run -d --name whodb-readguard-maria -p 127.0.0.1:53329:3306 -e MARIADB_ROOT_PASSWORD=guard-local -e MARIADB_DATABASE=guard mariadb:latest --secure-file-priv=/tmp
docker run -d --name whodb-readguard-clickhouse -p 127.0.0.1:59009:9000 -e CLICKHOUSE_USER=guard -e CLICKHOUSE_PASSWORD=guard-local -e CLICKHOUSE_DB=guard -e CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT=1 clickhouse/clickhouse-server:25.5
```

Wait for the database health checks to succeed, then run sequentially:

```sh
# From core/
go test ./src/sqlguard ./src/plugins/... ./src/source/adapters ./src/bamlconfig ./graph
WHODB_READGUARD_LOCAL=1 go test -tags integration ./src/plugins/clickhouse -run '^TestClickHouseProtectedExecution$' -count=1
go test ./src/sqlguard -run '^$' -fuzz '^FuzzProtectedRead$' -fuzztime=15s -parallel=2

# From cli/
go test ./pkg/mcp ./internal/database
go build -o /tmp/whodb-readguard-cli .
WHODB_READGUARD_LOCAL=1 WHODB_READGUARD_BINARY=/tmp/whodb-readguard-cli go test -tags integration ./pkg/mcp -run '^TestReadGuardTransport$' -count=1
```

Without WHODB_READGUARD_BINARY the transport suite runs the same MCP server
inside the test process over real HTTP. With it, it launches the specified
compiled CLI. Database observers verify no table/file creation before approval,
positive effects after approval, parameter preservation and replay rejection.
Native transaction tests bypass the classifier to check database write refusal.
SQLite tests independently disable query_only on a protected connection and
verify that file access mode still rejects writes. ClickHouse tests independently
exercise its native readonly setting. These tests fail when their required
local database is missing.

```sh
docker stop whodb-readguard-pg whodb-readguard-mysql whodb-readguard-maria whodb-readguard-clickhouse
docker rm -v whodb-readguard-pg whodb-readguard-mysql whodb-readguard-maria whodb-readguard-clickhouse
rm /tmp/whodb-readguard-cli
```

The live matrix above does not certify every PostgreSQL/MySQL-compatible product
or provider version. Those need independent connector integration evidence.

## Verification snapshot (2026-09-28)

The live tests used PostgreSQL 18.6, MySQL 8.4.10, MariaDB 13.0.2 and ClickHouse
25.5.11.15. The transport suite passed with the CE binary against all three
transactional engines in read-only, default-confirmation and strict-confirmation
modes. Both MySQL file-output forms were exercised with root/FILE privileges so
the negative result was not merely a missing-permission error. Their approved
positive controls created actual files in the disposable containers.

The focused Go suites, backend/CLI builds, and lint checks passed. The SQL fuzz
run completed 1,817,694 executions in 16 seconds without a failing input. These
are local results, not a patched release or production/hosted verification.
