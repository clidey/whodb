/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"fmt"

	"github.com/clidey/whodb/cli/pkg/identity"
	"github.com/spf13/cobra"
)

var guideCmd = &cobra.Command{
	Use:   "guide",
	Short: "Show the full usage guide",
	Long:  "Displays a comprehensive guide covering all features, shortcuts, and workflows.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(identity.ReplaceText(guideText))
	},
}

func init() {
	rootCmd.AddCommand(guideCmd)
}

const guideText = `
╔══════════════════════════════════════════════════════════════╗
║                    WhoDB CLI User Guide                      ║
╚══════════════════════════════════════════════════════════════╝

GETTING STARTED
───────────────
  whodb                                  Launch the TUI
  whodb connect --type postgres \        Connect directly
    --host localhost --user alice --database mydb
  whodb connect --type sqlite3 \         SQLite (no credentials)
    --database ./app.db
  whodb connect --docker                 Auto-detect Docker DBs

  Once connected, you'll see the Browser view with your tables.
  Press ? in any view for a quick shortcut reference.
  Reconnectable TUI sessions resume automatically on the next launch.

LAYOUTS
───────
  The TUI supports split-pane layouts like lazygit/Harlequin.

  Ctrl+L      Cycle layouts: Single → Explore → Query → Full
  Tab         Switch focus between panes
  Alt+←/→     Resize split ratio

  Layouts:
    Single   — One view at a time (default for narrow terminals)
    Explore  — Browser | Results side by side
    Query    — Editor / Results stacked
    Full     — Browser | Editor / Results (three panes)

THEMES
──────
  Ctrl+T      Cycle through 8 themes:
              Default, Light, Monokai, Dracula, Nord,
              Gruvbox, Tokyo Night, Catppuccin

  Your theme choice is saved and persists across sessions.

SQL EDITOR
──────────
  The editor has IDE-like features:

  opt/alt+Enter   Execute query
  Ctrl+F          Format/prettify SQL
  Ctrl+X          Run EXPLAIN on current query
  Ctrl+O          Open query in $EDITOR (vim, nano, etc.)
  Ctrl+N          New editor tab
  Ctrl+W          Close current tab
  Shift+←/→       Switch between tabs

  Autocomplete triggers automatically as you type. It's context-
  aware: after FROM it suggests tables, after WHERE column_name
  it suggests operators (=, LIKE, IN, etc.), and columns from
  referenced tables rank highest.

  Ghost text: Start typing and you'll see dimmed suggestions
  from your query history. Press → (right arrow) to accept.

  When the editor is empty, backend-generated suggestions appear
  based on the tables in the active schema or database.

  CLI:
    whodb explain --connection mydb "SELECT * FROM users"
    whodb explain --connection mydb --format json "SELECT * FROM users"
    whodb query --connection mydb --stream --format ndjson "SELECT * FROM users"

BROWSING DATA
─────────────
  In the Browser view:
    ↑↓←→ or hjkl   Navigate tables
    Enter           View table data
    f or /          Filter tables by name
    Ctrl+R          Refresh table list

  In the Results view:
    ↑↓              Scroll rows
    h/l or ←/→      Scroll columns
    s               Cycle page size (10/25/50/100)
    z               View cell content (JSON pretty-print)
    e               Export to CSV/Excel
    w               WHERE condition builder
    c               Column visibility toggle

WHERE CONDITIONS
────────────────
  Press w from Results to open the WHERE builder.

  a       Add a condition (pick field → operator → value)
  g       Create a new condition group
  t       Toggle a group between AND / OR
  d       Delete a condition or group
  Enter   Apply conditions to the query

  Groups allow nested logic:
    WHERE (name = 'alice' AND age > 18) OR (role = 'admin')

IMPORT / EXPORT
───────────────
  Import:
    Ctrl+G                    Open import wizard in TUI
    whodb import \        CLI import
      -c mydb -f data.csv -t users --create-table

    Supports CSV and Excel (.xlsx). Auto-detects delimiter
    and infers column types (INTEGER, REAL, BOOLEAN, TEXT).

  Export:
    Press e from Results, or:
    whodb export -c mydb -t users -o users.csv
    whodb export -c mydb -Q "SELECT * FROM users" -o users.csv --stream

MOCK DATA
─────────
  Generate FK-aware mock data from the CLI:

    whodb mock-data \        Analyze only
      -c mydb -t orders -r 50 --analyze

    whodb mock-data \        Generate after confirmation
      -c mydb -t orders -r 50

    whodb mock-data \        Overwrite existing rows
      -c mydb -t orders -r 50 --overwrite --yes

  The command analyzes parent-table dependencies first, then generates
  data in the correct order. Blocked tables reuse existing rows instead
  of writing new data.

CLOUD DISCOVERY
───────────────
  When cloud provider support is enabled, you can inspect configured
  providers and their discovered resources directly from the CLI, then
  prefill normal connection flows from a discovered resource ID:

    whodb cloud providers list
    whodb cloud providers test aws-prod-us-west-2
    whodb cloud providers refresh --all
    whodb cloud connections list
    whodb cloud connections list --provider aws-prod-us-west-2
    whodb connect --discovered aws-prod-us-west-2/prod-db
    whodb connections add --from-discovered aws-prod-us-west-2/prod-db --user alice --database app

  Cloud provider support follows the shared provider flags:
    WHODB_ENABLE_AWS_PROVIDER=true
    WHODB_ENABLE_AZURE_PROVIDER=true
    WHODB_ENABLE_GCP_PROVIDER=true

SCHEMA DIFF
───────────
  Ctrl+V      Open schema diff in the TUI

  Compare two saved connections, adjust schema overrides if needed,
  then browse the shared diff output in a scrollable view.

  CLI:
    whodb diff --from staging --to prod
    whodb diff --from staging --to prod --schema public
    whodb diff --from staging --to prod --format json

AI CHAT
───────
  Ctrl+A      Open AI chat

  Supports OpenAI, Anthropic, Ollama, and LM Studio.
  Use ←/→ to select provider and model.

  The AI has full schema awareness — it knows your tables and
  columns. Responses stream token-by-token.

  For SELECT queries, results appear inline. For mutations
  (INSERT, UPDATE, DELETE), you'll be asked to confirm before
  execution — press Enter on the message to run it.

  Set API keys via environment variables:
    WHODB_OPENAI_API_KEY=sk-...
    WHODB_ANTHROPIC_API_KEY=sk-ant-...

  Ollama and LM Studio work locally with no API key.

ER DIAGRAM
──────────
  Ctrl+K      Show ER diagram

  Displays all tables with columns, types, primary keys,
  and foreign key relationships using Unicode box drawing.

  Tab         Cycle between tables
  z           Toggle zoom (compact / expanded)
  ↑↓          Scroll

  CLI:
    whodb erd --connection mydb
    whodb erd --connection mydb --format json

BOOKMARKS
─────────
  Ctrl+B      Open bookmarks

  Save frequently used queries with a custom name.
  Press s to save the current editor query, Enter to load
  a saved bookmark, d to delete.

  CLI:
    whodb bookmarks list
    whodb bookmarks save recent-users "SELECT * FROM users ORDER BY id DESC"
    whodb bookmarks load recent-users
    whodb bookmarks delete recent-users

COMMAND LOG
───────────
  Ctrl+D      Toggle command log

  Shows every operation the CLI performs under the hood:
  schema fetches, table loads, your queries, exports, imports —
  all with timestamps, durations, and row counts. Like lazygit's
  command log.

HISTORY
───────
  Ctrl+H      Open query history

  Browse past queries, re-run them, or load into the editor.

SSH TUNNELS
───────────
  For databases behind a bastion/jump host:

  1. In the connection form, toggle "SSH Tunnel" to On
  2. Fill in SSH Host, SSH User, SSH Key File (or SSH Password)
  3. The CLI creates a local tunnel automatically

  Requires the host to be in your ~/.ssh/known_hosts file.

SSL
───
  The CLI also supports SSL mode selection and certificate files
  in both command mode and the TUI connection form.

  Examples:
    whodb connect \
      --type postgres --host localhost --user alice --database mydb \
      --ssl-mode verify-ca --ssl-ca ./ca.pem

    whodb connections add \
      --name prod --type Postgres --host db.internal --user alice --database mydb \
      --ssl-mode verify-identity --ssl-ca ./ca.pem --ssl-server-name db.internal

DOCKER DETECTION
────────────────
  The CLI auto-detects running Docker database containers.
  They appear in the connection list tagged with "(docker)".

  Selecting a Docker connection opens the form pre-filled
  with the detected type, host, and port — just add credentials.

  CLI: whodb connect --docker

CONNECTION PROFILES
───────────────────
  Ctrl+P      Open profiles

  A profile bundles: connection + theme + page size + timeout.

  To save:    Press s, enter a name (e.g., "production")
  To apply:   Select a profile and press Enter
  To delete:  Select and press d

  CLI: whodb --profile production
  CLI management:
    whodb profiles list
    whodb profiles save production --connection prod --theme Dracula --page-size 100 --timeout 30
    whodb profiles show production
    whodb profiles delete production

  Useful for switching between dev/staging/prod with different
  visual cues (e.g., red theme for production).

READ-ONLY MODE
──────────────
  Ctrl+Y      Toggle read-only mode

  When on, all mutation queries (INSERT, UPDATE, DELETE, DROP,
  ALTER, CREATE, TRUNCATE) are blocked. A [READ-ONLY] badge
  appears in the status bar. Persists across sessions.

DATA QUALITY AUDIT
──────────────────
  Ctrl+U      Open audit view (TUI)

  Scans tables and reports:
    - Null rates per column (configurable thresholds)
    - Duplicate rows on unique-looking columns
    - Orphaned foreign key references
    - Low cardinality columns
    - Missing primary keys
    - Type mismatches (e.g. _id column with TEXT type)

  Each issue is color-coded: ✓ green (ok), ⚠ yellow (warning),
  ✗ red (error). Press Enter on an issue to see the actual rows.

  CLI usage:
    whodb audit --type sqlite3 --database ./app.db
    whodb audit --type postgres --host localhost --user alice --database mydb
    whodb audit --connection mydb --table users
    whodb audit --connection mydb --format json

  Custom thresholds:
    whodb audit --connection mydb --null-warning 20 --null-error 70

  Default thresholds: >10% null = warning, >50% null = error,
  <5 distinct values = low cardinality warning.

PROGRAMMATIC USAGE
──────────────────
  The CLI supports non-interactive commands for scripting:

  whodb query "SELECT * FROM users" -c mydb -f json
  whodb schemas -c mydb -f json
  whodb tables -c mydb -s public -f json
  whodb columns -c mydb -t users
  whodb diff --from staging --to prod --format json
  whodb explain --connection mydb --format json "SELECT * FROM users"
  whodb erd --connection mydb --format json
  whodb suggestions --connection mydb --format json
  whodb bookmarks list --format json
  whodb bookmarks load recent-users --format json
  whodb profiles list --format json
  whodb profiles show production --format json
  whodb history list --format json
  whodb connections list
  whodb connections test mydb --format json
  whodb history clear --format json
  whodb mock-data --connection mydb --table orders --rows 10 --analyze --format json

  Query/list commands support structured output with -f json where available.
  Action/analysis commands emit {command, success, data} envelopes.
  Output formats: table, plain, json, ndjson, csv (use -f flag).
  ndjson emits one JSON object per row for streaming-friendly pipes.
  Pipe-friendly: auto-detects TTY and uses plain format when piped.

MCP SERVER
──────────
  whodb mcp serve

  Runs as a Model Context Protocol server for AI assistants
  (Claude Desktop, Claude Code, etc.). Supports stdio and HTTP
  transport with configurable security modes.

CONFIGURATION
─────────────
  Config file: ~/.whodb/config.json

  Stores: connections, saved queries, profiles, history settings,
  display preferences (theme, page size), query timeout, AI
  consent, and read-only mode.

  Passwords are stored in the OS keyring (macOS Keychain,
  Linux Secret Service) when available.

  Environment variables for connections:
    WHODB_POSTGRES='[{"alias":"prod","host":"..."}]'
    WHODB_MYSQL_1='{"alias":"dev","host":"..."}'
`
