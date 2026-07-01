#!/usr/bin/env python3
"""
Diffs core/go.mod between two git SHAs and outputs which E2E matrix databases
are affected by the changed Go module paths.

Reads BASE_SHA, HEAD_SHA, and GITHUB_OUTPUT from the environment.
Writes 'skip' and 'databases' to GITHUB_OUTPUT.
"""
import subprocess, os, re, sys

base_sha = os.environ["BASE_SHA"]
head_sha = os.environ["HEAD_SHA"]

result = subprocess.run(
    ["git", "diff", f"{base_sha}...{head_sha}", "--", "core/go.mod"],
    capture_output=True, text=True, check=True,
)

changed = set()
for line in result.stdout.splitlines():
    if not line.startswith("+") or line.startswith("+++"):
        continue
    m = re.match(r"^\+\s+([\w.\-/]+(?:/v\d+)?)\s+v[\d.]+", line)
    if m:
        changed.add(m.group(1))

# module prefix -> comma-separated E2E matrix database names, or "all"
# gorm.io/gorm is listed first: a bump to the shared ORM runs the full suite
MAPPING = [
    ("gorm.io/gorm",                            "all"),
    ("github.com/jackc/pgx",                    "postgres,cockroachdb,yugabytedb,questdb"),
    ("gorm.io/driver/postgres",                 "postgres,cockroachdb,yugabytedb,questdb"),
    ("github.com/go-sql-driver/mysql",          "mysql,mysql8,mariadb,tidb"),
    ("gorm.io/driver/mysql",                    "mysql,mysql8,mariadb,tidb"),
    ("gorm.io/driver/sqlite",                   "sqlite"),
    ("github.com/duckdb/duckdb-go",             "duckdb"),
    ("go.mongodb.org/mongo-driver",             "mongodb,ferretdb"),
    ("github.com/go-redis/redis",               "redis,valkey,dragonfly"),
    ("github.com/elastic/go-elasticsearch",     "elasticsearch,opensearch"),
    ("github.com/elastic/elastic-transport-go", "elasticsearch,opensearch"),
    ("github.com/ClickHouse/clickhouse-go",     "clickhouse"),
    ("gorm.io/driver/clickhouse",               "clickhouse"),
]

databases: set[str] = set()
run_all = False
for pkg in changed:
    for prefix, dbs in MAPPING:
        if pkg.startswith(prefix):
            if dbs == "all":
                run_all = True
            else:
                databases.update(d.strip() for d in dbs.split(","))

github_output = os.environ.get("GITHUB_OUTPUT", "")
if not github_output:
    print("GITHUB_OUTPUT not set", file=sys.stderr)
    sys.exit(1)

with open(github_output, "a") as f:
    if run_all:
        f.write("skip=false\n")
        f.write("databases=all\n")
    elif databases:
        f.write("skip=false\n")
        f.write(f"databases={','.join(sorted(databases))}\n")
    else:
        f.write("skip=true\n")
        f.write("databases=\n")
