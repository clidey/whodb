package platform

import (
	"sort"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

// AutomationOperations reports the root API fields implemented by the shared client.
func AutomationOperations() map[string][]string {
	seen := map[string]map[string]bool{"Query": {}, "Mutation": {}}
	add := func(document string) {
		parsed, err := parser.ParseQuery(&ast.Source{Input: document})
		if err != nil {
			panic(err)
		}
		for _, op := range parsed.Operations {
			kind := "Query"
			if op.Operation == ast.Mutation {
				kind = "Mutation"
			}
			for _, selection := range op.SelectionSet {
				if f, ok := selection.(*ast.Field); ok {
					seen[kind][f.Name] = true
				}
			}
		}
	}
	for _, document := range platformOperations {
		add(document)
	}
	for _, document := range platformQuerySpecs {
		add(document)
	}
	for _, spec := range platformMutationSpecs {
		add(spec.Query)
	}
	// Specialized methods execute these documents outside the generic registries.
	add(operationFunctionRun)
	add(operationFunctionRuns)
	seen["Query"]["PlatformResources"] = true
	seen["Query"]["SecurityAudit"] = true
	seen["Mutation"]["UploadProjectFile"] = true
	result := map[string][]string{}
	for kind, names := range seen {
		for name := range names {
			result[kind] = append(result[kind], name)
		}
		sort.Strings(result[kind])
	}
	return result
}

// AutomationFunctionFields reports function fields understood by the client.
func AutomationFunctionFields() []string {
	fields := make([]string, 0, len(functionSelectionFields))
	for name := range functionSelectionFields {
		fields = append(fields, name)
	}
	sort.Strings(fields)
	return fields
}

// AutomationDocuments returns independent copies of the bounded query and write
// documents so host-side tests can validate the complete automation contract.
func AutomationDocuments() map[string]string {
	documents := map[string]string{}
	for key, document := range platformOperations {
		documents[key] = document
	}
	for key, document := range platformQuerySpecs {
		documents[key] = document
	}
	for key, spec := range platformMutationSpecs {
		documents[key] = spec.Query
	}
	documents["FunctionRun"] = operationFunctionRun
	documents["FunctionRuns"] = operationFunctionRuns
	return documents
}
