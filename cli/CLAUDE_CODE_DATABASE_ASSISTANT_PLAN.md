# Claude Code-like Database Assistant for WhoDB CLI

## Executive Summary

This document outlines a comprehensive plan to implement a Claude Code-like experience within the WhoDB CLI, specifically optimized for database operations using a small local LLM (liquid/lfm2-1.2b) via LM Studio.

## Architecture Overview

### Core Philosophy
- **Database-First Design**: Every feature is optimized for database operations
- **Small Model Optimization**: Designed to work efficiently with a 1.2B parameter model
- **Tool-Based Approach**: Decompose complex operations into discrete, executable tools
- **Progressive Enhancement**: Gracefully handle model limitations with fallbacks
- **Context Awareness**: Maintain rich database context for intelligent suggestions

## 1. Enhanced Tool System for Database Operations

### Tool Interface Architecture

```go
// internal/ai/tools/interface.go
package tools

import (
    "github.com/clidey/whodb/cli/internal/database"
    "github.com/clidey/whodb/core/src/engine"
)

type DatabaseTool interface {
    // Metadata
    Name() string                      // e.g., "query_execute"
    Description() string                // Human-readable description
    Category() ToolCategory            // Organization category

    // Schema definition for parameters
    Parameters() []ParameterDef
    RequiredParams() []string

    // Execution
    Execute(params map[string]any, ctx *DatabaseContext) (ToolResult, error)
    ValidateParams(params map[string]any) error

    // Cost estimation
    EstimateCost() ResourceCost        // CPU, memory, time estimates
    RequiresConnection() bool           // Does this need active DB connection?
}

type ParameterDef struct {
    Name        string
    Type        ParamType              // string, int, bool, array, object
    Description string
    Required    bool
    Default     any
    Validators  []ParamValidator
}

type ToolResult struct {
    Success     bool
    Data        any                    // The actual result data
    Markdown    string                 // Formatted for display
    NextTools   []string              // Suggested follow-up tools
    Error       error
    Timing      time.Duration
}

type DatabaseContext struct {
    Manager         *database.Manager
    Schema          *SchemaGraph
    QueryHistory    *QueryHistoryBuffer
    Statistics      *TableStatistics
    WorkingSet      *WorkingSetTracker
    UserPreferences *UserPrefs
    SafeMode        bool
}

type ToolCategory string
const (
    CategoryQuery      ToolCategory = "query"
    CategorySchema     ToolCategory = "schema"
    CategoryAnalysis   ToolCategory = "analysis"
    CategoryMigration  ToolCategory = "migration"
    CategoryImportExport ToolCategory = "import_export"
    CategoryVisualization ToolCategory = "visualization"
)
```

### Core Database Tools Implementation

#### 1. Query Execution Tool
```go
// internal/ai/tools/query_tool.go
type QueryTool struct {
    maxRows      int
    timeout      time.Duration
    explainFirst bool
}

func (t *QueryTool) Execute(params map[string]any, ctx *DatabaseContext) (ToolResult, error) {
    query := params["query"].(string)

    // Safety checks
    if ctx.SafeMode && isDestructive(query) {
        return ToolResult{}, fmt.Errorf("destructive query blocked in safe mode")
    }

    // Optional explain first
    if t.explainFirst {
        plan, _ := ctx.Manager.ExplainQuery(query)
        if costTooHigh(plan) {
            return ToolResult{
                Success: false,
                Error: fmt.Errorf("query cost too high: %v", plan.Cost),
            }, nil
        }
    }

    // Execute with timing
    start := time.Now()
    result, err := ctx.Manager.ExecuteQuery(query)
    duration := time.Since(start)

    // Track in history
    ctx.QueryHistory.Add(QueryExecution{
        Query:    query,
        Duration: duration,
        RowCount: len(result.Rows),
        Error:    err,
    })

    return ToolResult{
        Success:  err == nil,
        Data:     result,
        Markdown: formatQueryResult(result, duration),
        NextTools: suggestNextTools(result),
        Timing:   duration,
        Error:    err,
    }, nil
}
```

#### 2. Schema Inspector Tool
```go
// internal/ai/tools/schema_inspector.go
type SchemaInspectorTool struct{}

func (t *SchemaInspectorTool) Execute(params map[string]any, ctx *DatabaseContext) (ToolResult, error) {
    tableName := params["table"].(string)
    detailed := params["detailed"].(bool)

    // Get basic structure
    columns, _ := ctx.Manager.GetColumns(ctx.Schema.CurrentSchema, tableName)

    // Get additional metadata if detailed
    var constraints []Constraint
    var indexes []Index
    var triggers []Trigger
    var foreignKeys []ForeignKey

    if detailed {
        constraints = ctx.Manager.GetConstraints(tableName)
        indexes = ctx.Manager.GetIndexes(tableName)
        triggers = ctx.Manager.GetTriggers(tableName)
        foreignKeys = ctx.Manager.GetForeignKeys(tableName)
    }

    result := SchemaInfo{
        TableName:   tableName,
        Columns:     columns,
        Constraints: constraints,
        Indexes:     indexes,
        Triggers:    triggers,
        ForeignKeys: foreignKeys,
    }

    return ToolResult{
        Success:  true,
        Data:     result,
        Markdown: formatSchemaInfo(result),
        NextTools: []string{"data_profile", "index_advisor"},
    }, nil
}
```

#### 3. Query Explain Tool
```go
// internal/ai/tools/explain_tool.go
type ExplainTool struct {
    analyzeBuffers bool
    verbose        bool
}

func (t *ExplainTool) Execute(params map[string]any, ctx *DatabaseContext) (ToolResult, error) {
    query := params["query"].(string)

    // Build explain query based on database type
    explainQuery := buildExplainQuery(ctx.Manager.GetDBType(), query, t.analyzeBuffers, t.verbose)

    result, err := ctx.Manager.ExecuteQuery(explainQuery)
    if err != nil {
        return ToolResult{Error: err}, err
    }

    // Parse explain output
    plan := parseExplainOutput(result, ctx.Manager.GetDBType())

    // Analyze for issues
    issues := analyzeQueryPlan(plan)

    return ToolResult{
        Success: true,
        Data: ExplainResult{
            Plan:   plan,
            Issues: issues,
            Cost:   plan.TotalCost,
        },
        Markdown: formatExplainPlan(plan, issues),
        NextTools: []string{"index_advisor", "query_optimizer"},
    }, nil
}
```

#### 4. Data Profiling Tool
```go
// internal/ai/tools/data_profiler.go
type DataProfilerTool struct {
    sampleSize int
}

func (t *DataProfilerTool) Execute(params map[string]any, ctx *DatabaseContext) (ToolResult, error) {
    table := params["table"].(string)
    columns := params["columns"].([]string)

    profile := TableProfile{
        TableName: table,
        RowCount:  0,
        Columns:   make([]ColumnProfile, 0),
    }

    // Get row count
    countResult, _ := ctx.Manager.ExecuteQuery(fmt.Sprintf(
        "SELECT COUNT(*) FROM %s",
        ctx.Manager.EscapeIdentifier(table),
    ))
    profile.RowCount = extractCount(countResult)

    // Profile each column
    for _, col := range columns {
        colProfile := ColumnProfile{
            Name: col,
        }

        // Get statistics
        stats := getColumnStatistics(ctx, table, col, t.sampleSize)
        colProfile.Stats = stats

        profile.Columns = append(profile.Columns, colProfile)
    }

    return ToolResult{
        Success:  true,
        Data:     profile,
        Markdown: formatDataProfile(profile),
        NextTools: []string{"index_advisor", "data_cleaner"},
    }, nil
}
```

#### 5. Index Advisor Tool
```go
// internal/ai/tools/index_advisor.go
type IndexAdvisorTool struct {
    queryHistory []QueryExecution
    threshold    float64
}

func (t *IndexAdvisorTool) Execute(params map[string]any, ctx *DatabaseContext) (ToolResult, error) {
    table := params["table"].(string)

    // Analyze recent queries
    relevantQueries := filterQueriesForTable(ctx.QueryHistory.Recent(100), table)

    // Get current indexes
    currentIndexes, _ := ctx.Manager.GetIndexes(table)

    // Analyze missing indexes
    suggestions := []IndexSuggestion{}

    for _, query := range relevantQueries {
        // Parse WHERE, JOIN, ORDER BY clauses
        clauses := parseQueryClauses(query.Query)

        // Check if existing indexes cover the query
        if !isQueryCovered(clauses, currentIndexes) {
            suggestion := generateIndexSuggestion(clauses, table)
            suggestions = append(suggestions, suggestion)
        }
    }

    // Rank suggestions by impact
    rankedSuggestions := rankIndexSuggestions(suggestions, relevantQueries)

    return ToolResult{
        Success:  true,
        Data:     rankedSuggestions,
        Markdown: formatIndexSuggestions(rankedSuggestions),
        NextTools: []string{"migration_tool"},
    }, nil
}
```

#### 6. Migration Generator Tool
```go
// internal/ai/tools/migration_tool.go
type MigrationTool struct {
    dryRun bool
    format MigrationFormat // SQL, JSON, YAML
}

func (t *MigrationTool) Execute(params map[string]any, ctx *DatabaseContext) (ToolResult, error) {
    changeType := params["type"].(string) // "add_index", "add_column", etc.
    target := params["target"].(map[string]any)

    migration := Migration{
        ID:        generateMigrationID(),
        Type:      changeType,
        Timestamp: time.Now(),
    }

    // Generate forward migration
    upSQL, err := generateUpMigration(changeType, target, ctx)
    if err != nil {
        return ToolResult{Error: err}, err
    }
    migration.Up = upSQL

    // Generate rollback
    downSQL, err := generateDownMigration(changeType, target, ctx)
    migration.Down = downSQL

    // Validate migration
    if !t.dryRun {
        // Test in transaction
        testErr := testMigrationInTransaction(ctx, migration)
        if testErr != nil {
            migration.ValidationError = testErr
        }
    }

    return ToolResult{
        Success:  true,
        Data:     migration,
        Markdown: formatMigration(migration),
        NextTools: []string{"backup_tool"},
    }, nil
}
```

#### 7. Data Import/Export Tool
```go
// internal/ai/tools/import_export_tool.go
type ImportExportTool struct {
    format     DataFormat // CSV, JSON, Excel, SQL
    batchSize  int
}

func (t *ImportExportTool) Execute(params map[string]any, ctx *DatabaseContext) (ToolResult, error) {
    operation := params["operation"].(string) // "import" or "export"

    switch operation {
    case "export":
        return t.executeExport(params, ctx)
    case "import":
        return t.executeImport(params, ctx)
    default:
        return ToolResult{Error: fmt.Errorf("unknown operation: %s", operation)}, nil
    }
}
```

#### 8. Visualization Tool
```go
// internal/ai/tools/visualization_tool.go
type VisualizationTool struct {
    renderer ChartRenderer
}

func (t *VisualizationTool) Execute(params map[string]any, ctx *DatabaseContext) (ToolResult, error) {
    chartType := params["type"].(string) // "bar", "line", "distribution", "heatmap"
    data := params["data"].([][]any)

    chart := t.renderer.Render(chartType, data, params["options"].(map[string]any))

    return ToolResult{
        Success:  true,
        Data:     chart,
        Markdown: chart.ASCII(), // ASCII art representation
        NextTools: []string{"export_tool"},
    }, nil
}
```

## 2. LM Studio Integration

### Provider Implementation

```go
// internal/ai/providers/lmstudio.go
package providers

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type LMStudioProvider struct {
    BaseURL       string
    Model         string
    MaxTokens     int
    Temperature   float32
    Stream        bool
    Timeout       time.Duration

    // Optimization for small models
    PromptCache   *PromptCache
    TokenCounter  *TokenCounter
    SystemPrompt  string
}

func NewLMStudioProvider(config LMStudioConfig) *LMStudioProvider {
    return &LMStudioProvider{
        BaseURL:     config.BaseURL, // Default: "http://localhost:1234/v1"
        Model:       config.Model,    // Default: "liquid/lfm2-1.2b"
        MaxTokens:   2048,
        Temperature: 0.3,             // Lower for more deterministic database operations
        Stream:      true,
        Timeout:     30 * time.Second,
        PromptCache: NewPromptCache(1000),
        TokenCounter: NewTokenCounter(config.Model),
        SystemPrompt: getDatabaseSystemPrompt(),
    }
}

func (p *LMStudioProvider) Complete(request CompletionRequest) (*CompletionResponse, error) {
    // Token optimization for small model
    optimizedPrompt := p.optimizePrompt(request.Prompt)

    // Check cache
    if cached, found := p.PromptCache.Get(optimizedPrompt); found {
        return cached, nil
    }

    // Build request
    apiRequest := LMStudioRequest{
        Model:       p.Model,
        Messages:    p.buildMessages(optimizedPrompt),
        MaxTokens:   p.MaxTokens,
        Temperature: p.Temperature,
        Stream:      p.Stream,
    }

    // Make API call
    resp, err := p.callAPI(apiRequest)
    if err != nil {
        return nil, err
    }

    // Cache successful responses
    p.PromptCache.Set(optimizedPrompt, resp)

    return resp, nil
}

func (p *LMStudioProvider) optimizePrompt(prompt string) string {
    // Compress for small model efficiency
    compressed := CompressPrompt(prompt, p.TokenCounter.Limit())

    // Add database-specific context
    withContext := fmt.Sprintf("%s\n\n%s", p.SystemPrompt, compressed)

    return withContext
}

func (p *LMStudioProvider) StreamComplete(request CompletionRequest) (<-chan StreamChunk, error) {
    chunks := make(chan StreamChunk, 100)

    go func() {
        defer close(chunks)

        // Similar to Complete but with streaming
        apiRequest := LMStudioRequest{
            Model:  p.Model,
            Messages: p.buildMessages(request.Prompt),
            Stream: true,
        }

        resp, err := p.streamAPI(apiRequest)
        if err != nil {
            chunks <- StreamChunk{Error: err}
            return
        }

        // Parse SSE stream
        p.parseSSEStream(resp.Body, chunks)
    }()

    return chunks, nil
}
```

### Prompt Optimization for Small Models

```go
// internal/ai/optimization/prompt_optimizer.go
package optimization

type PromptOptimizer struct {
    abbreviations map[string]string
    schemaCache   *SchemaCache
    maxTokens     int
}

func (o *PromptOptimizer) Optimize(prompt string, context DatabaseContext) string {
    // 1. Apply abbreviations
    optimized := o.applyAbbreviations(prompt)

    // 2. Compress schema references
    optimized = o.compressSchemaRefs(optimized, context)

    // 3. Remove redundant information
    optimized = o.removeRedundancy(optimized)

    // 4. Truncate if needed
    if o.countTokens(optimized) > o.maxTokens {
        optimized = o.intelligentTruncate(optimized, o.maxTokens)
    }

    return optimized
}

func (o *PromptOptimizer) applyAbbreviations(text string) string {
    replacements := map[string]string{
        "SELECT":    "SEL",
        "FROM":      "FRM",
        "WHERE":     "WHR",
        "GROUP BY":  "GRP",
        "ORDER BY":  "ORD",
        "INNER JOIN": "IJ",
        "LEFT JOIN": "LJ",
        "database":  "db",
        "table":     "tbl",
        "column":    "col",
        "index":     "idx",
        "constraint": "cnst",
    }

    result := text
    for full, abbr := range replacements {
        result = strings.ReplaceAll(result, full, abbr)
    }
    return result
}

func (o *PromptOptimizer) compressSchemaRefs(text string, ctx DatabaseContext) string {
    // Only include tables mentioned in the prompt or working set
    relevantTables := extractTableNames(text)
    relevantTables = append(relevantTables, ctx.WorkingSet.Tables()...)

    // Build minimal schema description
    miniSchema := buildMinimalSchema(ctx.Schema, relevantTables)

    return strings.Replace(text, "{{SCHEMA}}", miniSchema, 1)
}
```

### Database-Specific System Prompts

```go
// internal/ai/prompts/system_prompts.go
func getDatabaseSystemPrompt() string {
    return `You are a database assistant. Rules:
1. Output valid SQL only
2. Use standard SQL syntax
3. Prefer JOINs over subqueries
4. Always use parameterized queries
5. Limit results to 1000 rows max
6. Format: {"sql": "...", "explanation": "..."}`
}

func getToolSystemPrompt() string {
    return `You are a tool-using database assistant.
Available tools: query_execute, schema_inspect, explain_query, data_profile, index_advisor
Format: {"tool": "name", "params": {...}}
Chain tools for complex tasks.`
}
```

## 3. Multi-Step Operation Planner

### Task Decomposition System

```go
// internal/ai/planner/task_planner.go
package planner

type TaskPlanner struct {
    registry    *ToolRegistry
    maxSteps    int
    parallelism int
}

type DatabaseTask struct {
    ID           string
    Description  string
    Status       TaskStatus
    Steps        []TaskStep
    Dependencies []string
    Result       any
    Error        error
    CreatedAt    time.Time
    CompletedAt  *time.Time
}

type TaskStep struct {
    ID          string
    Tool        string
    Parameters  map[string]any
    DependsOn   []string
    CanParallel bool
    Result      any
    Error       error
    StartedAt   *time.Time
    CompletedAt *time.Time
}

func (p *TaskPlanner) PlanTask(query string, context DatabaseContext) (*DatabaseTask, error) {
    // Parse user intent
    intent := p.parseIntent(query)

    // Generate task steps
    steps := p.generateSteps(intent, context)

    // Optimize execution order
    optimizedSteps := p.optimizeStepOrder(steps)

    // Create task
    task := &DatabaseTask{
        ID:          uuid.New().String(),
        Description: intent.Summary,
        Status:      TaskStatusPending,
        Steps:       optimizedSteps,
        CreatedAt:   time.Now(),
    }

    return task, nil
}

func (p *TaskPlanner) ExecuteTask(task *DatabaseTask, ctx DatabaseContext) error {
    task.Status = TaskStatusRunning

    // Create execution graph
    graph := p.buildExecutionGraph(task.Steps)

    // Execute steps
    executor := NewStepExecutor(p.registry, p.parallelism)

    for !graph.IsComplete() {
        // Get ready steps (no pending dependencies)
        readySteps := graph.GetReadySteps()

        // Execute in parallel if possible
        results := executor.ExecuteSteps(readySteps, ctx)

        // Update graph with results
        for stepID, result := range results {
            graph.MarkComplete(stepID, result)

            // Update task
            for i, step := range task.Steps {
                if step.ID == stepID {
                    task.Steps[i].Result = result.Data
                    task.Steps[i].Error = result.Error
                    now := time.Now()
                    task.Steps[i].CompletedAt = &now
                }
            }
        }
    }

    task.Status = TaskStatusCompleted
    now := time.Now()
    task.CompletedAt = &now

    return nil
}
```

### Example Task Flows

```go
// internal/ai/planner/task_templates.go

var taskTemplates = map[string]TaskTemplate{
    "optimize_slow_queries": {
        Description: "Identify and optimize slow queries",
        Steps: []StepTemplate{
            {Tool: "query_profiler", Params: map[string]any{"limit": 10}},
            {Tool: "explain_query", DependsOn: []string{"0"}, Parallel: true},
            {Tool: "data_profile", DependsOn: []string{"0"}, Parallel: true},
            {Tool: "index_advisor", DependsOn: []string{"1", "2"}},
            {Tool: "migration_tool", DependsOn: []string{"3"}},
        },
    },
    "analyze_table": {
        Description: "Comprehensive table analysis",
        Steps: []StepTemplate{
            {Tool: "schema_inspect", Params: map[string]any{"detailed": true}},
            {Tool: "data_profile", Parallel: true},
            {Tool: "index_analyzer", Parallel: true},
            {Tool: "relationship_mapper", Parallel: true},
            {Tool: "visualization", DependsOn: []string{"1"}},
        },
    },
}
```

## 4. Enhanced UI Components

### Split View Implementation

```go
// internal/tui/enhanced_chat_view.go
type EnhancedChatView struct {
    *ChatView                         // Embed existing chat

    // Split view components
    splitMode      bool
    splitPosition  float64           // 0.0-1.0 for split ratio
    resultPanel    *ResultPanel
    visualPanel    *VisualizationPanel

    // Task tracking
    activeTask     *DatabaseTask
    taskProgress   *ProgressBar

    // Tool palette
    toolPalette    *ToolPalette
    paletteVisible bool

    // Context display
    contextPanel   *ContextPanel
    showContext    bool
}

func (v *EnhancedChatView) Update(msg tea.Msg) (*EnhancedChatView, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+k":
            v.paletteVisible = !v.paletteVisible
            return v, nil

        case "ctrl+p":
            if v.activeTask != nil {
                v.showTaskProgress()
            }
            return v, nil

        case "ctrl+/":
            v.showContext = !v.showContext
            return v, nil

        case "ctrl+r":
            if v.lastOperation != nil {
                return v, v.rerunOperation()
            }

        case "ctrl+z":
            if v.canUndo() {
                return v, v.undoLastChange()
            }
        }

    case toolSelectedMsg:
        return v, v.executeTool(msg.Tool, msg.Params)

    case taskProgressMsg:
        v.updateTaskProgress(msg)
        return v, nil
    }

    // Delegate to embedded ChatView
    return v, v.ChatView.Update(msg)
}

func (v *EnhancedChatView) View() string {
    if v.splitMode {
        return v.renderSplitView()
    }
    return v.renderNormalView()
}

func (v *EnhancedChatView) renderSplitView() string {
    leftWidth := int(float64(v.width) * v.splitPosition)
    rightWidth := v.width - leftWidth - 1

    // Left panel: Chat
    leftPanel := lipgloss.NewStyle().
        Width(leftWidth).
        Height(v.height).
        Render(v.ChatView.View())

    // Right panel: Results/Visualizations
    rightContent := v.resultPanel.View()
    if v.visualPanel.HasContent() {
        rightContent = lipgloss.JoinVertical(
            lipgloss.Center,
            v.visualPanel.View(),
            rightContent,
        )
    }

    rightPanel := lipgloss.NewStyle().
        Width(rightWidth).
        Height(v.height).
        Render(rightContent)

    // Join panels
    content := lipgloss.JoinHorizontal(
        lipgloss.Top,
        leftPanel,
        lipgloss.NewStyle().
            Width(1).
            Height(v.height).
            Background(lipgloss.Color("240")).
            Render(" "),
        rightPanel,
    )

    // Add status bar
    statusBar := v.renderStatusBar()

    return lipgloss.JoinVertical(
        lipgloss.Left,
        content,
        statusBar,
    )
}
```

### Tool Palette Component

```go
// internal/tui/tool_palette.go
type ToolPalette struct {
    tools        []DatabaseTool
    filtered     []DatabaseTool
    selected     int
    searchInput  textinput.Model
    visible      bool

    // Categories
    categories   []ToolCategory
    activeCategory ToolCategory
}

func (p *ToolPalette) View() string {
    if !p.visible {
        return ""
    }

    // Search box
    searchBox := p.searchInput.View()

    // Category tabs
    tabs := p.renderCategoryTabs()

    // Tool list
    toolList := p.renderToolList()

    // Keyboard hints
    hints := p.renderKeyboardHints()

    return styles.DialogBox.Render(
        lipgloss.JoinVertical(
            lipgloss.Left,
            "🔧 Tool Palette",
            searchBox,
            tabs,
            toolList,
            hints,
        ),
    )
}

func (p *ToolPalette) renderToolList() string {
    var items []string

    for i, tool := range p.filtered {
        icon := p.getToolIcon(tool)
        name := tool.Name()
        desc := tool.Description()

        style := styles.ListItem
        if i == p.selected {
            style = styles.ActiveListItem
        }

        item := fmt.Sprintf("%s %s - %s", icon, name, desc)
        items = append(items, style.Render(item))
    }

    return lipgloss.JoinVertical(lipgloss.Left, items...)
}
```

## 5. Database-Specific Agents

### Agent System Architecture

```go
// internal/ai/agents/base.go
package agents

type Agent interface {
    Name() string
    Description() string
    CanHandle(query string) bool
    Priority() int
    Execute(query string, ctx *DatabaseContext) (AgentResponse, error)
}

type AgentResponse struct {
    Success     bool
    Message     string
    Actions     []Action
    Suggestions []string
    Data        any
}

type Action struct {
    Type       ActionType
    Tool       string
    Parameters map[string]any
    Result     any
}

type AgentRegistry struct {
    agents []Agent
}

func (r *AgentRegistry) SelectAgent(query string) Agent {
    var candidates []Agent

    for _, agent := range r.agents {
        if agent.CanHandle(query) {
            candidates = append(candidates, agent)
        }
    }

    // Sort by priority
    sort.Slice(candidates, func(i, j int) bool {
        return candidates[i].Priority() > candidates[j].Priority()
    })

    if len(candidates) > 0 {
        return candidates[0]
    }

    return r.defaultAgent
}
```

### Specialized Agent Implementations

#### Query Optimizer Agent
```go
// internal/ai/agents/query_optimizer.go
type QueryOptimizerAgent struct {
    analyzer     *QueryAnalyzer
    optimizer    *QueryOptimizer
    benchmarker  *QueryBenchmarker
}

func (a *QueryOptimizerAgent) CanHandle(query string) bool {
    keywords := []string{"optimize", "slow", "performance", "speed up", "improve query"}
    return containsAny(strings.ToLower(query), keywords)
}

func (a *QueryOptimizerAgent) Execute(query string, ctx *DatabaseContext) (AgentResponse, error) {
    // Extract SQL query to optimize
    sqlQuery := extractSQLFromQuery(query)

    // Analyze current performance
    analysis := a.analyzer.Analyze(sqlQuery, ctx)

    // Generate optimizations
    optimizations := a.optimizer.Optimize(sqlQuery, analysis, ctx)

    // Benchmark improvements
    benchmarks := a.benchmarker.Compare(sqlQuery, optimizations, ctx)

    return AgentResponse{
        Success: true,
        Message: formatOptimizationResults(optimizations, benchmarks),
        Actions: buildOptimizationActions(optimizations),
        Suggestions: generateNextSteps(analysis),
        Data: OptimizationResult{
            Original:      sqlQuery,
            Optimizations: optimizations,
            Benchmarks:    benchmarks,
        },
    }, nil
}
```

#### Data Analyst Agent
```go
// internal/ai/agents/data_analyst.go
type DataAnalystAgent struct {
    profiler    *DataProfiler
    statistician *Statistician
    visualizer  *DataVisualizer
}

func (a *DataAnalystAgent) Execute(query string, ctx *DatabaseContext) (AgentResponse, error) {
    // Determine analysis type
    analysisType := detectAnalysisType(query)

    switch analysisType {
    case "distribution":
        return a.analyzeDistribution(query, ctx)
    case "correlation":
        return a.analyzeCorrelation(query, ctx)
    case "trends":
        return a.analyzeTrends(query, ctx)
    case "outliers":
        return a.detectOutliers(query, ctx)
    default:
        return a.generalAnalysis(query, ctx)
    }
}
```

#### Schema Designer Agent
```go
// internal/ai/agents/schema_designer.go
type SchemaDesignerAgent struct {
    normalizer   *SchemaNormalizer
    validator    *SchemaValidator
    recommender  *DesignRecommender
}

func (a *SchemaDesignerAgent) Execute(query string, ctx *DatabaseContext) (AgentResponse, error) {
    // Parse design request
    request := parseDesignRequest(query)

    // Generate schema design
    design := a.generateDesign(request, ctx)

    // Validate design
    issues := a.validator.Validate(design)

    // Generate recommendations
    recommendations := a.recommender.Recommend(design, ctx)

    // Create DDL statements
    ddl := generateDDL(design, ctx.Manager.GetDBType())

    return AgentResponse{
        Success: true,
        Message: formatDesignResponse(design, recommendations),
        Actions: []Action{
            {
                Type: ActionTypeMigration,
                Tool: "migration_tool",
                Parameters: map[string]any{
                    "ddl": ddl,
                    "design": design,
                },
            },
        },
        Data: design,
    }, nil
}
```

## 6. Smart Query Building

### Natural Language to SQL System

```go
// internal/ai/nl2sql/query_builder.go
package nl2sql

type SmartQueryBuilder struct {
    context      *DatabaseContext
    dialect      SQLDialect
    safeMode     bool
    fuzzyMatcher *FuzzyMatcher
    joinDetector *JoinDetector
    templates    *QueryTemplates
}

type QueryIntent struct {
    Action      QueryAction    // SELECT, INSERT, UPDATE, DELETE
    Tables      []TableRef
    Columns     []ColumnRef
    Conditions  []Condition
    Joins       []JoinClause
    GroupBy     []ColumnRef
    OrderBy     []OrderClause
    Limit       *int
}

func (b *SmartQueryBuilder) BuildQuery(naturalLanguage string) (string, error) {
    // Parse intent from natural language
    intent := b.parseIntent(naturalLanguage)

    // Resolve table and column names with fuzzy matching
    intent = b.resolveFuzzyRefs(intent)

    // Detect required joins from relationships
    intent = b.detectJoins(intent)

    // Apply safety checks
    if b.safeMode {
        intent = b.applySafetyLimits(intent)
    }

    // Build SQL based on dialect
    sql := b.buildSQL(intent)

    // Validate generated SQL
    if err := b.validateSQL(sql); err != nil {
        return "", fmt.Errorf("invalid SQL generated: %w", err)
    }

    return sql, nil
}

func (b *SmartQueryBuilder) resolveFuzzyRefs(intent QueryIntent) QueryIntent {
    resolved := intent

    // Fuzzy match table names
    for i, tableRef := range intent.Tables {
        if matched := b.fuzzyMatcher.MatchTable(tableRef.Name); matched != "" {
            resolved.Tables[i].Name = matched
        }
    }

    // Fuzzy match column names
    for i, colRef := range intent.Columns {
        if matched := b.fuzzyMatcher.MatchColumn(colRef.Name, colRef.Table); matched != "" {
            resolved.Columns[i].Name = matched
        }
    }

    return resolved
}

func (b *SmartQueryBuilder) detectJoins(intent QueryIntent) QueryIntent {
    if len(intent.Tables) <= 1 {
        return intent
    }

    // Find foreign key relationships
    relationships := b.context.Schema.FindRelationships(intent.Tables)

    // Generate optimal join path
    joinPath := b.joinDetector.FindOptimalPath(relationships)

    // Add joins to intent
    for _, rel := range joinPath {
        intent.Joins = append(intent.Joins, JoinClause{
            Type:      "INNER",
            Table:     rel.TargetTable,
            Condition: fmt.Sprintf("%s.%s = %s.%s",
                rel.SourceTable, rel.SourceColumn,
                rel.TargetTable, rel.TargetColumn,
            ),
        })
    }

    return intent
}
```

### Query Template System

```go
// internal/ai/nl2sql/templates.go
type QueryTemplates struct {
    templates map[string]*QueryTemplate
}

type QueryTemplate struct {
    Name        string
    Pattern     string
    SQLTemplate string
    Variables   []Variable
}

var commonTemplates = []QueryTemplate{
    {
        Name:    "top_n",
        Pattern: "top (\\d+) (.+) by (.+)",
        SQLTemplate: `SELECT {{columns}}
FROM {{table}}
ORDER BY {{order_column}} DESC
LIMIT {{n}}`,
    },
    {
        Name:    "recent_records",
        Pattern: "recent (.+) from (.+)",
        SQLTemplate: `SELECT *
FROM {{table}}
WHERE {{date_column}} >= NOW() - INTERVAL '{{interval}}'
ORDER BY {{date_column}} DESC`,
    },
    {
        Name:    "count_by_group",
        Pattern: "count (.+) by (.+)",
        SQLTemplate: `SELECT {{group_column}}, COUNT(*) as count
FROM {{table}}
GROUP BY {{group_column}}
ORDER BY count DESC`,
    },
}
```

## 7. Result Intelligence

### Smart Result Handler

```go
// internal/tui/smart_results.go
type SmartResultsView struct {
    *ResultsView  // Embed existing

    // Intelligence features
    analyzer        *ResultAnalyzer
    formatter       *SmartFormatter
    navigator       *RelationshipNavigator
    exporter        *SmartExporter

    // UI enhancements
    outliers        []Outlier
    relationships   []Relationship
    exportSuggestion *ExportSuggestion
}

func (v *SmartResultsView) analyzeResults(results *engine.GetRowsResult) {
    // Detect data types
    v.detectDataTypes(results)

    // Find outliers
    v.outliers = v.analyzer.FindOutliers(results)

    // Detect relationships
    v.relationships = v.navigator.DetectRelationships(results)

    // Suggest export format
    v.exportSuggestion = v.exporter.SuggestFormat(results)
}

func (v *SmartResultsView) View() string {
    base := v.ResultsView.View()

    // Add intelligence overlay
    if len(v.outliers) > 0 {
        base = v.highlightOutliers(base)
    }

    // Add relationship indicators
    if len(v.relationships) > 0 {
        base = v.addRelationshipLinks(base)
    }

    // Add export suggestion
    if v.exportSuggestion != nil {
        base = lipgloss.JoinVertical(
            lipgloss.Left,
            base,
            v.renderExportSuggestion(),
        )
    }

    return base
}
```

## 8. Learning System

### Pattern Recognition Engine

```go
// internal/ai/learning/pattern_engine.go
package learning

type PatternLearner struct {
    store        *PatternStore
    analyzer     *PatternAnalyzer
    userProfile  *UserProfile
}

type Pattern struct {
    ID          string
    Type        PatternType
    Frequency   int
    LastUsed    time.Time
    Context     map[string]any
    Success     bool
}

type UserProfile struct {
    ID               string
    NamingConvention NamingStyle
    CommonFilters    []Filter
    FrequentJoins    []JoinPattern
    QueryStyle       QueryStyle
    Shortcuts        map[string]string
}

func (l *PatternLearner) Learn(execution QueryExecution) {
    // Extract patterns
    patterns := l.analyzer.ExtractPatterns(execution)

    // Store patterns
    for _, pattern := range patterns {
        l.store.Save(pattern)
    }

    // Update user profile
    l.updateProfile(patterns)

    // Learn from mistakes
    if execution.Error != nil {
        l.learnFromError(execution)
    }
}

func (l *PatternLearner) Suggest(context QueryContext) []Suggestion {
    // Get relevant patterns
    patterns := l.store.GetRelevant(context)

    // Generate suggestions
    suggestions := []Suggestion{}

    for _, pattern := range patterns {
        if pattern.Frequency > 3 && pattern.Success {
            suggestion := l.generateSuggestion(pattern, context)
            suggestions = append(suggestions, suggestion)
        }
    }

    // Sort by relevance
    sort.Slice(suggestions, func(i, j int) bool {
        return suggestions[i].Relevance > suggestions[j].Relevance
    })

    return suggestions[:min(5, len(suggestions))]
}
```

### Mistake Correction System

```go
// internal/ai/learning/correction_system.go
type CorrectionSystem struct {
    mistakes    map[string]Correction
    classifier  *ErrorClassifier
}

type Correction struct {
    ErrorPattern string
    Correction   string
    Explanation  string
    Frequency    int
}

func (s *CorrectionSystem) SuggestCorrection(error error, query string) *Correction {
    // Classify error
    errorType := s.classifier.Classify(error)

    // Find matching correction
    for pattern, correction := range s.mistakes {
        if matches(error.Error(), pattern) {
            return &correction
        }
    }

    // Generate new correction if possible
    if suggestion := s.generateCorrection(errorType, query); suggestion != nil {
        s.mistakes[error.Error()] = *suggestion
        return suggestion
    }

    return nil
}
```

## 9. Safety and Governance

### Safety System Implementation

```go
// internal/ai/safety/safety_manager.go
package safety

type SafetyManager struct {
    config       SafetyConfig
    auditor      *Auditor
    validator    *QueryValidator
    rollback     *RollbackManager
}

type SafetyConfig struct {
    EnableTransactions bool
    MaxRowsAffected    int
    RequireWhereClause bool
    BlockTruncate      bool
    BlockDrops         bool
    DryRunDDL          bool
    AuditLog           bool
}

func (m *SafetyManager) WrapExecution(f func() error) error {
    // Start transaction
    tx := m.beginTransaction()
    defer tx.Rollback()

    // Create savepoint
    savepoint := tx.CreateSavepoint()

    // Execute with monitoring
    err := f()

    if err != nil {
        // Rollback to savepoint
        tx.RollbackTo(savepoint)
        m.auditor.LogFailure(err)
        return err
    }

    // Check impact
    impact := tx.GetImpact()
    if impact.RowsAffected > m.config.MaxRowsAffected {
        tx.RollbackTo(savepoint)
        return fmt.Errorf("too many rows affected: %d", impact.RowsAffected)
    }

    // Commit if safe
    tx.Commit()
    m.auditor.LogSuccess(impact)

    return nil
}

func (m *SafetyManager) ValidateQuery(query string) error {
    // Parse query
    parsed, err := m.validator.Parse(query)
    if err != nil {
        return err
    }

    // Check for destructive operations
    if m.config.BlockTruncate && parsed.IsTruncate() {
        return fmt.Errorf("TRUNCATE operations are blocked")
    }

    if m.config.BlockDrops && parsed.IsDrop() {
        return fmt.Errorf("DROP operations are blocked")
    }

    // Require WHERE clause for UPDATE/DELETE
    if m.config.RequireWhereClause {
        if (parsed.IsUpdate() || parsed.IsDelete()) && !parsed.HasWhere() {
            return fmt.Errorf("UPDATE/DELETE requires WHERE clause")
        }
    }

    return nil
}
```

### Audit System

```go
// internal/ai/safety/audit.go
type Auditor struct {
    storage AuditStorage
    encoder *AuditEncoder
}

type AuditEntry struct {
    ID          string
    Timestamp   time.Time
    User        string
    Action      string
    Query       string
    Result      string
    Impact      Impact
    Error       error
    Duration    time.Duration
    Context     map[string]any
}

func (a *Auditor) LogOperation(op Operation) {
    entry := AuditEntry{
        ID:        uuid.New().String(),
        Timestamp: time.Now(),
        User:      op.User,
        Action:    op.Type,
        Query:     op.Query,
        Result:    op.Result,
        Impact:    op.Impact,
        Error:     op.Error,
        Duration:  op.Duration,
        Context:   op.Context,
    }

    // Encode sensitive data
    entry = a.encoder.Encode(entry)

    // Store
    a.storage.Save(entry)
}
```

## 10. Configuration System

### Configuration Structure

```yaml
# ~/.whodb-cli/ai.yaml
ai:
  provider: lmstudio

  lmstudio:
    url: http://localhost:1234/v1
    model: liquid/lfm2-1.2b
    max_tokens: 2048
    temperature: 0.3
    timeout: 30s
    stream: true

  optimization:
    cache_enabled: true
    cache_ttl: 3600
    compress_prompts: true
    abbreviations: true
    max_context_tokens: 1000

  features:
    tools_enabled: true
    multi_step_planning: true
    auto_explain: true
    safe_mode: true
    learning_enabled: true
    split_view: false

  tools:
    query_execute:
      enabled: true
      max_rows: 1000
      timeout: 30s
      explain_first: true

    schema_inspector:
      enabled: true
      include_stats: true

    data_profiler:
      enabled: true
      sample_size: 1000

    index_advisor:
      enabled: true
      threshold: 0.7

    migration_tool:
      enabled: true
      dry_run: true

  safety:
    enable_transactions: true
    max_rows_affected: 1000
    require_where_clause: true
    block_truncate: true
    block_drops: true
    dry_run_ddl: true
    audit_log: true

  learning:
    enabled: true
    storage_path: ~/.whodb-cli/patterns.db
    max_patterns: 10000
    min_frequency: 3

  ui:
    split_view_ratio: 0.6
    show_suggestions: true
    auto_complete: true
    syntax_highlighting: true
```

## 11. Performance Optimizations

### Caching System

```go
// internal/ai/cache/cache_manager.go
package cache

type CacheManager struct {
    schemaCache  *SchemaCache
    promptCache  *PromptCache
    resultCache  *ResultCache
    patternCache *PatternCache
}

type SchemaCache struct {
    entries map[string]*SchemaCacheEntry
    ttl     time.Duration
}

type SchemaCacheEntry struct {
    Schema    *engine.Schema
    Tables    []*engine.Table
    Relations []*engine.Relation
    Timestamp time.Time
}

func (c *SchemaCache) Get(database string) (*SchemaCacheEntry, bool) {
    entry, exists := c.entries[database]
    if !exists {
        return nil, false
    }

    // Check TTL
    if time.Since(entry.Timestamp) > c.ttl {
        delete(c.entries, database)
        return nil, false
    }

    return entry, true
}
```

### Token Optimization

```go
// internal/ai/optimization/token_manager.go
type TokenManager struct {
    counter     *TokenCounter
    compressor  *TextCompressor
    prioritizer *ContentPrioritizer
}

func (m *TokenManager) OptimizeForModel(content string, maxTokens int) string {
    // Count current tokens
    currentTokens := m.counter.Count(content)

    if currentTokens <= maxTokens {
        return content
    }

    // Prioritize content
    sections := m.prioritizer.Prioritize(content)

    // Compress iteratively
    compressed := content
    for _, section := range sections {
        compressed = m.compressor.CompressSection(compressed, section)

        if m.counter.Count(compressed) <= maxTokens {
            break
        }
    }

    return compressed
}
```

## 12. Testing Strategy

### Unit Tests

```go
// internal/ai/tools/query_tool_test.go
func TestQueryToolExecution(t *testing.T) {
    tool := &QueryTool{maxRows: 100}
    ctx := &DatabaseContext{
        Manager: mockManager,
        SafeMode: true,
    }

    tests := []struct {
        name    string
        params  map[string]any
        wantErr bool
    }{
        {
            name:    "valid select",
            params:  map[string]any{"query": "SELECT * FROM users"},
            wantErr: false,
        },
        {
            name:    "destructive in safe mode",
            params:  map[string]any{"query": "DROP TABLE users"},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := tool.Execute(tt.params, ctx)
            if (err != nil) != tt.wantErr {
                t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Integration Tests

```go
// internal/ai/integration_test.go
func TestEndToEndAIFlow(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()

    // Create AI system
    ai := setupAISystem(t)

    // Test query optimization flow
    response, err := ai.Process("optimize my slow queries")
    require.NoError(t, err)

    // Verify tools were called
    assert.Contains(t, response.ToolsCalled, "query_profiler")
    assert.Contains(t, response.ToolsCalled, "explain_query")
    assert.Contains(t, response.ToolsCalled, "index_advisor")
}
```

## Implementation Timeline

### Week 1: Core Infrastructure
- Day 1-2: LM Studio provider integration
- Day 3-4: Tool system architecture
- Day 5-7: Basic context management and caching

### Week 2: Essential Tools
- Day 1-2: QueryTool and SchemaInspectorTool
- Day 3-4: ExplainTool and DataProfilerTool
- Day 5-6: Multi-step planner
- Day 7: Enhanced chat view with split mode

### Week 3: Advanced Features
- Day 1-2: Specialized agents (QueryOptimizer, DataAnalyst)
- Day 3-4: Smart query builder with NL2SQL
- Day 5-6: Learning system and pattern recognition
- Day 7: Testing and debugging

### Week 4: Polish & Optimization
- Day 1-2: UI enhancements and tool palette
- Day 3-4: Performance tuning for small model
- Day 5-6: Safety system and governance
- Day 7: Documentation and examples

## Success Metrics

1. **Response Time**: < 2 seconds for simple queries with local model
2. **Accuracy**: > 90% correct SQL generation for common patterns
3. **Tool Success Rate**: > 95% successful tool executions
4. **Memory Usage**: < 100MB additional memory overhead
5. **User Satisfaction**: Reduce time to complete database tasks by 50%

## Future Enhancements

1. **Model Fine-tuning**: Fine-tune the small model specifically for database operations
2. **Plugin System**: Allow custom tools and agents via plugins
3. **Collaboration**: Multi-user support with shared contexts
4. **Cloud Sync**: Sync patterns and preferences across machines
5. **Alternative Models**: Support for other local models (Ollama, llama.cpp)
6. **Visual Query Builder**: Drag-and-drop interface integrated with AI
7. **Performance Monitoring**: Real-time database performance dashboard
8. **Automated Testing**: AI-generated test cases for schema changes

## Conclusion

This comprehensive plan provides a robust foundation for implementing a Claude Code-like experience in the WhoDB CLI, specifically optimized for database operations with a small local LLM. The modular architecture allows for incremental implementation while maintaining high quality and performance standards.