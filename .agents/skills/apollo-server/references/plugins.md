# Plugins Reference

## Table of Contents

- [Plugin Structure](#plugin-structure)
- [Server Lifecycle Hooks](#server-lifecycle-hooks)
- [Request Lifecycle Hooks](#request-lifecycle-hooks)
- [Common Plugin Patterns](#common-plugin-patterns)
- [Built-in Plugins](#built-in-plugins)

## Plugin Structure

Plugins are objects that implement lifecycle hooks:

```typescript
import { ApolloServerPlugin } from "@apollo/server";

const myPlugin: ApolloServerPlugin<MyContext> = {
  // Server lifecycle
  async serverWillStart(service) {
    console.log("Server starting");
    return {
      async drainServer() {
        console.log("Server draining");
      },
      async serverWillStop() {
        console.log("Server stopping");
      },
    };
  },

  // Request lifecycle
  async requestDidStart(requestContext) {
    console.log("Request started");
    return {
      async parsingDidStart() {
        return async (err) => {
          if (err) console.log("Parsing error:", err);
        };
      },
      async validationDidStart() {
        return async (errs) => {
          if (errs) console.log("Validation errors:", errs);
        };
      },
      async didResolveOperation(requestContext) {
        console.log("Operation:", requestContext.operationName);
      },
      async executionDidStart() {
        return {
          willResolveField({ info }) {
            const start = Date.now();
            return () => {
              console.log(`${info.fieldName}: ${Date.now() - start}ms`);
            };
          },
        };
      },
      async willSendResponse(requestContext) {
        console.log("Sending response");
      },
    };
  },
};

const server = new ApolloServer({
  typeDefs,
  resolvers,
  plugins: [myPlugin],
});
```

## Server Lifecycle Hooks

### serverWillStart

Called when `server.start()` is invoked, before the server begins accepting requests.

```typescript
const plugin: ApolloServerPlugin = {
  async serverWillStart(service) {
    // service.schema - the GraphQL schema
    // service.apollo - Apollo config (if using Apollo Studio)

    console.log("Schema types:", Object.keys(service.schema.getTypeMap()));

    // Return object with cleanup hooks
    return {
      async drainServer() {
        // Called when server.stop() begins
        // Use to stop accepting new requests
        await closeConnections();
      },

      async serverWillStop() {
        // Called after drainServer, before server fully stops
        // Use for final cleanup
        await db.disconnect();
      },

      schemaDidLoadOrUpdate(schemaContext) {
        // Called when schema changes (e.g., with gateway)
        console.log("Schema updated");
      },
    };
  },
};
```

### drainServer Pattern

```typescript
import { ApolloServerPluginDrainHttpServer } from "@apollo/server/plugin/drainHttpServer";
import http from "http";

const httpServer = http.createServer(app);

const server = new ApolloServer({
  typeDefs,
  resolvers,
  plugins: [
    ApolloServerPluginDrainHttpServer({ httpServer }),
    // Custom drain plugin
    {
      async serverWillStart() {
        return {
          async drainServer() {
            // Stop background jobs
            await scheduler.stop();
            // Close database connections
            await db.end();
          },
        };
      },
    },
  ],
});
```

## Request Lifecycle Hooks

Request lifecycle flows in this order:

```
requestDidStart
  └─> didResolveSource
  └─> parsingDidStart / parsingDidEnd
  └─> validationDidStart / validationDidEnd
  └─> didResolveOperation
  └─> responseForOperation (can short-circuit)
  └─> executionDidStart
        └─> willResolveField (per field)
  └─> didEncounterErrors (if errors)
  └─> willSendResponse
```

### Full Request Lifecycle

```typescript
const plugin: ApolloServerPlugin<MyContext> = {
  async requestDidStart(requestContext) {
    const start = Date.now();
    const { request, contextValue } = requestContext;

    console.log("Query:", request.query);
    console.log("Variables:", request.variables);

    return {
      async didResolveSource(requestContext) {
        // Source (query string) has been resolved
      },

      async parsingDidStart(requestContext) {
        // Return end hook for when parsing completes
        return async (err) => {
          if (err) {
            console.error("Parse error:", err);
          }
        };
      },

      async validationDidStart(requestContext) {
        return async (errs) => {
          if (errs?.length) {
            console.error("Validation errors:", errs);
          }
        };
      },

      async didResolveOperation(requestContext) {
        // Operation name and type are now available
        console.log("Operation:", requestContext.operationName);
        console.log("Type:", requestContext.operation?.operation);
      },

      async responseForOperation(requestContext) {
        // Return response to skip execution (e.g., cached response)
        // Return null to continue normal execution
        return null;
      },

      async executionDidStart(requestContext) {
        return {
          willResolveField({ source, args, contextValue, info }) {
            const fieldStart = Date.now();
            return (error, result) => {
              const duration = Date.now() - fieldStart;
              if (duration > 100) {
                console.log(`Slow field ${info.fieldName}: ${duration}ms`);
              }
            };
          },
        };
      },

      async didEncounterErrors(requestContext) {
        for (const error of requestContext.errors) {
          console.error("GraphQL Error:", error);
        }
      },

      async willSendResponse(requestContext) {
        console.log(`Request completed in ${Date.now() - start}ms`);
      },
    };
  },
};
```

## Common Plugin Patterns

### Logging Plugin

```typescript
const loggingPlugin: ApolloServerPlugin<MyContext> = {
  async requestDidStart({ request, contextValue }) {
    const start = Date.now();
    const requestId = crypto.randomUUID();

    console.log(
      JSON.stringify({
        type: "request_start",
        requestId,
        operationName: request.operationName,
        userId: contextValue.user?.id,
      }),
    );

    return {
      async willSendResponse({ response }) {
        console.log(
          JSON.stringify({
            type: "request_end",
            requestId,
            duration: Date.now() - start,
            errors: response.body.kind === "single" ? (response.body.singleResult.errors?.length ?? 0) : 0,
          }),
        );
      },
    };
  },
};
```

### Timing Plugin

```typescript
const timingPlugin: ApolloServerPlugin = {
  async requestDidStart() {
    const fieldTimes: Map<string, number[]> = new Map();

    return {
      async executionDidStart() {
        return {
          willResolveField({ info }) {
            const start = process.hrtime.bigint();
            return () => {
              const duration = Number(process.hrtime.bigint() - start) / 1e6;
              const key = `${info.parentType.name}.${info.fieldName}`;
              const times = fieldTimes.get(key) ?? [];
              times.push(duration);
              fieldTimes.set(key, times);
            };
          },
        };
      },

      async willSendResponse({ response }) {
        if (response.body.kind === "single") {
          response.body.singleResult.extensions = {
            ...response.body.singleResult.extensions,
            timing: Object.fromEntries(fieldTimes),
          };
        }
      },
    };
  },
};
```

### Error Tracking Plugin

```typescript
const errorTrackingPlugin: ApolloServerPlugin<MyContext> = {
  async requestDidStart({ request, contextValue }) {
    return {
      async didEncounterErrors({ errors, request }) {
        for (const error of errors) {
          // Skip client errors
          if (error.extensions?.code === "BAD_USER_INPUT") continue;

          // Report to error tracking service
          await errorTracker.captureException(error.originalError ?? error, {
            extra: {
              operationName: request.operationName,
              query: request.query,
              variables: request.variables,
              userId: contextValue.user?.id,
            },
          });
        }
      },
    };
  },
};
```

### Caching Plugin

```typescript
const cachingPlugin: ApolloServerPlugin = {
  async requestDidStart({ request }) {
    const cacheKey = createHash("sha256")
      .update(request.query ?? "")
      .update(JSON.stringify(request.variables ?? {}))
      .digest("hex");

    return {
      async responseForOperation() {
        const cached = await cache.get(cacheKey);
        if (cached) {
          return JSON.parse(cached);
        }
        return null;
      },

      async willSendResponse({ response }) {
        if (response.body.kind === "single" && !response.body.singleResult.errors) {
          await cache.set(cacheKey, JSON.stringify(response.body.singleResult), {
            ttl: 300,
          });
        }
      },
    };
  },
};
```

## Built-in Plugins

### ApolloServerPluginDrainHttpServer

Gracefully shuts down HTTP server during `server.stop()`:

```typescript
import { ApolloServerPluginDrainHttpServer } from "@apollo/server/plugin/drainHttpServer";

const server = new ApolloServer({
  typeDefs,
  resolvers,
  plugins: [ApolloServerPluginDrainHttpServer({ httpServer })],
});
```

### ApolloServerPluginLandingPageLocalDefault

Shows Apollo Sandbox for local development:

```typescript
import { ApolloServerPluginLandingPageLocalDefault } from "@apollo/server/plugin/landingPage/default";

const server = new ApolloServer({
  typeDefs,
  resolvers,
  plugins: [ApolloServerPluginLandingPageLocalDefault({ embed: true })],
});
```

### ApolloServerPluginLandingPageProductionDefault

Shows production landing page:

```typescript
import { ApolloServerPluginLandingPageProductionDefault } from "@apollo/server/plugin/landingPage/default";

const server = new ApolloServer({
  typeDefs,
  resolvers,
  plugins: [
    process.env.NODE_ENV === "production"
      ? ApolloServerPluginLandingPageProductionDefault()
      : ApolloServerPluginLandingPageLocalDefault({ embed: true }),
  ],
});
```

### ApolloServerPluginUsageReporting

Reports metrics to Apollo Studio:

```typescript
import { ApolloServerPluginUsageReporting } from "@apollo/server/plugin/usageReporting";

const server = new ApolloServer({
  typeDefs,
  resolvers,
  plugins: [
    ApolloServerPluginUsageReporting({
      sendVariableValues: { all: true },
      sendHeaders: { all: true },
    }),
  ],
});
```

### ApolloServerPluginInlineTrace

Includes trace data in responses (for federated graphs):

```typescript
import { ApolloServerPluginInlineTrace } from "@apollo/server/plugin/inlineTrace";

const server = new ApolloServer({
  typeDefs,
  resolvers,
  plugins: [ApolloServerPluginInlineTrace()],
});
```
