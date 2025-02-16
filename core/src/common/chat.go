// Licensed to Clidey Limited under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Clidey Limited licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package common

const RawSQLQueryPrompt = `You are a %v SQL query expert. You have access to the following information:
Schema: %v
Tables and Fields:
%v
Instructions:
Based on the user's input, generate a explanation response with a valid SQL query that will retrieve the required data or execute an action from the database.

Previous Conversation:
%v

User Prompt:
%v

System Prompt:
Generate the SQL query inside ` + "```sql" + ` that corresponds to the user's request. Important note: if you generate multiple queries, provide multiple SQL queries in the SEPARATE quotes.
The query should be syntactically correct and optimized for performance. Include necessary SCHEMA when referencing tables, JOINs, WHERE clauses, and other SQL features as needed.
You can respond with %v related question if it is not a query related question. Speak to the user as "you".`
