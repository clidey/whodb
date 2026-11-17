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
	{"MySubscription", "subscription", "Read the selected organization's current subscription and renewal/cancellation status; does not change billing.", nil},
	{"AvailablePlans", "plans", "Inspect configured plans, prices and entitlements. Returns a configuration error when billing is unavailable.", nil},
	{"PlanFeatures", "plan_features", "Inspect the feature grants for one plan key.", []string{"plan"}},
	{"PreviewSubscriptionChange", "subscription_preview", "Preview plan eligibility and blockers without opening checkout or changing the subscription.", []string{"plan", "interval"}},
	{"ListScriptPackages", "script_packages", "List the platform's curated script package catalog; this does not install packages.", nil},
	{"SearchScriptPackages", "script_package_search", "Search the curated script package catalog by language, query and runtime; this does not install packages.", nil},
	{"GetScriptPackage", "script_package", "Inspect a curated script package by language and name; this does not install packages.", []string{"language", "name"}},
	{"TransformRun", "transform_run", "Inspect one authorized transform run by id, including runs outside the recent history window.", []string{"transformId", "id"}},
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
		"MySubscription":             `query CLISubscription { MySubscription { id orgId plan billingInterval pendingPlan pendingBillingInterval pendingChangeEffectiveAt status seatCount cancelAtPeriodEnd currentPeriodStart currentPeriodEnd gracePeriodEnd } }`,
		"AvailablePlans":             `query CLIPlans { AvailablePlans { id name description prices { stripePriceId interval unitAmount currency } entitlements { ` + entitlementFields + ` } } }`,
		"PlanFeatures":               `query CLIPlanFeatures($plan:String!) { PlanFeatures(plan:$plan) { featureKey allowed } }`,
		"PreviewSubscriptionChange":  `query CLISubscriptionPreview($plan:String!,$interval:String!) { PreviewSubscriptionChange(plan:$plan,interval:$interval) { currentPlan targetPlan targetInterval eligible targetEntitlements { ` + entitlementFields + ` } blockers { key scopeType scopeId scopeName currentValue limitValue } } }`,
		"ListScriptPackages":         `query CLIListScriptPackages($language:String) { ListScriptPackages(language:$language) { name version description language runtime categories } }`,
		"SearchScriptPackages":       `query CLISearchScriptPackages($language:String,$query:String,$runtime:String) { SearchScriptPackages(language:$language,query:$query,runtime:$runtime) { name version description language runtime categories } }`,
		"GetScriptPackage":           `query CLIGetScriptPackage($language:String!,$name:String!) { GetScriptPackage(language:$language,name:$name) { name version description language runtime categories } }`,
		"TransformRun":               operationTransformRun,
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
	return c.Operation == "MySubscription" || c.Operation == "AvailablePlans" || c.Operation == "PlanFeatures" || c.Operation == "PreviewSubscriptionChange" || c.Operation == "BillingCurrentUsage" || c.Operation == "BillingCurrentEntitlements" || c.Operation == "CanAccessFeature"
}

func init() {
	const key = "action:execute:script"
	GenericWriteSpecs[key] = GenericWriteSpec{Resource: "script", Action: "execute", Mutation: "ExecuteScript", Mode: GenericWriteModeInput, InjectProjectID: true}
	PayloadShapes[key] = PayloadShape{Key: key, Resource: "script", Action: "execute", Description: "Execute a script preview in the configured isolated runtime with project edit permission and confirmation. sampleData is JSON with columns and rows; omitted sampleData uses the platform's built-in sample. Check success, runtime and error in the result.", Fields: []PayloadField{
		{Name: "language", Type: "String", Required: true, Description: "Script language: python, r or julia"},
		{Name: "script", Type: "String", Required: true, Description: "Script source code; executed on the platform, never in the CLI process"},
		{Name: "dependencies", Type: "[String!]", Required: true, Description: "Curated package names required by the script"},
		{Name: "sampleData", Type: "String", Description: "Optional JSON object with columns and rows of string cells"},
	}}
	platformMutationSpecs["ExecuteScript"] = mutationSpecWithInput("ExecuteScript", "ExecuteScriptInput", "success runtime executionTimeMs output error inputRowCount outputRowCount")
}
