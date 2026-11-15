package platform

// CapabilityRead describes one bounded platform read shared by CLI and MCP discovery.
type CapabilityRead struct {
	Operation   string
	Tool        string
	Description string
	Required    []string
}

// CapabilityReads documents the platform operations added to the automation surface.
var CapabilityReads = []CapabilityRead{
	{"EdgeRunners", "edge_runners", "List visible edge devices, presence and the caller's manage/execute permissions.", nil},
	{"EdgeFunctionRun", "edge_run", "Inspect one edge run, including status, logs, output and pinned function version.", []string{"id"}},
	{"EdgeFunctionRuns", "edge_runs", "List recent edge runs for a function. Use edge_run to inspect a run again; do not start a duplicate run after a timeout.", []string{"functionId"}},
	{"EdgeSchedules", "edge_schedules", "List edge interval schedules for a function. These are distinct from hosted function cron schedules.", []string{"functionId"}},
	{"AppAccessDirectory", "app_access", "Search effective app users and direct team grants, at most 100 users per page. Requires sharing permission. Continue with offset while offset + pageSize is below total.", []string{"appId"}},
	{"ProjectTransformRuns", "project_transform_runs", "List recent runs across viewable project transforms, bounded by total limit and per_transform.", nil},
	{"AgentSessions", "agent_sessions", "List the signed-in user's AI sessions in the selected project; this does not start or resume an AI run.", nil},
	{"AgentSession", "agent_session", "Read the signed-in user's AI session and persisted messages. Artifacts can be inspected separately.", []string{"sessionId"}},
	{"AnalysisArtifact", "analysis_artifact", "Read an authorized analysis artifact's metadata, warnings and provenance. Partial status and warnings must be preserved when interpreting results.", []string{"id"}},
	{"AnalysisArtifactPage", "analysis_artifact_page", "Read a bounded page of analysis rows. Continue with nextCursor until absent; preserve execution completeness metadata.", []string{"id"}},
	{"BillingCurrentUsage", "usage", "Read current organization usage and enforced limits. This does not modify billing.", nil},
	{"BillingCurrentEntitlements", "entitlements", "Read the caller's effective feature entitlements and limits.", nil},
	{"CanAccessFeature", "feature_access", "Check whether the caller can access one platform feature key.", []string{"featureKey"}},
}

const edgeRunFields = `id functionId runnerId scheduleId status input output logs error functionVersion startedAt completedAt createdAt`
const edgeScheduleFields = `id functionId runnerId intervalSeconds nextAt active functionVersion createdAt`
const entitlementFields = `limits { key value unlimited } features { key enabled }`

func init() {
	queries := map[string]string{
		"EdgeRunners":                `query CLIEdgeRunners($projectId: ID!) { EdgeRunners(projectId: $projectId) { id name online canManage canExecute lastSeenAt createdAt } }`,
		"EdgeFunctionRun":            `query CLIEdgeFunctionRun($projectId: ID!, $id: ID!) { EdgeFunctionRun(projectId: $projectId, id: $id) { ` + edgeRunFields + ` } }`,
		"EdgeFunctionRuns":           `query CLIEdgeFunctionRuns($projectId: ID!, $functionId: ID!, $limit: Int) { EdgeFunctionRuns(projectId: $projectId, functionId: $functionId, limit: $limit) { ` + edgeRunFields + ` } }`,
		"EdgeSchedules":              `query CLIEdgeSchedules($projectId: ID!, $functionId: ID!) { EdgeSchedules(projectId: $projectId, functionId: $functionId) { ` + edgeScheduleFields + ` } }`,
		"AppAccessDirectory":         `query CLIAppAccessDirectory($projectId: ID!, $appId: ID!, $search: String, $pageSize: Int, $offset: Int) { AppAccessDirectory(projectId: $projectId, appId: $appId, search: $search, pageSize: $pageSize, offset: $offset) { users { id email displayName } teams { id name memberCount } total pageSize offset } }`,
		"ProjectTransformRuns":       `query CLIProjectTransformRuns($projectId: ID!, $perTransform: Int, $limit: Int) { ProjectTransformRuns(projectId: $projectId, perTransform: $perTransform, limit: $limit) { id transformId transformName status startedAt completedAt } }`,
		"AgentSessions":              `query CLIAgentSessions($projectId: ID!, $limit: Int) { AgentSessions(projectId: $projectId, limit: $limit) { sessionType id projectId sourceId title status activeRunId modelType providerId model createdAt updatedAt } }`,
		"AgentSession":               `query CLIAgentSession($projectId: ID!, $sessionId: ID!) { AgentSession(projectId: $projectId, sessionId: $sessionId) { sessionType id projectId sourceId title status activeRunId modelType providerId model createdAt updatedAt messages { id role kind text content createdAt } } }`,
		"AnalysisArtifact":           `query CLIAnalysisArtifact($id: ID!) { AnalysisArtifact(id: $id) { id kind status title datasetId viewType encoding options schema { id label physicalType semanticType unit nullable } warnings provenance { id planHash sources warnings } } }`,
		"AnalysisArtifactPage":       `query CLIAnalysisArtifactPage($id: ID!, $cursor: String, $limit: Int!) { AnalysisArtifactPage(id: $id, cursor: $cursor, limit: $limit) { artifactId rows nextCursor execution } }`,
		"BillingCurrentUsage":        `query CLIBillingCurrentUsage { BillingCurrentUsage { currentPlan currentInterval entitlements { ` + entitlementFields + ` } usage { key scopeType scopeId scopeName value } } }`,
		"BillingCurrentEntitlements": `query CLIBillingCurrentEntitlements { BillingCurrentEntitlements { ` + entitlementFields + ` } }`,
		"CanAccessFeature":           `query CLICanAccessFeature($featureKey: String!) { CanAccessFeature(featureKey: $featureKey) }`,
	}
	for name, query := range queries {
		platformQuerySpecs[name] = query
	}
	type write struct {
		resource, action, mutation, identity, args, params, fields string
		project                                                    bool
		shape                                                      []PayloadField
	}
	writes := []write{
		{"edge_function", "run", "StartEdgeFunctionRun", "functionId", "$projectId: ID!, $functionId: ID!, $runnerId: ID!, $input: String!", "projectId: $projectId, functionId: $functionId, runnerId: $runnerId, input: $input", edgeRunFields, true, []PayloadField{{Name: "runnerId", Type: "string", Required: true, Description: "Assigned runner id; requires execution access and a promoted asynchronous Python function or live Python service"}, {Name: "input", Type: "string", Required: true, Description: "JSON object input"}}},
		{"edge_schedule", "create", "CreateEdgeSchedule", "", "$projectId: ID!, $functionId: ID!, $runnerId: ID!, $input: String!, $intervalSeconds: Int!", "projectId: $projectId, functionId: $functionId, runnerId: $runnerId, input: $input, intervalSeconds: $intervalSeconds", edgeScheduleFields, true, []PayloadField{{Name: "functionId", Type: "string", Required: true, Description: "Promoted asynchronous Python function or live Python service id"}, {Name: "runnerId", Type: "string", Required: true, Description: "Assigned edge runner id; caller needs device execution access"}, {Name: "input", Type: "string", Required: true, Description: "JSON object input"}, {Name: "intervalSeconds", Type: "integer", Required: true, Description: "Interval in seconds, from 10 through 3600"}}},
		{"edge_schedule", "stop", "StopEdgeSchedule", "id", "$projectId: ID!, $id: ID!", "projectId: $projectId, id: $id", edgeScheduleFields, true, nil},
		{"agent_session", "cancel", "CancelAgentSession", "sessionId", "$sessionId: ID!", "sessionId: $sessionId", statusResponseFields, false, nil},
	}
	for _, w := range writes {
		key := "action:" + w.action + ":" + w.resource
		if w.action == "create" {
			key = "create:" + w.resource
		}
		GenericWriteSpecs[key] = GenericWriteSpec{Resource: w.resource, Action: w.action, Mutation: w.mutation, Mode: GenericWriteModeDirect, NeedsID: w.identity != "", IdentityField: w.identity, InjectProjectID: w.project}
		PayloadShapes[key] = PayloadShape{Key: key, Resource: w.resource, Action: w.action, Description: w.mutation + " through the platform's authorized operation; requires confirmation.", Fields: w.shape}
		platformMutationSpecs[w.mutation] = mutationSpecWithDirect(w.mutation, w.args, w.params, w.fields)
	}
}

// OrganizationScoped reports reads whose authority does not require a project.
func (c CapabilityRead) OrganizationScoped() bool {
	return c.Operation == "BillingCurrentUsage" || c.Operation == "BillingCurrentEntitlements" || c.Operation == "CanAccessFeature"
}
