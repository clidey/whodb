# whodb-sdk (Java)

Official Java SDK for the [WhoDB](https://whodb.com) hosted platform — your
ontology as in-code function APIs.

## Install

```xml
<dependency>
  <groupId>com.clidey</groupId>
  <artifactId>whodb-sdk</artifactId>
  <version>LATEST</version>
</dependency>
```

Requires Java ≥ 17. One dependency (`jackson-databind`); HTTP via the JDK's
built-in `java.net.http.HttpClient`.

## Quickstart

```java
import com.clidey.whodb.OntologyHandle;
import com.clidey.whodb.WhoDB;
import java.util.List;
import java.util.Map;

// Production: API key (create one in org settings → API keys).
WhoDB client = new WhoDB(new WhoDB.Config(
    System.getenv("WHODB_API_KEY"), null, null, null, null, null, null));

// Local development: zero config — reuses your `whodb login` session.
// WhoDB client = new WhoDB();

OntologyHandle users = client.ontology("User");

Map<String, Object> user = users.get("u_123");            // null when absent
List<Map<String, Object>> rows = users.list(new OntologyHandle.ListOptions(
    Map.of("status", Map.of("eq", "active")), null, 100, 0));
users.create(Map.of("email", "a@b.co"));
users.createMany(rowsToInsert, "import-42");               // idempotency key
users.update("u_123", Map.of("plan", "pro"));
List<Map<String, Object>> orders = users.followLink("u_123", "orders", 50, 0);

// Iterate everything, page by page:
for (List<Map<String, Object>> page :
        users.pages(new OntologyHandle.ListOptions(null, null, 500, 0))) {
    System.out.println(page.size());
}
```

Rows are `Map<String, Object>` with native-typed values; timestamps hydrate
to `java.time.OffsetDateTime`.

## Authentication

Credential precedence: `Config` fields (`apiKey` / `token` / `credentials`)
→ `WHODB_API_KEY` env var → the `whodb` CLI's stored login. Workspace
(`org` / `project`) is optional with an API key; set `project` when the key
has access to more than one project.

Inside a WhoDB Function, `new WhoDB()` auto-detects the runtime and switches
to the in-container IPC transport (unix socket in Docker, TCP in K8s).

## Errors

All failures throw `WhoDBException`; `kind()` identifies the class: `AUTH`,
`NOT_FOUND`, `VALIDATION`, `VERSION` (SDK outdated for the platform API —
upgrade the artifact), `CLI_CREDENTIALS`, `TRANSPORT_CAPABILITY`, `PLATFORM`
(with `code()`), `TRANSPORT`.

## License

Apache-2.0
