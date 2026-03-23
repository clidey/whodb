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

const SQL_SAFE_KEYWORDS = ['SELECT', 'WITH', 'EXPLAIN', 'DESCRIBE', 'SHOW', 'USE'];

const SQL_DESTRUCTIVE_KEYWORDS = [
    'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'DROP', 'ALTER',
    'TRUNCATE', 'REPLACE', 'MERGE', 'CALL', 'EXEC', 'EXECUTE',
];

const SQL_ALL_KEYWORDS = [
    ...SQL_SAFE_KEYWORDS,
    'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'DROP', 'ALTER', 'SET',
];

export const isValidSQLQuery = (text: string): boolean => {
    const trimmed = text.trim();
    if (!trimmed) return false;
    const upperText = trimmed.toUpperCase();
    return SQL_ALL_KEYWORDS.some(keyword => upperText.startsWith(keyword));
};

export const isDestructiveQuery = (text: string): boolean => {
    const trimmed = text.trim();
    if (!trimmed) return false;
    const upperText = trimmed.toUpperCase();
    if (SQL_SAFE_KEYWORDS.some(keyword => upperText.startsWith(keyword))) return false;
    if (SQL_DESTRUCTIVE_KEYWORDS.some(keyword => upperText.startsWith(keyword))) return true;
    // For anything else (non-SQL, SET statements, etc.), consider potentially destructive
    return true;
};
