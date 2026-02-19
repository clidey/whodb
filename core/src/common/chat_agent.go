//go:build !arm && !riscv64

package common

import (
	"context"
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/baml_client"
	"github.com/clidey/whodb/core/baml_client/types"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

const maxAgentSteps = 10
const maxToolResultChars = 2000

// Emitter abstracts how agent output reaches the client.
// The SSE streaming path and the non-streaming Wails path each provide their own implementation.
type Emitter interface {
	SendChunk(chunk map[string]any)
	SendMessage(msg *model.AIChatMessage)
	SendDone()
}

// RunAgenticChat executes the agent loop: calls BAML AgentStep repeatedly,
// dispatches tool actions, and emits results via the provided Emitter.
// Returns nil on success, or an error if the first BAML call fails
// (caller should fall back to the existing BAML streaming path).
func RunAgenticChat(
	ctx context.Context,
	emitter Emitter,
	plugin *engine.Plugin,
	config *engine.PluginConfig,
	schema string,
	tableDetails string,
	previousConversation string,
	userQuery string,
) error {
	conversation := ManageConversationContext(ctx, previousConversation, config.ExternalModel, config.Credentials.Type)

	dbContext := types.DatabaseContext{
		Database_type:         config.Credentials.Type,
		Schema:                schema,
		Tables_and_fields:     tableDetails,
		Previous_conversation: conversation,
	}

	callOpts := SetupAIClientWithLogging(config.ExternalModel)
	var toolHistory strings.Builder

	for step := 0; step < maxAgentSteps; step++ {
		action, err := baml_client.AgentStep(ctx, dbContext, toolHistory.String(), userQuery, callOpts...)
		if err != nil {
			log.Logger.WithError(err).Errorf("Agentic chat: AgentStep failed at step %d", step)
			return err
		}

		log.DebugFileAlways("Agentic chat step %d: action=%s", step, action.Action)

		if msg := derefStr(action.Message); msg != "" {
			emitter.SendChunk(map[string]any{"type": "message", "text": msg})
		}

		switch action.Action {
		case types.AgentActionTypeListTables:
			handleListTables(&toolHistory, plugin, config, schema)

		case types.AgentActionTypeDescribeTable:
			handleDescribeTable(&toolHistory, plugin, config, schema, derefStr(action.Argument))

		case types.AgentActionTypeShowRelationships:
			handleShowRelationships(&toolHistory, plugin, config, schema, derefStr(action.Argument))

		case types.AgentActionTypeExecuteSQL:
			handleExecuteSQL(&toolHistory, emitter, plugin, config, derefStr(action.Argument), action.Requires_confirmation)

		case types.AgentActionTypeFinalAnswer:
			handleFinalAnswer(emitter, plugin, config, derefStr(action.Message), action.Sql)
			return nil

		default:
			appendToolResult(&toolHistory, string(action.Action), "Error: unknown action type")
		}
	}

	emitter.SendMessage(&model.AIChatMessage{
		Type: "message",
		Text: "I've reached the maximum number of steps. Here's what I found so far based on my exploration.",
	})
	emitter.SendDone()
	return nil
}

// --- action handlers ---

func handleListTables(history *strings.Builder, plugin *engine.Plugin, config *engine.PluginConfig, schema string) {
	units, err := plugin.GetStorageUnits(config, schema)
	if err != nil {
		appendToolResult(history, "list_tables", "Error: "+err.Error())
		return
	}
	names := make([]string, len(units))
	for i, u := range units {
		names[i] = u.Name
	}
	appendToolResult(history, "list_tables", "Tables: "+strings.Join(names, ", "))
}

func handleDescribeTable(history *strings.Builder, plugin *engine.Plugin, config *engine.PluginConfig, schema, tableName string) {
	if tableName == "" {
		appendToolResult(history, "describe_table", "Error: no table name provided")
		return
	}
	columns, err := plugin.GetColumnsForTable(config, schema, tableName)
	if err != nil {
		appendToolResult(history, "describe_table("+tableName+")", "Error: "+err.Error())
		return
	}
	descs := make([]string, len(columns))
	for i, col := range columns {
		d := fmt.Sprintf("%s (%s)", col.Name, col.Type)
		if col.IsPrimary {
			d += " PK"
		}
		if col.IsForeignKey {
			d += fmt.Sprintf(" FK->%s.%s", derefStr(col.ReferencedTable), derefStr(col.ReferencedColumn))
		}
		descs[i] = d
	}
	appendToolResult(history, "describe_table("+tableName+")", "Columns: "+strings.Join(descs, ", "))
}

func handleShowRelationships(history *strings.Builder, plugin *engine.Plugin, config *engine.PluginConfig, schema, tableName string) {
	if tableName == "" {
		appendToolResult(history, "show_relationships", "Error: no table name provided")
		return
	}
	rels, err := plugin.GetForeignKeyRelationships(config, schema, tableName)
	if err != nil {
		appendToolResult(history, "show_relationships("+tableName+")", "Error: "+err.Error())
		return
	}
	if len(rels) == 0 {
		appendToolResult(history, "show_relationships("+tableName+")", "No foreign key relationships found")
		return
	}
	descs := make([]string, 0, len(rels))
	for _, rel := range rels {
		descs = append(descs, fmt.Sprintf("%s -> %s.%s", rel.ColumnName, rel.ReferencedTable, rel.ReferencedColumn))
	}
	appendToolResult(history, "show_relationships("+tableName+")", strings.Join(descs, "; "))
}

func handleExecuteSQL(history *strings.Builder, emitter Emitter, plugin *engine.Plugin, config *engine.PluginConfig, sql string, requiresConfirmation bool) {
	if sql == "" {
		appendToolResult(history, "execute_sql", "Error: no SQL query provided")
		return
	}
	if requiresConfirmation {
		emitter.SendMessage(&model.AIChatMessage{
			Type:                 "sql:mutation",
			Text:                 sql,
			RequiresConfirmation: true,
		})
		appendToolResult(history, "execute_sql", "Mutation sent to user for confirmation (not executed)")
		return
	}
	result, err := plugin.RawExecute(config, sql)
	if err != nil {
		appendToolResult(history, "execute_sql", "Error: "+err.Error())
		return
	}
	emitter.SendMessage(&model.AIChatMessage{
		Type:   "sql:get",
		Text:   sql,
		Result: toModelResult(result),
	})
	appendToolResult(history, "execute_sql", formatQueryResult(sql, result))
}

func handleFinalAnswer(emitter Emitter, plugin *engine.Plugin, config *engine.PluginConfig, message string, sql *string) {
	emitter.SendMessage(&model.AIChatMessage{Type: "message", Text: message})

	if sql != nil && *sql != "" {
		result, err := plugin.RawExecute(config, *sql)
		if err != nil {
			emitter.SendMessage(&model.AIChatMessage{Type: "error", Text: err.Error()})
		} else {
			emitter.SendMessage(&model.AIChatMessage{
				Type:   "sql:get",
				Text:   *sql,
				Result: toModelResult(result),
			})
		}
	}

	emitter.SendDone()
}

// --- helpers ---

func appendToolResult(history *strings.Builder, action, result string) {
	if len(result) > maxToolResultChars {
		result = result[:maxToolResultChars] + "... (truncated)"
	}
	fmt.Fprintf(history, "\n[%s] %s\n", action, result)
}

func formatQueryResult(sql string, result *engine.GetRowsResult) string {
	if result == nil {
		return "Query executed: " + sql + "\nNo results returned."
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Query: %s\nRows: %d\n", sql, len(result.Rows))
	if len(result.Columns) > 0 {
		names := make([]string, len(result.Columns))
		for i, c := range result.Columns {
			names[i] = c.Name
		}
		fmt.Fprintf(&sb, "Columns: %s\n", strings.Join(names, ", "))
	}
	for i := 0; i < len(result.Rows) && i < 5; i++ {
		fmt.Fprintf(&sb, "  %s\n", strings.Join(result.Rows[i], " | "))
	}
	if len(result.Rows) > 5 {
		fmt.Fprintf(&sb, "  ... and %d more rows\n", len(result.Rows)-5)
	}
	return sb.String()
}

func toModelResult(result *engine.GetRowsResult) *model.RowsResult {
	if result == nil {
		return nil
	}
	columns := make([]*model.Column, len(result.Columns))
	for i, col := range result.Columns {
		columns[i] = &model.Column{Type: col.Type, Name: col.Name}
	}
	return &model.RowsResult{Columns: columns, Rows: result.Rows}
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
