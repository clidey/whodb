// Agentic Chat Mode for WhoDB
// Called in a loop to explore schema, execute queries, self-correct, and build answers

// Action types the agent can take
enum AgentActionType {
  ListTables @alias("list_tables") @description("List all tables in the schema")
  DescribeTable @alias("describe_table") @description("Get columns and types for a specific table")
  ShowRelationships @alias("show_relationships") @description("Get foreign key relationships for a table")
  ExecuteSQL @alias("execute_sql") @description("Execute a SQL query against the database")
  FinalAnswer @alias("final_answer") @description("Return the final answer to the user")
}

// Structured action returned by each agent step
class AgentAction {
  action AgentActionType @description("The action to take")
  argument string? @description("Table name for describe_table/show_relationships, or SQL query for execute_sql")
  message string? @description("Status message to show the user explaining what you're doing")
  sql string? @description("For final_answer: optional SQL query to display alongside the answer")
  requires_confirmation bool @description("Whether this action requires user confirmation before executing (for mutations)")
}

// Single step in the agent loop
function AgentStep(
  context: DatabaseContext,
  tool_history: string,
  user_query: string
) -> AgentAction {
  client DefaultClient
  prompt #"
    You are a database assistant that explores schemas, runs queries, and answers questions step by step.

    Database: {{ context.database_type }}
    Schema: {{ context.schema }}

    Available Tables and Fields:
    {{ context.tables_and_fields }}

    Previous Conversation:
    {{ context.previous_conversation }}

    ### Tool History (actions taken so far in this turn):
    {{ tool_history }}

    ### User's Question:
    {{ user_query }}

    ### Available Actions:
    You must return ONE action per step. Choose the best next action:

    1. **list_tables** - List all tables. Use when you need to discover what tables exist.
       - argument: not needed
       - message: explain what you're doing

    2. **describe_table** - Get columns for a specific table. Use to understand table structure before writing SQL.
       - argument: the table name (without schema prefix)
       - message: explain what you're doing

    3. **show_relationships** - Get foreign key relationships for a table. Use to understand joins.
       - argument: the table name (without schema prefix)
       - message: explain what you're doing

    4. **execute_sql** - Run a SQL query. Use after you understand the schema well enough.
       - argument: the complete SQL query
       - message: explain what the query does
       {% if context.schema %}
       - IMPORTANT: EVERY table reference MUST include the schema: {{ context.schema }}.table_name
       {% else %}
       - IMPORTANT: Do NOT add any schema prefix to table names
       {% endif %}

    5. **final_answer** - Return your answer to the user. Use when you have all information needed.
       - argument: not needed
       - message: your complete answer to the user
       - sql: optionally include a SQL query to display

    ### Strategy:
    - If the user asks about data, explore the schema first (describe relevant tables), then write SQL
    - If a previous SQL query errored (visible in tool_history), analyze the error and fix the query
    - If you already have query results in tool_history, summarize them in your final_answer
    - For mutations (INSERT/UPDATE/DELETE/CREATE/ALTER/DROP), set requires_confirmation to true
    - Be efficient: don't list_tables if tables_and_fields already has the info
    - Reach final_answer within a few steps - don't over-explore
    - NEVER repeat an action that already succeeded in tool_history

    {{ ctx.output_format }}
  "#
}

// Summarize conversation history when it gets too long
function SummarizeConversation(
  conversation: string,
  database_type: string
) -> string {
  client DefaultClient
  prompt #"
    Summarize this database chat conversation concisely, preserving:
    - Key facts learned about the database schema
    - Important query results and their meaning
    - The user's ongoing goals or questions
    - Any errors encountered and how they were resolved

    Database type: {{ database_type }}

    Conversation to summarize:
    {{ conversation }}

    Return a concise summary (aim for ~20% of the original length).
    Focus on information that would be needed to continue the conversation.

    {{ ctx.output_format }}
  "#
}
