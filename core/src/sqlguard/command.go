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

package sqlguard

import "strings"

// mongoReadMethods are the shell methods that only read. Anything else — including
// an unrecognized method — is treated as mutating, because the Mongo command
// surface includes drop, dropDatabase, and bulkWrite.
var mongoReadMethods = map[string]bool{
	"find": true, "findone": true, "countdocuments": true, "count": true,
	"aggregate": true, "distinct": true, "estimateddocumentcount": true,
	"getindexes": true, "stats": true, "explain": true,
}

// redisReadCommands are the Redis commands that only read. Anything else is
// treated as mutating: the write surface is large (SET, DEL, FLUSHALL, EVAL,
// CONFIG SET, SHUTDOWN…) and an allowlist is the only safe direction.
var redisReadCommands = map[string]bool{
	"GET": true, "MGET": true, "STRLEN": true, "GETRANGE": true,
	"EXISTS": true, "TYPE": true, "TTL": true, "PTTL": true,
	"KEYS": true, "SCAN": true, "RANDOMKEY": true, "DBSIZE": true,
	"HGET": true, "HMGET": true, "HGETALL": true, "HKEYS": true, "HVALS": true,
	"HLEN": true, "HEXISTS": true, "HSCAN": true, "HRANDFIELD": true,
	"LRANGE": true, "LLEN": true, "LINDEX": true, "LPOS": true,
	"SMEMBERS": true, "SISMEMBER": true, "SMISMEMBER": true, "SCARD": true,
	"SSCAN": true, "SRANDMEMBER": true, "SINTER": true, "SUNION": true, "SDIFF": true,
	"ZRANGE": true, "ZREVRANGE": true, "ZRANGEBYSCORE": true, "ZREVRANGEBYSCORE": true,
	"ZRANGEBYLEX": true, "ZSCORE": true, "ZCARD": true, "ZCOUNT": true,
	"ZRANK": true, "ZREVRANK": true, "ZSCAN": true, "ZMSCORE": true,
	"XRANGE": true, "XREVRANGE": true, "XLEN": true, "XINFO": true,
	"GETBIT": true, "BITCOUNT": true, "BITPOS": true,
	"PFCOUNT": true, "OBJECT": true, "MEMORY": true, "CONFIG": true,
	"INFO": true, "PING": true, "ECHO": true, "TIME": true, "LOLWUT": true,
	"CLIENT": true, "COMMAND": true, "LASTSAVE": true,
}

// ClassifyCommand classifies a source-native command for sources whose query
// surface is not SQL. Unknown source types fall through to Classify, and every
// unrecognized command is reported as mutating so a source-specific write verb
// can never reach execution without confirmation.
func ClassifyCommand(sourceType, command string) Classification {
	switch normalizeSourceType(sourceType) {
	case "mongodb", "mongo", "documentdb":
		return classifyMongo(command)
	case "redis", "valkey":
		return classifyRedis(command)
	default:
		return Classify(command)
	}
}

// normalizeSourceType lowercases and strips separators so "MongoDB",
// "mongo_db", and "mongo-db" all resolve to the same key.
func normalizeSourceType(sourceType string) string {
	lower := strings.ToLower(strings.TrimSpace(sourceType))
	lower = strings.ReplaceAll(lower, "-", "")
	lower = strings.ReplaceAll(lower, "_", "")
	lower = strings.ReplaceAll(lower, " ", "")
	return lower
}

// classifyMongo reads the method name out of a `db.collection.method(...)` or
// `db.method(...)` shell command.
func classifyMongo(command string) Classification {
	method := mongoMethodName(command)
	if method == "" {
		return Classification{Type: StatementUnknown, Mutating: true, Reason: "unparsable Mongo command is treated as mutating"}
	}
	if mongoReadMethods[method] {
		return Classification{Type: StatementCommand, Reason: "Mongo " + method + " only reads"}
	}
	return Classification{Type: StatementCommand, Mutating: true, Reason: "Mongo " + method + " is not a known read method"}
}

// mongoMethodName returns the lowercased trailing method of a shell command, or
// "" when the command has no recognizable `.method(` form.
func mongoMethodName(command string) string {
	trimmed := strings.TrimSpace(command)
	paren := strings.Index(trimmed, "(")
	if paren < 0 {
		paren = len(trimmed)
	}
	head := strings.TrimSpace(trimmed[:paren])
	dot := strings.LastIndex(head, ".")
	if dot < 0 || dot == len(head)-1 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(head[dot+1:]))
}

// classifyRedis reads the verb out of a Redis command line. Subcommand-bearing
// verbs (CONFIG, CLIENT, OBJECT, MEMORY, XINFO, COMMAND) are read-only only for
// their GET/LIST/INFO forms.
func classifyRedis(command string) Classification {
	fields := strings.Fields(strings.ToUpper(strings.TrimSpace(command)))
	if len(fields) == 0 {
		return Classification{Type: StatementUnknown, Mutating: true, Reason: "empty Redis command is treated as mutating"}
	}
	verb := strings.Trim(fields[0], "\"'`")
	if !redisReadCommands[verb] {
		return Classification{Type: StatementCommand, Mutating: true, Reason: "Redis " + verb + " is not a known read command"}
	}
	if redisSubcommandVerbs[verb] {
		if len(fields) < 2 || !redisReadSubcommands[fields[1]] {
			sub := ""
			if len(fields) > 1 {
				sub = " " + fields[1]
			}
			return Classification{Type: StatementCommand, Mutating: true, Reason: "Redis " + verb + sub + " is not a known read subcommand"}
		}
	}
	return Classification{Type: StatementCommand, Reason: "Redis " + verb + " only reads"}
}

// redisSubcommandVerbs are read-listed verbs whose write forms live in a
// subcommand (e.g. CONFIG SET, CLIENT KILL, MEMORY PURGE).
var redisSubcommandVerbs = map[string]bool{
	"CONFIG": true, "CLIENT": true, "OBJECT": true, "MEMORY": true,
	"XINFO": true, "COMMAND": true,
}

// redisReadSubcommands are the subcommand forms of redisSubcommandVerbs that
// only read.
var redisReadSubcommands = map[string]bool{
	"GET": true, "LIST": true, "INFO": true, "COUNT": true, "DOCS": true,
	"STATS": true, "USAGE": true, "ENCODING": true, "FREQ": true,
	"IDLETIME": true, "REFCOUNT": true, "HELP": true, "GETNAME": true,
	"ID": true, "STREAM": true, "GROUPS": true, "CONSUMERS": true,
	"DOCTOR": true, "MALLOC-STATS": true, "GETKEYS": true, "GETKEYSANDFLAGS": true,
}
