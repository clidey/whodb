package mcp

import (
	"fmt"
	"slices"
	"strings"
)

// ValidateToolProfile rejects unknown discovery profiles instead of silently
// exposing the full catalog. Profiles never change authorization or write mode.
func ValidateToolProfile(profile string) error {
	if !slices.Contains([]string{"", "all", "data", "functions", "workspace"}, profile) {
		return fmt.Errorf("unknown tool profile %q; choose all, data, functions, or workspace", profile)
	}
	return nil
}

func profileAllowsTool(profile, name string) bool {
	if profile == "" || profile == "all" {
		return true
	}
	if ValidateToolProfile(profile) != nil {
		return false
	}
	name = strings.TrimPrefix(name, "whodb_")
	if profile == "workspace" {
		for _, extension := range platformExtensions {
			if extension.Name == "whodb_"+name {
				return true
			}
		}
	}
	if name == "mcp_setup" {
		return true
	}
	if !strings.HasPrefix(name, "platform_") {
		return profile == "data"
	}
	name = strings.TrimPrefix(name, "platform_")
	if slices.Contains(strings.Fields("status hosts orgs projects workspace_resolve workflow_recipe resolve_resource setup_status capabilities my_grants project_access confirm pending write_safety prepare_safe_write write_plan"), name) {
		return true
	}
	var names string
	switch profile {
	case "data":
		names = "sources source_types source_fields source_objects source_columns source_rows source_constraints source_content source_workflow datasets dataset dataset_preview ontologies ontology ontology_describe ontology_rows ontology_query ontology_aggregate ontology_stats ontology_follow_link ontology_record_capabilities data_model_summary read_workflow"
	case "functions":
		names = "functions function function_runs function_wait runtime_readiness ai_providers secrets ontologies ontology_describe create update delete action restore object_versions"
	case "workspace":
		names = "workspace_summary workspace_map resource_graph project_health next_actions build_plan gap_analysis project_resources project_create project_rename project_delete service_accounts resource_access resource_permissions apps packages package_installations org_resources org_members teams team_members security_audit"
	}
	return slices.Contains(strings.Fields(names), name)
}
