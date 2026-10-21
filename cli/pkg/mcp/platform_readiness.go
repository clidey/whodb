package mcp

import (
	"context"
	"time"
)

// PlatformReadinessEvidence separates configuration from observed execution.
// Unknown is explicit: a metadata response does not establish runtime health.
type PlatformReadinessEvidence struct {
	Resource   string `json:"resource"`
	Configured string `json:"configured"`
	Reachable  string `json:"reachable"`
	Verified   string `json:"verified"`
	ObservedAt string `json:"observed_at"`
	Evidence   string `json:"evidence"`
}

func runtimeMetadataEvidence(snapshot *platformWorkspaceSnapshot) []PlatformReadinessEvidence {
	result := make([]PlatformReadinessEvidence, 0, 4)
	counts := platformWorkspaceCounts(snapshot)
	for _, area := range []string{"ai_providers", "secrets", "functions", "transforms"} {
		state := "absent"
		if snapshot.unavailable[area] {
			state = "unknown"
		} else if counts[area] > 0 {
			state = "configured"
		}
		result = append(result, PlatformReadinessEvidence{Resource: area, Configured: state,
			Reachable: "unknown", Verified: "unknown", ObservedAt: snapshot.observedAt,
			Evidence: "Configuration inventory only; worker capacity, provider authentication and execution have not been probed."})
	}
	return result
}

func addFunctionReadinessEvidence(ctx context.Context, session *platformToolSession, functionID string, readiness *PlatformRuntimeReadiness) {
	evidence := PlatformReadinessEvidence{Resource: "function:" + functionID, Configured: "unknown", Reachable: "unknown", Verified: "unknown", ObservedAt: time.Now().UTC().Format(time.RFC3339), Evidence: "No successful execution evidence available."}
	runs, err := session.Client.FunctionRuns(ctx, session.Host.DefaultProjectID, functionID, 1)
	if err != nil {
		readiness.Warnings = append(readiness.Warnings, "Function execution evidence unavailable.")
	} else if len(runs) > 0 {
		run := runs[0]
		evidence.Evidence = "Latest persisted run " + run.ID + " has status " + run.Status + "; historical evidence does not establish current worker availability."
		if run.Status == "completed" && run.CompletedAt != nil {
			evidence.Verified = "verified"
			evidence.ObservedAt = *run.CompletedAt
		}
	}
	readiness.Evidence = append(readiness.Evidence, evidence)
}
