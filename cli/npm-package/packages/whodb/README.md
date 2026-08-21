# whodb

The [WhoDB](https://whodb.com) CLI — an interactive, production-ready command-line
interface for navigating SQL and NoSQL databases, with a TUI, programmatic
commands, and an MCP server for AI assistants.

## Use `@clidey/whodb` instead

This unscoped package is a thin wrapper kept published so the name isn't
squatted by someone else. The primary, namespaced package is
[`@clidey/whodb`](https://www.npmjs.com/package/@clidey/whodb):

```bash
npm install -g @clidey/whodb
whodb --help
```

Installing this package also works and provides the same `whodb` command —
it depends on and forwards to
[`@clidey/whodb-cli`](https://www.npmjs.com/package/@clidey/whodb-cli), same
as `@clidey/whodb` does. Prefer the scoped name in new setups.

## Documentation

Full CLI documentation, command reference, and MCP setup:
https://github.com/clidey/whodb/tree/main/cli#readme
