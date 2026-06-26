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

const MONGO_SAFE_METHODS = new Set(['find', 'findOne', 'countDocuments', 'aggregate', 'distinct']);
const MONGO_SOURCE_TYPES = new Set(['mongodb', 'documentdb', 'ferretdb']);

const REDIS_SAFE_COMMANDS = new Set([
    'BITCOUNT', 'BITPOS', 'DBSIZE', 'DUMP', 'EXISTS', 'GET', 'GETBIT', 'GETRANGE',
    'HEXISTS', 'HGET', 'HGETALL', 'HKEYS', 'HLEN', 'HMGET', 'HRANDFIELD', 'HSCAN',
    'HSTRLEN', 'HVALS', 'INFO', 'KEYS', 'LINDEX', 'LLEN', 'LPOS', 'LRANGE',
    'MGET', 'OBJECT', 'PFCOUNT', 'PTTL', 'RANDOMKEY', 'SCAN', 'SCARD', 'SDIFF',
    'SINTER', 'SISMEMBER', 'SMEMBERS', 'SMISMEMBER', 'SRANDMEMBER',
    'STRLEN', 'SUNION', 'TTL', 'TYPE', 'XLEN', 'XPENDING', 'XRANGE', 'XREAD',
    'XREVRANGE', 'ZCARD', 'ZCOUNT', 'ZDIFF', 'ZINTER', 'ZLEXCOUNT', 'ZRANDMEMBER',
    'ZRANGE', 'ZRANGEBYLEX', 'ZRANGEBYSCORE', 'ZRANK', 'ZREVRANGE',
    'ZREVRANGEBYLEX', 'ZREVRANGEBYSCORE', 'ZREVRANK', 'ZSCAN', 'ZSCORE', 'ZUNION',
]);
const REDIS_SOURCE_TYPES = new Set(['redis', 'elasticache', 'valkey', 'dragonfly']);

const MONGO_SHELL_METHOD_PATTERN = /^\s*db(?:\.getCollection\(\s*(?:"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*')\s*\)|\.[A-Za-z_][\w.]*)\.(\w+)\s*\(/s;
const MONGO_AGGREGATE_WRITE_STAGE_PATTERN = /(?:^|[{,]\s*)(?:"\$out"|'\$out'|\$out|"\$merge"|'\$merge'|\$merge)\s*:/s;

export const isValidSQLQuery = (text: string): boolean => {
    const trimmed = text.trim();
    if (!trimmed) return false;
    const upperText = trimmed.toUpperCase();
    return SQL_ALL_KEYWORDS.some(keyword => upperText.startsWith(keyword));
};

/**
 * Returns whether a scratchpad query should require write confirmation.
 */
export const isDestructiveQuery = (text: string, sourceType?: string): boolean => {
    const trimmed = text.trim();
    if (!trimmed) return false;
    const normalizedSourceType = sourceType?.toLowerCase();
    if (normalizedSourceType != null && MONGO_SOURCE_TYPES.has(normalizedSourceType)) {
        return isDestructiveMongoQuery(trimmed);
    }
    if (normalizedSourceType != null && REDIS_SOURCE_TYPES.has(normalizedSourceType)) {
        return isDestructiveRedisQuery(trimmed);
    }
    if (MONGO_SHELL_METHOD_PATTERN.test(trimmed)) {
        return isDestructiveMongoQuery(trimmed);
    }
    const upperText = trimmed.toUpperCase();
    if (SQL_SAFE_KEYWORDS.some(keyword => upperText.startsWith(keyword))) return false;
    if (SQL_DESTRUCTIVE_KEYWORDS.some(keyword => upperText.startsWith(keyword))) return true;
    // For anything else (non-SQL, SET statements, etc.), consider potentially destructive
    return true;
};

function isDestructiveMongoQuery(text: string): boolean {
    const methodMatch = text.match(MONGO_SHELL_METHOD_PATTERN);
    if (methodMatch == null) {
        return true;
    }
    if (methodMatch[1] === 'aggregate') {
        return MONGO_AGGREGATE_WRITE_STAGE_PATTERN.test(text);
    }
    return !MONGO_SAFE_METHODS.has(methodMatch[1]);
}

function isDestructiveRedisQuery(text: string): boolean {
    const command = text.trim().split(/\s+/, 1)[0]?.toUpperCase();
    if (!command) {
        return false;
    }
    return !REDIS_SAFE_COMMANDS.has(command);
}
