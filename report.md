  ---
WhoDB Tech Debt & Code Quality Audit — Master Report

Scope: CE + EE backend, frontend, desktop, GraphQL (126K lines)
Excluded: CLI, GitHub Actions, dev scripts, e2e tests

  ---
CRITICAL (Fix immediately — crashes, security, data corruption)

1. Nil pointer dereference in Login resolver — server crash

core/graph/schema.resolvers.go:67, 163
if !src.MainEngine.Choose(engine.DatabaseType(credentials.Type)).IsAvailable(...)
Choose() returns nil for unregistered types. Calling .IsAvailable() on nil panics. The Health resolver at line 1073 does check for nil — this is inconsistent.

2. SQL injection in NullifyFKColumn — column name concatenated raw

core/src/plugins/gorm/constraints.go:116
result := db.Table(tableName).Where(column + " IS NOT NULL").Update(column, gorm.Expr("NULL"))
column comes from schema metadata. Should use QuoteIdentifier.

3. ExportData bypasses connection cache — connection leak

core/src/plugins/gorm/export.go:65-66
Calls p.DB(config) directly instead of plugins.WithConnection. Creates a new uncached connection that is never closed, leaking the pool on every full-table export.

4. DynamoDB filter expression injection (EE)

ee/core/src/plugins/dynamodb/dynamodb.go:401-404
where.Atomic.Key and where.Atomic.Operator are interpolated directly into DynamoDB filter expressions without using ExpressionAttributeNames (#-prefixed placeholders).

5. log.Fatalf in non-fatal code path (EE Keeper)

ee/core/src/integrations/keeper/keeper.go:88, 92
GetLoginProfiles calls log.Fatalf on Keeper API failure, killing the entire server. The function signature already returns error. Currently disabled, but a crash bug if re-enabled.

6. GetPluginForContext has no nil check on credentials

core/graph/resolver.go:40-41
Immediately dereferences creds.Type without checking if GetCredentials(ctx) returned nil.

  ---
HIGH (Significant quality/correctness issues)

7. Plugin architecture violations — 9 database-type checks in shared GORM code

gorm/update.go:109,114, gorm/delete.go:93, gorm/graph.go:39, gorm/plugin.go:537,665, gorm/db.go:145, gorm/utils.go:88
Violates project rule #4. ClickHouse, Postgres, and SQLite3 branches should be in per-plugin overrides.

8. Dead code: storageUnitsMap built and never read

core/src/plugins/gorm/graph.go:63-65
Map allocated and populated on every graph query, never used.

9. Connection cache race condition (TOCTOU)

core/src/plugins/common.go:190-213
Between unlock → ping → re-lock, another goroutine can create a duplicate connection. Manual lock/unlock without defer makes reasoning difficult.

10. Settings global variable has no synchronization

core/src/settings/settings.go:31-58
currentSettings is mutated by UpdateSettings without a mutex. Concurrent requests create a data race.

11. Sensitive credentials logged at Debug level

core/graph/resolver.go:41-45
GetPluginForContext logs hostname, advanced key/value pairs (which include SSL certs, passwords) on every API request at Debug level.

12. Sensitive SQL query logged at Error level

core/graph/schema.resolvers.go:1381
Full raw SQL query (which may contain literal PII) logged when RawExecute fails.

13. Frontend: 70 any types across production code

Key offenders: store/index.ts (5), raw-execute.tsx (7), desktop.ts (5), posthog.tsx (4), ee-modules.d.ts (3 blanket any for EE boundary).

14. Frontend: Module-level dynamic import race condition

frontend/src/pages/storage-unit/explore-storage-unit.tsx:89-99, frontend/src/utils/functions.ts:30-73
Fire-and-forget import('@ee/...') assigns to module-level let variables. Code using these before the promise resolves silently falls back to CE behavior with no indication.

15. Frontend: serializableCheck: false in Redux store

frontend/src/store/index.ts:178
Disables all Redux serialization safety. The Date objects in scratchpad history and the manual localStorage corruption cleanup exist because this guard was removed.

16. God components — 5 components over 1,000 lines

┌──────────────────────────┬───────┬───────────────────────────────────────────────────────────────┐
│           File           │ Lines │                             Hooks                             │
├──────────────────────────┼───────┼───────────────────────────────────────────────────────────────┤
│ table.tsx                │ 1,874 │ Type sets, icons, input props, table, graph card — all in one │
├──────────────────────────┼───────┼───────────────────────────────────────────────────────────────┤
│ raw-execute.tsx          │ 1,188 │ 25+ useState, 6+ useEffect                                    │
├──────────────────────────┼───────┼───────────────────────────────────────────────────────────────┤
│ login.tsx                │ 1,146 │ Credential gen, form state, profiles, auto-login              │
├──────────────────────────┼───────┼───────────────────────────────────────────────────────────────┤
│ chat.tsx                 │ 1,139 │ SSE streaming, title gen, suggestions, SQL exec, charts       │
├──────────────────────────┼───────┼───────────────────────────────────────────────────────────────┤
│ explore-storage-unit.tsx │ 1,078 │ Pagination, filtering, sorting, where conditions, export      │
└──────────────────────────┴───────┴───────────────────────────────────────────────────────────────┘

17. eslint-disable / @ts-ignore proliferation — 20+ instances

6 eslint-disable-next-line react-hooks/exhaustive-deps (stale closure risk), 8 @ts-ignore hiding real type errors, 1 new Function() eval in EE dev-console.

18. Console logging in production — 75 calls

console.log/warn/error throughout production frontend code. Should use structured logger or be removed.

19. EE: DynamoDB creates a new connection on every operation

ee/core/src/plugins/dynamodb/db.go
Every method calls DB(config) which calls ListTables for connectivity validation. No connection caching unlike all GORM-based plugins.

20. EE: components/index.ts exports non-existent ./theme module

ee/frontend/src/components/index.ts:18
export * from './theme' — no such file exists. Build-time error.

21. EE: SVG attributes use invalid hyphenated JSX names

ee/frontend/src/heroicons.tsx:41,81,98,107,112,114
fill-rule and clip-rule instead of fillRule and clipRule. Silently dropped by React.

22. Desktop: wails.ee.json committed in CE directory

desktop-ce/wails.ee.json
Contains EE branding strings. Violates CE/EE separation rule.

23. errgroup.WithContext context discarded

core/graph/schema.resolvers.go:1242, 1330
g, _ := errgroup.WithContext(ctx)
Derived context assigned to _. Goroutine errors cannot cancel siblings.

24. Auth allow-list uses fragile string matching

core/src/auth/auth.go:231-244
Operation names hardcoded as strings ("Login", "GetProfiles"). Any GraphQL rename silently breaks the allowlist.

  ---
MEDIUM (Quality, consistency, maintainability)

25. GraphQL schema uses String! where enums should exist

LoginCredentials.Type (should be DatabaseType), AIChatMessage.Type, MockDataGenerationInput.Method, HealthStatus.Server/Database, DatabaseQuerySuggestion.category, LocalAWSProfile.Source/AuthType.

26. Three incompatible error handling patterns in GraphQL

Pattern A: return nil, err (GraphQL errors). Pattern B: in-band Status: false with error key. Pattern C: partial results + error. AnalyzeMockDataDependencies returns nil error with populated Error field — callers see "success" with embedded error.

27. Record/RecordInput overloaded in 9 different contexts

Attributes, aliasMap, credentials, add/delete/update values — all use the same generic key-value bag type with recursive Extra field.

28. Resolver monolith — schema.resolvers.go is 1,900 lines

No service layer. Resolver talks directly to plugins, auth, settings, env, analytics, and LLM packages.

29. Localization violations — hardcoded English strings

- error-boundary.tsx:54-70 — "Whoops, something went wrong", "Go home"
- chat.tsx:115-116,841,1034-1088 — "Copied!", "Copy", "Query cancelled", 6 fallback strings with t('key') || 'Fallback'
- explore-storage-unit.tsx:821 — "Custom"
- sql-agent.tsx (EE) — "Thinking", "Execute SQL query?", "Loading suggestions..."

30. Duplicated code patterns

- log.WithError(err).Error(fmt.Sprintf(...)) — 35+ locations should use structured fields
- InsertRow/UpdateQuery/DeleteQuery/CountQuery in SQLBuilder — 4 copies of same 3-line table name construction (line 189-252)
- PRAGMA foreign_key_list in SQLite — identical at lines 204-205 and 647-648
- scrollToBottom in chat.tsx — same setTimeout pattern 5 times
- Chart data-mapping logic duplicated between line-chart.tsx and pie-chart.tsx (EE)
- getProfileLabel duplicated in sidebar and health-overlays
- isMutation check duplicated in chat_baml.go and http_ai_stream.go (EE)
- Operation type conversion duplicated in same two EE files

31. QuoteIdentifier silent fallback

core/src/plugins/gorm/sqlbuilder.go:82
If no dialect available, returns identifier unquoted: return identifier. Should error.

32. RegistryPlugin is a misspelling of RegisterPlugin

core/src/engine/engine.go:53
Used throughout codebase. Cosmetic but confusing.

33. "Houdini" naming in chat store

frontend/src/store/chat.ts:49
Redux slice named houdiniSlice, exports HoudiniActions. Leftover codename.

34. EE: Fallback to openai-generic for unknown provider types

ee/core/src/common/chat_baml.go:176-190
Silent fallback violates "no fallback logic unless asked" rule.

35. EE: Missing build tags on most plugin files

MSSQL, Oracle, DynamoDB plugin files rely on module-level isolation rather than explicit //go:build ee tags. Brittle if imported from different entry point.

36. EE: DynamoDB.AddStorageUnit creates invalid key schema

ee/core/src/plugins/dynamodb/add.go:149-175
Adds all non-PK fields to keySchema. DynamoDB key schema only allows partition + optional sort key.

37. EE: Oracle reports size as "MB" but queries KB

ee/core/src/plugins/oracle/oracle.go:117-122
SQL divides by 1024 (bytes → KB), display says "%.2f MB".

38. Frontend: Password stored in plaintext in localStorage

frontend/src/config/graphql-client.ts:156-164, frontend/src/store/auth.ts
Auto-login reads currentProfile.Password from Redux/localStorage.

39. serverStarted channel never receives — startup is purely time-based

core/server.go:89-142
Nothing sends on the channel. Server always falls through to 2-second timeout.

40. stopCleanup channel panics if CloseAllConnections called twice

core/src/plugins/common.go:52-57, 160-166

41. Desktop: useDesktopMenu useEffect missing dependencies

frontend/src/hooks/useDesktop.ts:214
Closes over navigate, location, currentAuth but only lists [isDesktop].

42. Desktop: Exported files written world-readable (0644)

desktop-common/app.go:38,143,167
Database exports should use 0600 for sensitive data.

43. get-chat.graphql missing RequiresConfirmation field

Frontend can't show confirmation prompt for SQL execution.

44. EE: Three GraphQL queries are no-op stubs returning nil, nil

ee/core/graph/resolver.go:157-167
CodeCompletion, ExplainQuery, GenerateQuery — dead API surface.

45. EE: ExportData scans DynamoDB table twice

ee/core/src/plugins/dynamodb/export.go:52-155
First pass discovers column names, second pass writes rows. Doubles RCU consumption.

  ---
LOW (Cleanup, naming, minor inconsistencies)

- interface{} → any inconsistency in DynamoDB plugin files (EE)
- IntPtr helper in engine package (belongs in util)
- ValidateColumnType does no actual param validation
- sanitizeID operates on bytes not runes — breaks with Unicode
- readRequestBody uses strings.Builder instead of bytes.Buffer
- DatabaseMetadata.AliasMap iteration order non-deterministic
- InitPlugin() called lazily in AddRow instead of at construction
- Copyright year inconsistencies (2025 vs 2026)
- go 1.25.4 in desktop-common go.mod (invalid Go version)
- DynamicExport in table.tsx is dead abstraction (EEExport hardcoded to null)
- 4 desktop hooks are trivial pass-throughs adding no value
- chooseRandomItems throws if fewer items than requested
- position variable in SQL parser declared but never used
- addRowTimeoutRef cleanup effect references a timeout that's never set
- Missing doc comments on 30+ exported Go functions
- Various TODO comments marking known but unaddressed edge cases

  ---
Recommended Prioritization

┌────────────────────┬────────────────────────────────────────────────────────────────────────────────────────────────────────┬───────────┐
│      Priority      │                                                 Items                                                  │  Effort   │
├────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────┼───────────┤
│ P0 — Ship-blockers │ #1 nil deref, #2 SQL injection, #3 connection leak, #6 nil check                                       │ 1-2 hours │
├────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────┼───────────┤
│ P1 — Security      │ #4 DynamoDB injection, #11-12 credential logging, #38 plaintext password, #31 QuoteIdentifier fallback │ Half day  │
├────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────┼───────────┤
│ P2 — Architecture  │ #7 plugin violations (9 spots), #8 dead code, #28 resolver monolith, #16 god components                │ 2-3 days  │
├────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────┼───────────┤
│ P3 — Type safety   │ #13 any types, #15 Redux serializable, #17 ts-ignore, #25 string-typed enums                           │ 1-2 days  │
├────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────┼───────────┤
│ P4 — Consistency   │ #29 localization, #30 duplicated code, #32-33 naming, #18 console logs                                 │ 1-2 days  │
├────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────┼───────────┤
│ P5 — EE-specific   │ #19 DynamoDB caching, #20 broken export, #21 SVG attrs, #34-37                                         │ 1 day     │
└────────────────────┴────────────────────────────────────────────────────────────────────────────────────────────────────────┴───────────┘

Want me to start with the P0 items?