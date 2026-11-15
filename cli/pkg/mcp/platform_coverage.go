package mcp

import platformapi "github.com/clidey/whodb/cli/internal/platform"

// PlatformAutomationOperations exposes the implemented API inventory for host coverage checks.
func PlatformAutomationOperations() map[string][]string {
	result := platformapi.AutomationOperations()
	for _, extension := range platformExtensions {
		kind := "Query"
		if extension.Write {
			kind = "Mutation"
		}
		result[kind] = append(result[kind], extension.Operation)
	}
	return result
}

// PlatformAutomationFunctionFields reports the function fields understood by automation.
func PlatformAutomationFunctionFields() []string { return platformapi.AutomationFunctionFields() }
