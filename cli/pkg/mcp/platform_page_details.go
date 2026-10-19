package mcp

import (
	"context"
	"encoding/json"
)

const (
	platformResourceFunction   = "function"
	platformResourceOntology   = "ontology"
	platformResourceDataset    = "dataset"
	platformResourceAIProvider = "ai_provider"
)

// Explicit fields can request full definitions for just the selected page.
// Default list calls stay metadata-only; the existing detail tools are unchanged.
func platformPageDetails(ctx context.Context, session *platformToolSession, resource string, items []map[string]any, fields []string) ([]map[string]any, bool, error) {
	if len(fields) == 0 || len(items) == 0 {
		return items, false, nil
	}
	needsDetail := false
	for _, field := range fields {
		if _, ok := items[0][field]; !ok {
			needsDetail = true
			break
		}
	}
	if !needsDetail {
		return items, false, nil
	}
	clipped := false
	for i, item := range items {
		id, _ := item["id"].(string)
		var data any
		var err error
		switch resource {
		case platformResourceFunction:
			function, e := session.Client.Function(ctx, session.Host.DefaultProjectID, id, fields)
			err = e
			if function != nil {
				clipped = truncateFunction(function, defaultPlatformContentLimit) || clipped
			}
			data = function
		case platformResourceOntology:
			data, err = session.Client.Ontology(ctx, session.Host.DefaultProjectID, id)
		case platformResourceDataset:
			data, err = session.Client.Dataset(ctx, session.Host.DefaultProjectID, id)
		case "app":
			data, err = session.Client.PlatformQuery(ctx, "AppDetail", map[string]any{"projectId": session.Host.DefaultProjectID, "id": id})
		default:
			return items, false, nil
		}
		if err != nil {
			return nil, false, err
		}
		raw, err := json.Marshal(data)
		if err != nil {
			return nil, false, err
		}
		var object map[string]any
		if err := json.Unmarshal(raw, &object); err != nil {
			return nil, false, err
		}
		items[i] = object
	}
	return items, clipped, nil
}
