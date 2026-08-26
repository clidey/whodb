# whodb-sdk

Official Python SDK for the [WhoDB](https://whodb.com) hosted platform — your
ontology, datasets, and sources as in-code function APIs.

## Install

```bash
pip install whodb-sdk
```

Requires Python ≥ 3.10.

## Quickstart

```python
import os
from whodb import WhoDB

# Production: API key (create one in org settings → API keys).
whodb = WhoDB(api_key=os.environ["WHODB_API_KEY"])

# Local development: zero config — reuses your `whodb login` session.
# whodb = WhoDB()

users = whodb.ontology("User")

user = users.get("u_123")
active = users.list(where={"status": {"eq": "active"}}, page_size=100).all()
users.create({"email": "a@b.co"})
users.create_many(rows, idempotency_key="import-42")
users.update("u_123", {"plan": "pro"})
orders = users.follow_link("u_123", "orders").all()

# Iterate everything, page by page:
for page in users.list(page_size=500).pages():
    print(len(page.rows))

# Async client:
from whodb import AsyncWhoDB
awhodb = AsyncWhoDB(api_key=os.environ["WHODB_API_KEY"])
user = await awhodb.ontology("User").get("u_123")
```

## Authentication

Credential precedence: constructor args (`api_key=` / `token=` /
`credentials=`) → `WHODB_API_KEY` env var → the `whodb` CLI's stored login.
Workspace (`org=` / `project=`) is optional with an API key; pass `project=`
when the key has access to more than one project.

Inside a WhoDB Function, `WhoDB()` auto-detects the runtime and needs no
configuration at all.

## Typed clients

```bash
whodb sdk generate --language python --out whodb_gen/
```

## License

Apache-2.0
