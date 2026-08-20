# whodb

The WhoDB CLI — an interactive database management tool with a TUI, programmatic
commands, and an MCP server.

This package is a thin wrapper around
[`@clidey/whodb-cli`](https://www.npmjs.com/package/@clidey/whodb-cli), which
contains the actual launcher and platform binaries. Installing either package
provides the `whodb` command.

```bash
npm install -g whodb
whodb --help
```

Or without installing:

```bash
npx whodb
```

Documentation: https://github.com/clidey/whodb/tree/main/cli#readme
