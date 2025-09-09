/*
 * Copyright 2025 Clidey, Inc.
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

import {autocompletion, Completion, CompletionContext, CompletionResult} from "@codemirror/autocomplete";
import {ApolloClient} from "@apollo/client";
import {ColumnsDocument, GetSchemaDocument, GetStorageUnitsDocument} from "@graphql";

/**
 * Advanced SQL Autocomplete for CodeMirror
 *
 * Features included:
 * - Schema suggestions after FROM / JOIN / database selector points
 * - Table suggestions for schema.<cursor>
 * - Column suggestions for table.<cursor> and alias.<cursor>
 * - Alias extraction & alias-aware suggestions (alias.col)
 * - Mixed-context suggestions (WHERE / ON) showing aliases, table names and columns from involved tables
 * - SELECT clause suggestions: columns (qualified/unqualified), SQL functions, snippets
 * - Snippets for common SQL patterns (JOIN ON, WHERE IN, CTE templates)
 * - SQL keyword fallback completions
 * - Prefix-aware filtering and safe error handling
 *
 * Notes:
 * - The GraphQL queries are expected to provide arrays: Schema (string[]), StorageUnit (with Name), Columns (with Name, Type)
 * - If your GraphQL schema differs, adapt the `fetch` helpers accordingly.
 */

interface AutocompleteOptions {
    apolloClient: ApolloClient<any>;
}

// SQL keywords to ignore when determining context
const SQL_KEYWORDS = new Set([
    'SELECT', 'FROM', 'WHERE', 'JOIN', 'INNER', 'LEFT', 'RIGHT', 'OUTER',
    'ON', 'AND', 'OR', 'ORDER', 'BY', 'GROUP', 'HAVING', 'LIMIT', 'OFFSET',
    'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'DROP', 'ALTER', 'TABLE', 'INTO',
]);

/* ---------- Helpers to fetch metadata from backend via Apollo ---------- */

async function fetchSchemas(client: ApolloClient<any>): Promise<string[]> {
    try {
        const res = await client.query({query: GetSchemaDocument, fetchPolicy: 'cache-first'});
        // Adapt to your GraphQL response shape
        return res.data?.Schema ?? [];
    } catch (e) {
        console.error('fetchSchemas error', e);
        return [];
    }
}

async function fetchTables(client: ApolloClient<any>, schema: string): Promise<{ Name: string }[]> {
    try {
        const res = await client.query({
            query: GetStorageUnitsDocument,
            variables: {schema},
            fetchPolicy: 'cache-first',
        });
        return res.data?.StorageUnit ?? [];
    } catch (e) {
        console.error('fetchTables error', e);
        return [];
    }
}

async function fetchColumns(client: ApolloClient<any>, schema: string, storageUnit: string): Promise<{
    Name: string,
    Type?: string
}[]> {
    try {
        const res = await client.query({
            query: ColumnsDocument,
            variables: {schema, storageUnit},
            fetchPolicy: 'cache-first',
        });
        return res.data?.Columns ?? [];
    } catch (e) {
        console.error('fetchColumns error', e);
        return [];
    }
}

/* ---------- Utility: extract tables + aliases from a query (robust) ---------- */

function extractTablesAndAliasesFromQuery(text: string): Array<{ schema?: string, table: string, alias?: string }> {
    const tables: Array<{ schema?: string, table: string, alias?: string }> = [];
    // Use case-insensitive matching on original text but capture real-case table names where possible
    // We'll search in the whole text for FROM and JOIN occurrences. This is heuristic but practical.
    const patterns = [
        /FROM\s+(?:(`?([\w$]+)`?)\.)?(?:`?([\w$]+)`?)(?:\s+(?:AS\s+)?(`?([\w$]+)`?))?/gi,
        /JOIN\s+(?:(`?([\w$]+)`?)\.)?(?:`?([\w$]+)`?)(?:\s+(?:AS\s+)?(`?([\w$]+)`?))?/gi
    ];

    for (const pattern of patterns) {
        let match;
        while ((match = pattern.exec(text)) !== null) {
            // match groups vary because of optional groups; unify them
            // If captured with backticks groups indices differ; handle gracefully.
            // We'll prefer group values that are present and not equal to the other capture groups.
            // match example groups can be [full, schemaWithBackticks?, schemaNoBT?, tableNoBT?, aliasWithBT?, aliasNoBT?]
            const groups = match.slice(1).filter(g => g !== undefined);
            // Try to derive schema, table, alias by scanning captured groups for alphabetical strings
            let schema: string | undefined;
            let table: string | undefined;
            let alias: string | undefined;
            // naive reconstruction:
            // if pattern was FROM schema.table alias -> match groups include schema, table, alias at expected positions
            // fallback: try to locate by index heuristics
            if (match[2] && match[3]) { // pattern with separate groups
                schema = match[2];
                table = match[3];
                alias = match[5] || undefined;
            } else if (match[1] && match[3]) {
                schema = match[1].replace(/`/g, '');
                table = match[3].replace(/`/g, '');
                alias = (match[4] || match[5])?.replace(/`/g, '');
            } else {
                // more forgiving fallback: pick first as schema if followed by '.', next as table, last as alias
                const tokens = match[0].split(/\s+/).slice(1); // tokens after FROM/JOIN
                // clean tokens
                const cleaned = tokens.map(t => t.replace(/[,;)]/g, '').replace(/`/g, ''));
                // best-effort
                if (cleaned.length > 0) {
                    const first = cleaned[0];
                    const dotIdx = first.indexOf('.');
                    if (dotIdx > -1) {
                        const s = first.substring(0, dotIdx).replace(/`/g, '');
                        const t = first.substring(dotIdx + 1).replace(/`/g, '');
                        schema = s;
                        table = t;
                        if (cleaned.length > 1) alias = cleaned[1].replace(/`/g, '');
                    } else {
                        table = first.replace(/`/g, '');
                        if (cleaned.length > 1) alias = cleaned[1].replace(/`/g, '');
                    }
                }
            }

            if (table) {
                tables.push({schema: schema || undefined, table, alias: alias || undefined});
            }
        }
    }

    return tables;
}

/* ---------- Build Completion object ---------- */

function createCompletion(label: string, type: string, detail?: string, apply?: string): Completion {
    return {
        label,
        type,
        detail: detail || type,
        apply: apply ?? label,
    };
}

/* ---------- SQL keywords and functions ---------- */

const SQL_KEYWORD_LIST = [
    'SELECT', 'FROM', 'WHERE', 'JOIN', 'INNER', 'LEFT', 'RIGHT', 'OUTER', 'FULL',
    'ON', 'AND', 'OR', 'NOT', 'NULL', 'TRUE', 'FALSE', 'ORDER', 'BY', 'GROUP',
    'HAVING', 'LIMIT', 'OFFSET', 'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'DROP',
    'ALTER', 'TABLE', 'INDEX', 'VIEW', 'DATABASE', 'SCHEMA', 'INTO', 'VALUES',
    'SET', 'AS', 'DISTINCT', 'COUNT', 'SUM', 'AVG', 'MAX', 'MIN', 'CASE', 'WHEN',
    'THEN', 'ELSE', 'END', 'IF', 'EXISTS', 'LIKE', 'IN', 'BETWEEN', 'IS', 'ASC',
    'DESC', 'UNION', 'ALL', 'ANY', 'SOME', 'WITH', 'RECURSIVE', 'CASCADE',
    'CONSTRAINT', 'PRIMARY', 'KEY', 'FOREIGN', 'REFERENCES', 'UNIQUE', 'CHECK',
    'DEFAULT', 'TRUNCATE', 'EXPLAIN', 'ANALYZE', 'GRANT', 'REVOKE', 'COMMIT',
    'ROLLBACK', 'TRANSACTION', 'BEGIN', 'START'
];

const SQL_FUNCTIONS = [
    {name: 'COUNT', detail: 'COUNT(expr)'},
    {name: 'SUM', detail: 'SUM(expr)'},
    {name: 'AVG', detail: 'AVG(expr)'},
    {name: 'MIN', detail: 'MIN(expr)'},
    {name: 'MAX', detail: 'MAX(expr)'},
    {name: 'COALESCE', detail: 'COALESCE(expr1, expr2)'},
    {name: 'CAST', detail: 'CAST(expr AS type)'},
    {name: 'CONCAT', detail: 'CONCAT(expr, ...)'},
];

/* ---------- Create keyword completions (fallback) ---------- */

function createKeywordCompletions(prefix: string, from: number): CompletionResult {
    const keywordCompletions = SQL_KEYWORD_LIST
        .filter(keyword => keyword.toLowerCase().startsWith(prefix.toLowerCase()))
        .map(keyword => createCompletion(keyword, 'keyword', 'SQL Keyword'));

    return {
        from,
        options: keywordCompletions,
        validFor: /^[\w]*$/,
    };
}

/* ---------- Context parsing ---------- */

/**
 * Determine SQL context near cursor.
 * Returns detailed context:
 * - type: 'schema' | 'table' | 'column' | 'mixed' | 'keyword' | null
 * - schema, table, alias if known
 * - tablesInQuery: list of tables+aliases from the current query
 * - tokenBeforeDot: part before the dot if there is a dot token
 */
function parseSQLContext(text: string, pos: number): {
    type: 'schema' | 'table' | 'column' | 'mixed' | 'keyword' | null,
    schema?: string,
    table?: string,
    alias?: string,
    tablesInQuery?: Array<{ schema?: string, table: string, alias?: string }>,
    tokenBeforeDot?: string,
} {
    const beforeCursor = text.slice(0, pos);
    const beforeUpper = beforeCursor.toUpperCase();

    // last indexes of key keywords
    const lastSelect = beforeUpper.lastIndexOf('SELECT');
    const lastFrom = beforeUpper.lastIndexOf('FROM');
    const lastJoin = beforeUpper.lastIndexOf('JOIN');
    const lastWhere = beforeUpper.lastIndexOf('WHERE');
    const lastOn = beforeUpper.lastIndexOf(' ON ');
    const lastWith = beforeUpper.lastIndexOf('WITH');
    const lastGroupBy = beforeUpper.lastIndexOf('GROUP BY');
    const lastOrderBy = beforeUpper.lastIndexOf('ORDER BY');

    // Token right before cursor (may include dots)
    const tokenMatch = /[A-Za-z0-9_\.`]+$/.exec(beforeCursor);
    const token = tokenMatch ? tokenMatch[0] : '';

    // If token contains a dot, separate parts (e.g., schema.table or alias.col or schema.)
    let tokenBeforeDot: string | undefined;
    let tokenAfterDot: string | undefined;
    if (token.includes('.')) {
        const idx = token.lastIndexOf('.');
        tokenBeforeDot = token.substring(0, idx).replace(/`/g, '');
        tokenAfterDot = token.substring(idx + 1);
    }

    // extract table/alias info from entire query
    const tablesInQuery = extractTablesAndAliasesFromQuery(text);

    // If the user has just typed "FROM " or "JOIN " (i.e., ends with those keywords + space)
    if (/\bFROM\s+$/i.test(beforeCursor) || /\bJOIN\s+$/i.test(beforeCursor)) {
        // suggest schemas (top-level) AND tables (unqualified)
        return {type: 'schema', tablesInQuery};
    }

    // If tokenBeforeDot exists and the part before dot is likely a schema (we'll assume any identifier can be a schema)
    if (tokenBeforeDot && !tokenBeforeDot.match(/^\d+$/)) {
        // If we are in a FROM or JOIN clause (immediately after FROM/JOIN) then schema.<cursor> -> table completions
        const afterFromMatch = /\bFROM\s+[A-Za-z0-9_`\.]*$/i.test(beforeCursor);
        const afterJoinMatch = /\bJOIN\s+[A-Za-z0-9_`\.]*$/i.test(beforeCursor);
        if (afterFromMatch || afterJoinMatch) {
            return {type: 'table', schema: tokenBeforeDot, tablesInQuery, tokenBeforeDot};
        }

        // Else if tokenBeforeDot matches an alias/table found in the query, then alias.<cursor> suggestions -> columns
        const matchedTable = tablesInQuery.find(t => t.alias?.toUpperCase() === tokenBeforeDot.toUpperCase() || t.table.toUpperCase() === tokenBeforeDot.toUpperCase());
        if (matchedTable) {
            return {
                type: 'column',
                table: matchedTable.table,
                schema: matchedTable.schema,
                alias: matchedTable.alias,
                tablesInQuery,
                tokenBeforeDot
            };
        }

        // Last fallback: If tokenBeforeDot looks like a schema (we can't be sure), prefer table completion
        return {type: 'table', schema: tokenBeforeDot, tablesInQuery, tokenBeforeDot};
    }

    // If we're after SELECT clause and before FROM: suggest columns + functions + snippets
    if (lastSelect > -1 && (lastFrom === -1 || lastSelect > lastFrom) && (!lastFrom || lastSelect > lastFrom)) {
        // If user is typing SELECT ... FROM later in the query, attempt to detect referenced table in FROM in the remainder of text
        const afterCursor = text.slice(pos);
        const combined = beforeUpper + afterCursor.toUpperCase();
        // Find first FROM after SELECT
        const fromAfterSelect = combined.indexOf('FROM', lastSelect);
        if (fromAfterSelect > -1) {
            // get table after FROM if any
            const fromMatch = /FROM\s+(?:(\w+)\.)?(\w+)/i.exec(combined.slice(fromSelectIndex(combined, lastSelect)));
            // Note: fromSelectIndex helper
        }
        return {type: 'column', tablesInQuery};
    }

    // If we are after WHERE / ON / AND / OR or inside JOIN ... ON: show mixed suggestions (aliases, tables, columns)
    if ((lastWhere > -1 && lastWhere > lastFrom && lastWhere > lastJoin) || (lastOn > -1 && lastOn > lastJoin)) {
        return {type: 'mixed', tablesInQuery};
    }

    // If token is empty but user is in general context, present keywords
    if (!token) {
        return {type: 'keyword', tablesInQuery};
    }

    // If user is typing an isolated identifier (no dot), and we can find tables in query:
    if (token && tablesInQuery.length > 0) {
        // If only one table in query, show its columns
        if (tablesInQuery.length === 1) {
            return {
                type: 'column',
                schema: tablesInQuery[0].schema,
                table: tablesInQuery[0].table,
                alias: tablesInQuery[0].alias,
                tablesInQuery
            };
        } else {
            // multiple tables: assume mixed
            return {type: 'mixed', tablesInQuery};
        }
    }

    // Default: null (let fallback keywords cover)
    return {type: null};
}

function fromSelectIndex(combinedUpper: string, lastSelectIndex: number) {
    // returns slice index where the 'FROM' following SELECT begins or lastSelectIndex if not found.
    const idx = combinedUpper.indexOf('FROM', lastSelectIndex);
    return idx === -1 ? lastSelectIndex : idx;
}

async function sqlAutocomplete(context: CompletionContext, options: AutocompleteOptions): Promise<CompletionResult | null> {
    const {apolloClient} = options;
    const {state, pos} = context;
    const text = state.doc.toString();

    // tokenBeforeCursor capturing identifiers with dots
    const tokenMatch = /[A-Za-z0-9_\.`]+$/.exec(text.slice(0, pos));
    const token = tokenMatch ? tokenMatch[0] : '';
    const from = tokenMatch ? pos - token.length : pos;

    // parse context
    const sqlContext = parseSQLContext(text, pos);

    // Helper to filter by prefix (case-insensitive)
    const applyPrefixFilter = (items: Completion[], prefix: string) => {
        if (!prefix) return items;
        const lower = prefix.toLowerCase();
        return items.filter(i => i.label.toLowerCase().startsWith(lower));
    };

    // Extract last simple word (no dot) for keyword fallback
    const simpleWordMatch = /[A-Za-z0-9_]+$/.exec(text.slice(0, pos));
    const simpleWord = simpleWordMatch ? simpleWordMatch[0] : '';

    // Build completions depending on context
    try {
        let completions: Completion[] = [];

        // If token contains a dot like "schema." or "alias."
        const dotMatch = token && token.includes('.') ? {
            raw: token,
            before: token.slice(0, token.lastIndexOf('.')).replace(/`/g, ''),
            after: token.slice(token.lastIndexOf('.') + 1)
        } : null;

        if (sqlContext.type === 'schema') {
            // Suggest schema names
            const schemas = await fetchSchemas(apolloClient);
            completions = schemas.map(s => createCompletion(s, 'namespace', 'Schema'));
            completions = applyPrefixFilter(completions, token);
            if (completions.length) {
                return {from, options: completions, validFor: /^[\w`\.]*$/};
            }
        }

        if (sqlContext.type === 'table') {
            // If dotMatch exists, tokenBeforeDot is the schema
            const schemaName = (dotMatch && dotMatch.before) || (sqlContext.schema ?? undefined);
            if (schemaName) {
                const tables = await fetchTables(apolloClient, schemaName);
                completions = tables.map((t: any) => createCompletion(t.Name, 'class', 'Table', t.Name));
                // If user typed "schema.tabl" we want to complete just the table part; since 'from' already points to the start of token,
                // applying the full table name will replace schema.tabl -> schema.table which is okay.
                const prefixToFilter = dotMatch ? dotMatch.after : token;
                completions = applyPrefixFilter(completions, prefixToFilter || '');
                if (completions.length) {
                    return {from: from + schemaName.length + 1, options: completions, validFor: /^[\w`\.]*$/};
                }
            } else {
                // No schema known: suggest schemas and unqualified tables (if your backend supports listing database-wide tables you'd fetch them here)
                const schemas = await fetchSchemas(apolloClient);
                completions = schemas.map(s => createCompletion(s, 'namespace', 'Schema'));
                completions = applyPrefixFilter(completions, token);
                if (completions.length) {
                    return {from, options: completions, validFor: /^[\w`\.]*$/};
                }
            }
        }

        if (sqlContext.type === 'column') {
            // Column suggestions — could be alias.col or table.col or plain column in single-table queries
            // If dotMatch exists and before part is alias/table/schema, handle accordingly
            if (dotMatch) {
                const before = dotMatch.before;
                // Is 'before' an alias used in the query?
                const tables = sqlContext.tablesInQuery ?? extractTablesAndAliasesFromQuery(text);
                const matched = tables.find(t => (t.alias && t.alias.toUpperCase() === before.toUpperCase()) || t.table.toUpperCase() === before.toUpperCase());
                if (matched) {
                    // fetch columns for matched.table (and matched.schema if present)
                    const schemaToUse = matched.schema || sqlContext.schema || (matched.table ? detectSchemaFromText(text, matched.table) : undefined);
                    if (matched.table && schemaToUse) {
                        const cols = await fetchColumns(apolloClient, schemaToUse, matched.table);
                        // produce both qualified and unqualified suggestions depending on whether user typed alias or table
                        const qualPrefix = before + '.';
                        completions = cols.map(c => createCompletion(`${qualPrefix}${c.Name}`, 'property', c.Type ?? 'column', `${qualPrefix}${c.Name}`));
                        // also add unqualified column names for quick insertion (if helpful)
                        const unqualified = cols.map(c => createCompletion(c.Name, 'property', c.Type ?? 'column', c.Name));
                        completions = completions.concat(unqualified);
                        const afterPrefix = dotMatch.after || '';
                        completions = applyPrefixFilter(completions, afterPrefix);
                        if (completions.length) {
                            return {from, options: completions, validFor: /^[\w`\.]*$/};
                        }
                    } else if (matched.table) {
                        // no schema but we can still ask backend maybe with default schema
                        const cols = await fetchColumns(apolloClient, matched.schema ?? '', matched.table);
                        completions = cols.map(c => createCompletion(`${before}.${c.Name}`, 'property', c.Type ?? 'column', `${before}.${c.Name}`));
                        completions = applyPrefixFilter(completions, dotMatch.after || '');
                        if (completions.length) {
                            return {from, options: completions, validFor: /^[\w`\.]*$/};
                        }
                    }
                } else {
                    // 'before' might be a schema name; in that case we want tables (handled earlier), or it might be unknown: try tables under that schema
                    const schemaGuess = before;
                    const tables = await fetchTables(apolloClient, schemaGuess);
                    completions = tables.map((t: any) => createCompletion(`${schemaGuess}.${t.Name}`, 'class', 'Table', `${schemaGuess}.${t.Name}`));
                    completions = applyPrefixFilter(completions, dotMatch.after || '');
                    if (completions.length) {
                        return {from, options: completions, validFor: /^[\w`\.]*$/};
                    }
                }
            } else {
                // No dot; if single table in query, list its columns; if multiple, list aliases + columns prefixed by alias
                const tables = sqlContext.tablesInQuery ?? extractTablesAndAliasesFromQuery(text);
                if (tables.length === 1) {
                    const t = tables[0];
                    const schemaToUse = t.schema || sqlContext.schema || detectSchemaFromText(text, t.table);
                    if (t.table && schemaToUse) {
                        const cols = await fetchColumns(apolloClient, schemaToUse, t.table);
                        completions = cols.map(c => createCompletion(c.Name, 'property', c.Type ?? 'column', c.Name));
                        completions = completions.concat(SQL_FUNCTIONS.map(f => createCompletion(f.name, 'function', f.detail, f.name + '()')));
                        completions = applyPrefixFilter(completions, token);
                        if (completions.length) {
                            return {from, options: completions, validFor: /^\w*$/};
                        }
                    }
                } else if (tables.length > 1) {
                    // multiple tables: add aliases + qualified columns
                    for (const t of tables) {
                        if (t.alias) {
                            completions.push(createCompletion(t.alias, 'variable', `Alias for ${t.table}`, t.alias));
                        }
                        completions.push(createCompletion(t.table, 'class', 'Table', t.table));
                        // attempt to fetch columns (best-effort) for top N tables to avoid latency explosion
                    }
                    // Also add SQL functions and snippets
                    completions = completions.concat(SQL_FUNCTIONS.map(f => createCompletion(f.name, 'function', f.detail, f.name + '()')));
                    completions = completions.concat(getSnippets());
                    completions = applyPrefixFilter(completions, token);
                    if (completions.length) {
                        return {from, options: completions, validFor: /^\w*$/};
                    }
                }
            }
        }

        if (sqlContext.type === 'mixed') {
            // Show a combined set: aliases, table names, qualified columns from all tables involved
            const tables = sqlContext.tablesInQuery ?? extractTablesAndAliasesFromQuery(text);
            // Add aliases and table names
            for (const t of tables) {
                if (t.alias) {
                    completions.push(createCompletion(t.alias, 'variable', `Alias for ${t.table}`, t.alias));
                }
                completions.push(createCompletion(t.table, 'class', 'Table', t.table));
            }

            // Fetch columns for up to 3 tables to keep performance reasonable
            const limitFetch = tables.slice(0, 3);
            for (const t of limitFetch) {
                const schemaToUse = t.schema || detectSchemaFromText(text, t.table) || '';
                if (t.table) {
                    const cols = await fetchColumns(apolloClient, schemaToUse, t.table);
                    for (const c of cols) {
                        // add alias-qualified if exists
                        if (t.alias) {
                            completions.push(createCompletion(`${t.alias}.${c.Name}`, 'property', c.Type ?? 'column', `${t.alias}.${c.Name}`));
                        }
                        completions.push(createCompletion(`${t.table}.${c.Name}`, 'property', c.Type ?? 'column', `${t.table}.${c.Name}`));
                        completions.push(createCompletion(c.Name, 'property', c.Type ?? 'column', c.Name));
                    }
                }
            }

            // Add functions and snippets
            completions = completions.concat(SQL_FUNCTIONS.map(f => createCompletion(f.name, 'function', f.detail, f.name + '()')));
            completions = completions.concat(getSnippets());

            completions = applyPrefixFilter(completions, token);
            if (completions.length) {
                return {from, options: completions, validFor: /^[\w`\.]*$/};
            }
        }

        if (sqlContext.type === 'keyword') {
            // show keywords + functions + snippets
            completions = SQL_KEYWORD_LIST.map(k => createCompletion(k, 'keyword', 'SQL Keyword'));
            completions = completions.concat(SQL_FUNCTIONS.map(f => createCompletion(f.name, 'function', f.detail, f.name + '()')));
            completions = completions.concat(getSnippets());
            completions = applyPrefixFilter(completions, simpleWord || token);
            if (completions.length) {
                return {from: simpleWord ? pos - simpleWord.length : from, options: completions, validFor: /^[\w]*$/};
            }
        }

    } catch (err) {
        console.error('sqlAutocomplete error:', err);
        // Fall through to keyword fallback
    }

    // Fallback: keywords/functions/snippets filtered by user input (if any)
    const fallbackPrefix = simpleWord || token || '';
    return createKeywordCompletions(fallbackPrefix, from);
}

function getSnippets(): Completion[] {
    const snippets: Completion[] = [];

    snippets.push(createCompletion('JOIN ... ON ...', 'snippet', 'Snippet: JOIN with ON', 'JOIN schema.table alias ON alias.column = other_alias.column'));
    snippets.push(createCompletion('LEFT JOIN ... ON ...', 'snippet', 'Snippet: LEFT JOIN with ON', 'LEFT JOIN schema.table alias ON alias.column = other_alias.column'));
    snippets.push(createCompletion('WHERE IN (...)', 'snippet', 'Snippet: WHERE IN', 'WHERE column IN (value1, value2)'));
    snippets.push(createCompletion('GROUP BY ...', 'snippet', 'Snippet: GROUP BY', 'GROUP BY column1, column2'));
    snippets.push(createCompletion('WITH CTE AS (...) SELECT ...', 'snippet', 'Snippet: CTE (WITH)', 'WITH cte_name AS (\n  SELECT columns FROM schema.table\n)\nSELECT * FROM cte_name'));
    snippets.push(createCompletion('SELECT DISTINCT ...', 'snippet', 'Snippet: SELECT DISTINCT', 'SELECT DISTINCT column FROM schema.table'));

    return snippets;
}

function detectSchemaFromText(text: string, tableName: string): string | undefined {
    // Search for pattern schema.tableName
    const re = new RegExp(`([A-Za-z0-9_]+)\\.${tableName}\\b`, 'i');
    const m = re.exec(text);
    if (m) return m[1];
    return undefined;
}

export function createSQLAutocomplete(options: AutocompleteOptions) {
    return autocompletion({
        override: [
            (context: CompletionContext) => sqlAutocomplete(context, options),
        ],
        activateOnTyping: true,
        defaultKeymap: true,
    });
}
