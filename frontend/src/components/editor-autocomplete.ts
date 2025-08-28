/**
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

import { CompletionContext, CompletionResult, autocompletion, Completion } from "@codemirror/autocomplete";
import { ApolloClient } from "@apollo/client";
import { ColumnsDocument, GetSchemaDocument, GetStorageUnitsDocument } from "@graphql";

interface AutocompleteOptions {
  apolloClient: ApolloClient<any>;
}

// SQL keywords to ignore when determining context
const SQL_KEYWORDS = new Set([
  'SELECT', 'FROM', 'WHERE', 'JOIN', 'INNER', 'LEFT', 'RIGHT', 'OUTER',
  'ON', 'AND', 'OR', 'ORDER', 'BY', 'GROUP', 'HAVING', 'LIMIT', 'OFFSET',
  'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'DROP', 'ALTER', 'TABLE', 'INTO'
]);

// Parse the SQL context to determine what type of suggestions to provide
function parseSQLContext(text: string, pos: number): { type: 'schema' | 'table' | 'column' | 'mixed' | null, schema?: string, table?: string, alias?: string, showTables?: boolean, singleTable?: boolean } {
  const beforeCursor = text.slice(0, pos);
  const beforeCursorUpper = beforeCursor.toUpperCase();
  const lines = beforeCursorUpper.split('\n');
  const currentLine = lines[lines.length - 1];
  
  // Find the last occurrence of key SQL keywords
  const lastSelect = beforeCursorUpper.lastIndexOf('SELECT');
  const lastFrom = beforeCursorUpper.lastIndexOf('FROM');
  const lastJoin = beforeCursorUpper.lastIndexOf('JOIN');
  const lastWhere = beforeCursorUpper.lastIndexOf('WHERE');
  
  // Check if we just typed FROM or JOIN
  if (currentLine.endsWith('FROM ') || currentLine.endsWith('JOIN ')) {
    return { type: 'schema' };
  }
  
  // Check for schema.table pattern (for table completion)
  const schemaTableMatch = /(\w+)\.$/i.exec(text.slice(Math.max(0, pos - 100), pos));
  if (schemaTableMatch) {
    // Determine if this is for table or column based on context
    const beforeDot = beforeCursorUpper.slice(0, pos - 1);
    
    // If we're after FROM or JOIN and no table selected yet, it's a table completion
    if ((beforeDot.match(/FROM\s+\w*$/i) || beforeDot.match(/JOIN\s+\w*$/i)) && 
        (!lastWhere || pos < lastWhere)) {
      return { type: 'table', schema: schemaTableMatch[1] };
    }
    
    // Otherwise it's a column completion with table prefix
    return { type: 'column', table: schemaTableMatch[1] };
  }
  
  // Extract all tables and their aliases from the query
  const extractTablesAndAliases = () => {
    const tables: Array<{schema?: string, table: string, alias?: string}> = [];
    // Match patterns like: FROM schema.table alias, FROM table AS alias, FROM table alias
    const tablePatterns = [
      /FROM\s+(?:(\w+)\.)?(\w+)(?:\s+(?:AS\s+)?(\w+))?/gi,
      /JOIN\s+(?:(\w+)\.)?(\w+)(?:\s+(?:AS\s+)?(\w+))?/gi
    ];
    
    for (const pattern of tablePatterns) {
      let match;
      while ((match = pattern.exec(beforeCursorUpper)) !== null) {
        tables.push({
          schema: match[1] || undefined,
          table: match[2],
          alias: match[3] || undefined
        });
      }
    }
    
    return tables;
  };
  
  const tables = extractTablesAndAliases();
  const singleTable = tables.length === 1;
  
  // Check if we're immediately after WHERE with no additional context
  if (lastWhere > -1 && lastWhere > lastFrom && lastWhere > lastJoin) {
    const afterWhere = beforeCursorUpper.slice(lastWhere + 5).trim();
    
    // If we're right after WHERE or after WHERE followed by whitespace only
    if (!afterWhere || afterWhere.match(/^\s*$/)) {
      return { 
        type: 'mixed',
        showTables: true,
        singleTable,
        schema: singleTable ? tables[0].schema : undefined,
        table: singleTable ? tables[0].table : undefined
      };
    }
  }
  
  // Check if we're in a SELECT context
  if (lastSelect > -1 && (!lastFrom || lastSelect > lastFrom)) {
    // We're in SELECT clause, show columns
    // Try to find table from a future FROM clause
    const afterCursor = text.slice(pos).toUpperCase();
    const fullText = beforeCursorUpper + afterCursor;
    const fromMatch = /FROM\s+(?:(\w+)\.)?(\w+)/i.exec(fullText.slice(lastSelect));
    
    if (fromMatch) {
      return { 
        type: 'column', 
        schema: fromMatch[1] || undefined,
        table: fromMatch[2]
      };
    }
  }
  
  // Check if we're in a context after WHERE but with some content
  if (lastWhere > -1 && lastWhere > lastFrom && lastWhere > lastJoin) {
    // Extract table name from the query
    if (tables.length > 0) {
      // If single table, return column context
      if (singleTable) {
        return { 
          type: 'column', 
          schema: tables[0].schema,
          table: tables[0].table
        };
      }
    }
  }
  
  // Check if we're in a context where columns make sense
  // This includes: after SELECT, after comma in SELECT, after WHERE, after AND/OR, etc.
  const recentText = beforeCursorUpper.slice(Math.max(0, pos - 50));
  if (recentText.match(/SELECT\s+\w*$/i) || 
      recentText.match(/,\s*\w*$/i) ||
      recentText.match(/WHERE\s+\w*$/i) ||
      recentText.match(/\s+(AND|OR)\s+\w*$/i) ||
      recentText.match(/\s+ON\s+\w*$/i) ||
      recentText.match(/\s+=\s*\w*$/i) ||
      recentText.match(/\s+(<|>|<=|>=|<>|!=)\s*\w*$/i)) {
    
    // Try to find the main table from the query
    if (tables.length > 0) {
      // If single table, return its info
      if (singleTable) {
        return { 
          type: 'column', 
          schema: tables[0].schema,
          table: tables[0].table
        };
      }
      // Multiple tables, just use the first one for now
      return { 
        type: 'column', 
        schema: tables[0].schema,
        table: tables[0].table
      };
    }
  }
  
  return { type: null };
}

// Extract tables and aliases from the full query
function extractTablesAndAliasesFromQuery(text: string): Array<{schema?: string, table: string, alias?: string}> {
  const tables: Array<{schema?: string, table: string, alias?: string}> = [];
  const textUpper = text.toUpperCase();
  
  // Match patterns like: FROM schema.table alias, FROM table AS alias, FROM table alias
  const tablePatterns = [
    /FROM\s+(?:(\w+)\.)?(\w+)(?:\s+(?:AS\s+)?(\w+))?/gi,
    /JOIN\s+(?:(\w+)\.)?(\w+)(?:\s+(?:AS\s+)?(\w+))?/gi
  ];
  
  for (const pattern of tablePatterns) {
    let match;
    while ((match = pattern.exec(textUpper)) !== null) {
      tables.push({
        schema: match[1] || undefined,
        table: match[2],
        alias: match[3] || undefined
      });
    }
  }
  
  return tables;
}

// Create completion items with proper formatting
function createCompletion(label: string, type: string, detail?: string): Completion {
  return {
    label,
    type,
    detail: detail || type,
    apply: label,
  };
}

// SQL keywords for fallback completion
const SQL_KEYWORD_LIST = [
  'SELECT', 'FROM', 'WHERE', 'JOIN', 'INNER', 'LEFT', 'RIGHT', 'OUTER', 'FULL',
  'ON', 'AND', 'OR', 'NOT', 'NULL', 'TRUE', 'FALSE', 'ORDER', 'BY', 'GROUP',
  'HAVING', 'LIMIT', 'OFFSET', 'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'DROP',
  'ALTER', 'TABLE', 'INDEX', 'VIEW', 'DATABASE', 'SCHEMA', 'INTO', 'VALUES',
  'SET', 'AS', 'DISTINCT', 'COUNT', 'SUM', 'AVG', 'MAX', 'MIN', 'CASE', 'WHEN',
  'THEN', 'ELSE', 'END', 'IF', 'EXISTS', 'LIKE', 'IN', 'BETWEEN', 'IS', 'ASC',
  'DESC', 'UNION', 'ALL', 'ANY', 'SOME', 'WITH', 'RECURSIVE', 'CASCADE',
  'CONSTRAINT', 'PRIMARY', 'KEY', 'FOREIGN', 'REFERENCES', 'UNIQUE', 'CHECK',
  'DEFAULT', 'AUTO_INCREMENT', 'SERIAL', 'TRUNCATE', 'EXPLAIN', 'ANALYZE',
  'GRANT', 'REVOKE', 'COMMIT', 'ROLLBACK', 'TRANSACTION', 'BEGIN', 'START'
];

// Create SQL keyword completions
function createKeywordCompletions(prefix: string, from: number): CompletionResult {
  const keywordCompletions = SQL_KEYWORD_LIST
    .filter(keyword => keyword.toLowerCase().startsWith(prefix.toLowerCase()))
    .map(keyword => createCompletion(keyword, 'keyword', 'SQL Keyword'));
    
  return {
    from,
    options: keywordCompletions,
    validFor: /^\w*$/,
  };
}

// Main autocomplete function
async function sqlAutocomplete(context: CompletionContext, options: AutocompleteOptions): Promise<CompletionResult | null> {
  const { apolloClient } = options;
  const { state, pos } = context;
  const text = state.doc.toString();
  
  // Parse the SQL context
  const sqlContext = parseSQLContext(text, pos);
  
  const word = context.matchBefore(/\w*/);
  const from = word?.from ?? pos;
  
  // Try to get custom completions first if we have a specific context
  if (sqlContext.type) {
    try {
      let completions: Completion[] = [];
      
      switch (sqlContext.type) {
        case 'schema':
          // Fetch all schemas
          const schemasResult = await apolloClient.query({
            query: GetSchemaDocument,
            fetchPolicy: 'cache-first',
          });
          
          if (schemasResult.data?.Schema) {
            completions = schemasResult.data.Schema.map((schema: string) => 
              createCompletion(schema, 'namespace', 'Schema')
            );
          }
          break;
          
        case 'table':
          // Fetch tables for the specified schema
          if (sqlContext.schema) {
            const tablesResult = await apolloClient.query({
              query: GetStorageUnitsDocument,
              variables: { schema: sqlContext.schema },
              fetchPolicy: 'cache-first',
            });
            
            if (tablesResult.data?.StorageUnit) {
              completions = tablesResult.data.StorageUnit.map((unit: any) => 
                createCompletion(unit.Name, 'class', 'Table')
              );
            }
          }
          break;
          
        case 'column':
          // Fetch columns for the specified table
          if (sqlContext.table) {
            // If we have a schema from the context, use it
            let schema = sqlContext.schema;
            
            // If no schema in context, try to find it from the query
            if (!schema) {
              const schemaMatch = new RegExp(`(\\w+)\\.${sqlContext.table}`, 'i').exec(text);
              schema = schemaMatch?.[1];
            }
            
            if (schema) {
              const columnsResult = await apolloClient.query({
                query: ColumnsDocument,
                variables: { 
                  schema: schema,
                  storageUnit: sqlContext.table 
                },
                fetchPolicy: 'cache-first',
              });
              
              if (columnsResult.data?.Columns) {
                completions = columnsResult.data.Columns.map((column: any) => 
                  createCompletion(column.Name, 'property', column.Type)
                );
              }
            }
          }
          break;
          
        case 'mixed':
          // Mixed context - show both tables and columns
          if (sqlContext.showTables) {
            // First, get all tables from the query
            const tables = extractTablesAndAliasesFromQuery(text);
            
            // Add table/alias suggestions
            for (const tableInfo of tables) {
              // Add alias if it exists
              if (tableInfo.alias) {
                completions.push(createCompletion(tableInfo.alias, 'variable', `Alias for ${tableInfo.table}`));
              }
              // Add table name
              completions.push(createCompletion(tableInfo.table, 'class', 'Table'));
            }
            
            // If single table, also fetch its columns
            if (sqlContext.singleTable && sqlContext.table && sqlContext.schema) {
              const columnsResult = await apolloClient.query({
                query: ColumnsDocument,
                variables: { 
                  schema: sqlContext.schema,
                  storageUnit: sqlContext.table 
                },
                fetchPolicy: 'cache-first',
              });
              
              if (columnsResult.data?.Columns) {
                const columnCompletions = columnsResult.data.Columns.map((column: any) => 
                  createCompletion(column.Name, 'property', column.Type)
                );
                completions = completions.concat(columnCompletions);
              }
            }
          }
          break;
      }
      
      // Filter completions based on the current word
      if (word && completions.length > 0) {
        const prefix = text.slice(word.from, pos).toLowerCase();
        completions = completions.filter(c => 
          c.label.toLowerCase().startsWith(prefix)
        );
      }
      
      // If we have custom completions, return them
      if (completions.length > 0) {
        return {
          from,
          options: completions,
          validFor: /^\w*$/,
        };
      }
      
    } catch (error) {
      console.error('Error fetching autocomplete suggestions:', error);
      // Fall through to keyword completions
    }
  }
  
  // Fallback to SQL keywords if no custom completions or if we have a word to complete
  if (word && word.text.length > 0) {
    return createKeywordCompletions(word.text, from);
  }
  
  // Show all keywords if no word typed yet
  if (!word || word.text.length === 0) {
    return createKeywordCompletions('', from);
  }
  
  return null;
}

// Export the autocomplete extension
export function createSQLAutocomplete(options: AutocompleteOptions) {
  return autocompletion({
    override: [
      (context: CompletionContext) => sqlAutocomplete(context, options),
    ],
    activateOnTyping: true,
  });
}