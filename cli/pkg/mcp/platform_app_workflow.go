package mcp

type platformAppWorkflow struct {
	Purpose      string   `json:"purpose"`
	FolderFormat []string `json:"folder_format"`
	Design       []string `json:"design"`
	BuildSteps   []string `json:"build_steps"`
	HostedMCP    []string `json:"hosted_mcp"`
	LocalCLI     []string `json:"local_cli"`
	Verification []string `json:"verification"`
	Limits       []string `json:"limits"`
}

func buildPlatformAppWorkflow() platformAppWorkflow {
	return platformAppWorkflow{
		Purpose: "Build an app from real project resources, verify its interactions, and promote only after review.",
		FolderFormat: []string{
			"app.json declares name, defaultPage, and ontology/function bindings; preserve other app settings.",
			"pages/<page>/tree.json holds the editable component tree; pages/<page>/state.json holds page state.",
			"pages/<page>/scripts/<handler>.js implements events; shared/state.json holds state that survives navigation.",
		},
		Design: []string{
			"Start from the user's brief, audience, main task, and any reference image or existing brand. Preserve an established design system when revising an app.",
			"Before editing, choose a specific layout concept, type hierarchy, 4-6 named color tokens, density, and one memorable detail grounded in the app's subject. Review those choices against the brief and revise generic defaults.",
			"Make the first screen's hierarchy lead to the primary task. Use realistic domain copy and data; make labels, dividers, color, and motion clarify meaning rather than decorate every surface.",
			"Keep the visual system consistent across pages, sheets, drawers, tables, and states. Check contrast, visible keyboard focus, readable type, and reduced-motion behavior where motion is used.",
			"Inspect rendered screenshots at desktop, mobile, tablet, and landscape sizes. Critique hierarchy, legibility, spacing, clipping, and density, then make targeted revisions and capture again.",
		},
		BuildSteps: []string{
			"Inspect the selected project, ontology schemas, representative rows, functions, and runtime dependencies before writing app code.",
			"Plan users, tasks, pages, overlays, state keys, loading/error states, scroll ownership, and resource bindings. Use separate pages for distinct workflows.",
			"Build the page layout and complete component subtrees, then bind data and connect events to scripts or built-in navigation handlers.",
			"For a pinned desktop sidebar, use a viewport-height root Row with overflow hidden, a full-height Sidebar, and a full-height main Col with minH 0 and overflow auto. The main pane must own long-page scrolling; a full-height root that overflows into the sandbox viewport makes a sticky sidebar scroll away.",
			"Trace each interaction from event through handler, state change, bridge call, and visible result. Check failed and empty states.",
			"Review desktop, mobile, tablet, and landscape views; verify the real workflow before promoting.",
		},
		HostedMCP: []string{
			"Use whodb_platform_workspace_summary, whodb_platform_ontology_describe, whodb_platform_ontology_rows, whodb_platform_functions, and whodb_platform_runtime_readiness to discover real resources.",
			"Call whodb_platform_app_components and whodb_platform_app_component before using unfamiliar components; call whodb_platform_app_sdks and whodb_platform_app_sdk before writing app data or function integrations.",
			"Use whodb_platform_apps and whodb_platform_app_files to inspect an existing draft. Read whodb://platform/schema for exact app create, update, file action, and promotion payloads.",
			"Use whodb_platform_app_logic_graph to inspect event-to-script and cross-page flows or find broken wiring before editing and after publishing a draft.",
			"Use whodb_platform_create, whodb_platform_update, and whodb_platform_action for hosted draft changes. Follow the active MCP write confirmation mode.",
			"Use whodb_platform_app_view and whodb_platform_app_screenshot to inspect rendered hosted pages at desktop, mobile, and tablet viewports. Request trace=true when the hosted runtime supports tracing.",
			"Read back files and promoted versions after writes; do not claim a flow works from file content alone.",
		},
		LocalCLI: []string{
			"With local filesystem and CLI access, run whodb apps init ./my-app --from <app-id-or-name> to start from an existing app, or whodb apps init ./my-app --name <name> for a starter.",
			"Edit the files in version control, then run whodb apps check ./my-app.",
			"Run whodb apps publish ./my-app --draft to create or sync the hosted draft and its resource bindings before preview.",
			"Choose an isolated development workspace before exercising app actions. apps dev and apps verify allow hosted writes by default; local files are overlaid in the browser, but ontology and function calls run against the selected hosted project.",
			"Run whodb apps dev ./my-app for live reload and whodb apps verify ./my-app --scenario <scenario.json> --format json for a screenshot, diagnostics, and flow trace. Keep --host, --org, and --project aligned with the MCP workspace.",
			"Do not pass --allow-writes=false when testing ctx.whodb.call(): every function call, including a reporting function, uses a GraphQL mutation and is blocked in that mode. Use --allow-writes=true only where function execution is permitted.",
			"Repeat verification with --mobile, --tablet, and --orientation landscape. Run whodb apps publish ./my-app --dry-run, then publish after reviewing the plan.",
		},
		Verification: []string{
			"Confirm every event handler exists and every navigation target has a page tree.",
			"Exercise create, edit, delete, cancel, empty, failure, and multi-page paths that the app promises.",
			"Scroll the longest desktop page and confirm the main pane moves while the sidebar stays visible; repeat after the phone layout collapses.",
			"Inspect screenshot, visible text, console/network diagnostics, and interaction trace; re-run after fixes.",
			"Confirm bound ontologies/functions match the selected hosted app before preview and after publish.",
		},
		Limits: []string{
			"Hosted MCP can inspect and edit hosted app files, but it cannot read or write a local version-controlled folder without separate filesystem access.",
			"MCP read-only mode and apps dev --allow-writes=false are separate controls. The latter blocks all preview function calls; a reporting function can still have side effects, so the CLI cannot infer that it is safe from its name.",
			"The hosted screenshot tool captures hosted files and can request a trace from compatible runtimes; the local CLI verify command captures local file overrides and returns a trace.",
			"MCP currently exposes file-level app editing, component/SDK references, and the in-app builder's logic audit, but not its surgical node-editing tools.",
		},
	}
}
