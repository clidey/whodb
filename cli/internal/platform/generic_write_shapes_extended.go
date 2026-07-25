package platform

// The extended shapes keep the generic MCP write tool discoverable without
// duplicating the platform GraphQL schema in the client.
func init() {
	PayloadShapes["create:app"] = PayloadShape{
		Key: "create:app", Resource: "app", Action: "create", Description: "Create an app from selected ontology and function ids. projectId is injected.",
		Fields: []PayloadField{
			{Name: "name", Type: "string", Required: true, Description: "App name"},
			{Name: "description", Type: "string", Required: true, Description: "App description"},
			{Name: "ontologyIds", Type: "[ID!]", Required: true, Description: "Ontology ids used by the app"},
			{Name: "readOnlyOntologyIds", Type: "[ID!]", Description: "Ontologies available read-only"},
			{Name: "functionIds", Type: "[ID!]", Description: "Function ids used by the app"},
		},
		Examples: []string{`{"name":"Customer portal","description":"Customer data app","ontologyIds":["ontology-id"],"functionIds":[]}`},
	}
	PayloadShapes["update:app"] = PayloadShape{
		Key: "update:app", Resource: "app", Action: "update", Description: "Update app metadata or generated content. id and projectId are injected.",
		Fields: []PayloadField{
			{Name: "name", Type: "string", Description: "App name"},
			{Name: "description", Type: "string", Description: "App description"},
			{Name: "html", Type: "string", Description: "Generated app HTML"},
			{Name: "conversation", Type: "string", Description: "Generation conversation metadata"},
			{Name: "ontologyIds", Type: "[ID!]", Description: "Ontology ids used by the app"},
		},
	}
	PayloadShapes["action:generate:app"] = PayloadShape{
		Key: "action:generate:app", Resource: "app", Action: "generate", Description: "Generate app content using an AI provider. appId and projectId are injected.",
		Fields: []PayloadField{
			{Name: "appId", Type: "ID", Required: true, Description: "App id"},
			{Name: "prompt", Type: "string", Required: true, Description: "Generation request"},
			{Name: "modelType", Type: "string", Required: true, Description: "Model type"},
			{Name: "model", Type: "string", Required: true, Description: "Model name"},
			{Name: "providerId", Type: "string", Description: "Optional provider id"},
			{Name: "token", Type: "string", Secret: true, Description: "Optional provider token"},
		},
	}
	PayloadShapes["action:upsert_file:app"] = PayloadShape{
		Key: "action:upsert_file:app", Resource: "app", Action: "upsert_file", Description: "Create or replace an app file.",
		Fields: []PayloadField{{Name: "appId", Type: "ID", Required: true, Description: "App id"}, {Name: "path", Type: "string", Required: true, Description: "File path"}, {Name: "content", Type: "string", Required: true, Description: "File content"}},
	}
	PayloadShapes["action:delete_file:app"] = PayloadShape{
		Key: "action:delete_file:app", Resource: "app", Action: "delete_file", Description: "Delete an app file.",
		Fields: []PayloadField{{Name: "appId", Type: "ID", Required: true, Description: "App id"}, {Name: "path", Type: "string", Required: true, Description: "File path"}},
	}
	PayloadShapes["create:package"] = PayloadShape{
		Key: "create:package", Resource: "package", Action: "create", Description: "Create an immutable package from selected versionable objects. projectId is injected.",
		Fields: []PayloadField{{Name: "name", Type: "string", Required: true, Description: "Package name"}, {Name: "version", Type: "string", Description: "Package version"}, {Name: "channel", Type: "string", Description: "Release channel"}, {Name: "description", Type: "string", Description: "Package description"}, {Name: "items", Type: "[PackageItemInput!]", Required: true, Description: "Objects included in the package"}},
	}
	PayloadShapes["action:install:package"] = PayloadShape{
		Key: "action:install:package", Resource: "package", Action: "install", Description: "Install a package into the selected project.",
		Fields: []PayloadField{{Name: "sourceProjectId", Type: "ID", Required: true, Description: "Package source project"}, {Name: "targetProjectId", Type: "ID", Required: true, Description: "Installation target project"}, {Name: "packageId", Type: "ID", Required: true, Description: "Package id"}, {Name: "bindings", Type: "[PackageRequirementBindingInput!]", Description: "Requirement bindings"}},
	}
}
