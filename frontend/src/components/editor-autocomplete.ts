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
function parseSQLContext(text: string, pos: number): { type: 'schema' | 'table' | 'column' | null, schema?: string, table?: string } {
  const beforeCursor = text.slice(0, pos).toUpperCase();
  const lines = beforeCursor.split('\n');
  const currentLine = lines[lines.length - 1];
  
  // Find the last occurrence of FROM, JOIN, or WHERE
  const lastFrom = beforeCursor.lastIndexOf('FROM');
  const lastJoin = beforeCursor.lastIndexOf('JOIN');
  const lastWhere = beforeCursor.lastIndexOf('WHERE');
  
  // Check if we're in a FROM or JOIN context
  const inFromContext = lastFrom > -1 && (lastWhere === -1 || lastFrom > lastWhere);
  const inJoinContext = lastJoin > -1 && lastJoin > lastFrom && (lastWhere === -1 || lastJoin > lastWhere);
  
  // Check if we just typed FROM or JOIN
  if (currentLine.endsWith('FROM ') || currentLine.endsWith('JOIN ')) {
    return { type: 'schema' };
  }
  
  // Check for schema.table pattern
  const schemaTableMatch = /(\w+)\.$/i.exec(text.slice(Math.max(0, pos - 100), pos));
  if (schemaTableMatch) {
    return { type: 'table', schema: schemaTableMatch[1] };
  }
  
  // Check if we're after WHERE or in a JOIN ON clause
  if (lastWhere > -1 && lastWhere > lastFrom && lastWhere > lastJoin) {
    // Extract table name from the query
    const tableMatch = /FROM\s+(?:(\w+)\.)?(\w+)/i.exec(beforeCursor);
    if (tableMatch) {
      return { 
        type: 'column', 
        schema: tableMatch[1] || undefined,
        table: tableMatch[2]
      };
    }
  }
  
  // Check for table.column pattern in WHERE or JOIN ON
  const tableColumnMatch = /(\w+)\.$/i.exec(text.slice(Math.max(0, pos - 50), pos));
  if (tableColumnMatch && (lastWhere > -1 || currentLine.includes(' ON '))) {
    return { type: 'column', table: tableColumnMatch[1] };
  }
  
  return { type: null };
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