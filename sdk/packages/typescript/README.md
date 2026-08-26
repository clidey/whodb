# @clidey/whodb

Official TypeScript/JavaScript SDK for the [WhoDB](https://whodb.com) hosted
platform — your ontology, datasets, and sources as in-code function APIs.

## Install

```bash
npm install @clidey/whodb
```

Requires Node.js ≥ 20.

## Quickstart

```ts
import { WhoDB } from '@clidey/whodb';

// Production: API key (create one in org settings → API keys).
// The key carries its org; the project auto-resolves when the key
// has access to exactly one project.
const whodb = new WhoDB({ apiKey: process.env.WHODB_API_KEY });

// Local development: zero config — reuses your `whodb login` session.
// const whodb = new WhoDB();

const users = whodb.ontology('User');

const user = await users.get('u_123');
const active = await users.list({ where: { status: { eq: 'active' } }, pageSize: 100 });
await users.create({ email: 'a@b.co' });
await users.createMany(rows, { idempotencyKey: 'import-42' });
await users.update('u_123', { plan: 'pro' });
const orders = await users.followLink('u_123', 'orders');

// Iterate everything, page by page:
for await (const page of users.list({ pageSize: 500 }).pages()) {
  console.log(page.rows.length);
}

// Datasets and sources:
const kpis = await whodb.dataset('weekly_kpis').rows();
const raw = await whodb.source('src_...').rows({ Kind: 'Table', Locator: 'events' });
```

## Authentication

Credential precedence:

1. `new WhoDB({ apiKey })` / `{ token }` / `{ credentials }` constructor options
2. `WHODB_API_KEY` environment variable
3. The `whodb` CLI's stored login (`whodb login`), gcloud-ADC-style — ideal
   for local development

Workspace (`org` / `project`) is optional with an API key; pass `{ project }`
or set `WHODB_PROJECT` when the key has access to more than one project.

## Typed clients

Generate typed entity classes from your project's ontology:

```bash
whodb sdk generate --language ts --out src/whodb-gen/
```

```ts
import { createClient } from './whodb-gen/whodb.generated.js';

const { ontology } = createClient({ apiKey: process.env.WHODB_API_KEY });
const user = await ontology.User.get('u_123'); // fully typed
```

## Versioning

Each SDK release is generated against one WhoDB platform release. Deprecated
operations warn once per process with their removal date; operations removed
from the platform raise `WhoDBVersionError` — upgrade the package to resolve.

## License

Apache-2.0
