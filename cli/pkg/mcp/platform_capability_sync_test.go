package mcp

import (
	"context"
	"reflect"
	"testing"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

func TestCapabilityReadsValidateScopeAndBounds(t *testing.T) {
	client := &fakePlatformClient{platformQueryResult: []any{map[string]any{"id": "one"}, map[string]any{"id": "two"}}}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) { return testPlatformSession(client), nil })
	for _, c := range platformapi.CapabilityReads {
		t.Run(c.Operation, func(t *testing.T) {
			in := PlatformCapabilityReadInput{ID: "record", FunctionID: "fn", AppID: "app", SessionID: "session", FeatureKey: "feature", Limit: 1}
			_, out, err := handlePlatformCapabilityRead(context.Background(), c, in, &SecurityOptions{MaxRows: 1})
			if err != nil || out.Error != "" {
				t.Fatalf("output %v err %v", out.Error, err)
			}
			expectedProject := "proj-1"
			if c.OrganizationScoped() {
				expectedProject = ""
			}
			if client.platformQueryOperation != c.Operation || client.platformQueryVariables["projectId"] != expectedProject {
				t.Fatalf("wrong operation/scope: %s %#v", client.platformQueryOperation, client.platformQueryVariables)
			}
			if client.platformQueryVariables["limit"] != 1 || out.Count != 1 || !out.Truncated {
				t.Fatalf("bounds not enforced: %#v", out)
			}
			if len(c.Required) > 0 {
				_, out, err = handlePlatformCapabilityRead(context.Background(), c, PlatformCapabilityReadInput{}, &SecurityOptions{})
				if err != nil || out.Error == "" {
					t.Fatalf("missing selector accepted: %#v %v", out, err)
				}
			}
		})
	}
}
func TestCapabilityArtifactPagePreservesContinuation(t *testing.T) {
	page := map[string]any{"rows": []any{map[string]any{"id": 1}}, "nextCursor": "opaque", "execution": map[string]any{"complete": false}, "artifactId": "artifact"}
	client := &fakePlatformClient{platformQueryResult: page}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) { return testPlatformSession(client), nil })
	for _, c := range platformapi.CapabilityReads {
		if c.Operation == "AnalysisArtifactPage" {
			_, out, err := handlePlatformCapabilityRead(context.Background(), c, PlatformCapabilityReadInput{ID: "artifact", Cursor: "previous", Limit: 2}, &SecurityOptions{MaxRows: 2})
			if err != nil || out.Error != "" || out.Count != 1 || !reflect.DeepEqual(out.Data, page) {
				t.Fatalf("lost continuation/completeness: %#v %v", out, err)
			}
			if client.platformQueryVariables["cursor"] != "previous" {
				t.Fatal("cursor not forwarded")
			}
		}
	}
}
func TestCapabilityWriteConfirmationBindsIdentifiers(t *testing.T) {
	session := testPlatformSession(&fakePlatformClient{})
	tests := []struct {
		kind             string
		in               PlatformGenericWriteInput
		mutation, target string
	}{
		{"action", PlatformGenericWriteInput{Resource: "edge_function", Action: "run", ID: "fn", Payload: map[string]any{"runnerId": "runner", "input": "{}"}}, "StartEdgeFunctionRun", "functionId"},
		{"action", PlatformGenericWriteInput{Resource: "edge_schedule", Action: "stop", ID: "schedule"}, "StopEdgeSchedule", "id"},
		{"action", PlatformGenericWriteInput{Resource: "agent_session", Action: "cancel", ID: "session"}, "CancelAgentSession", "sessionId"},
	}
	for _, tt := range tests {
		spec, vars, err := buildPlatformGenericWrite(session, tt.in, tt.kind)
		if err != nil {
			t.Fatal(err)
		}
		if spec.Mutation != tt.mutation || vars[tt.target] != tt.in.ID {
			t.Fatalf("wrong target: %#v %#v", spec, vars)
		}
	}
}

func TestCapabilityWritesWaitForConfirmation(t *testing.T) {
	clearPendingPlatformActions(t)
	client := &fakePlatformClient{}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) { return testPlatformSession(client), nil })
	_, pending, err := handlePlatformGenericWrite(context.Background(), "platform_action", PlatformGenericWriteInput{Resource: "edge_function", Action: "run", ID: "fn", Payload: map[string]any{"runnerId": "runner", "input": "{}"}}, "action", true)
	if err != nil || pending.Error != "" || !pending.ConfirmationRequired || client.mutationName != "" {
		t.Fatalf("write ran before confirmation: %#v %v", pending, err)
	}
	_, out, err := HandlePlatformConfirm(context.Background(), nil, ConfirmInput{Token: pending.ConfirmationToken})
	if err != nil || out.Error != "" || client.mutationName != "StartEdgeFunctionRun" || client.mutationVariables["functionId"] != "fn" || client.mutationVariables["projectId"] != "proj-1" {
		t.Fatalf("confirmed write lost target: %#v %v", out, err)
	}
}

func TestCapabilityOrganizationDiagnosticsNeedNoProject(t *testing.T) {
	client := &fakePlatformClient{platformQueryResult: map[string]any{"features": []any{}}}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		session := testPlatformSession(client)
		session.Host.DefaultProjectID = ""
		session.Host.DefaultProjectName = ""
		return session, nil
	})
	for _, c := range platformapi.CapabilityReads {
		if c.OrganizationScoped() {
			_, out, err := handlePlatformCapabilityRead(context.Background(), c, PlatformCapabilityReadInput{FeatureKey: "functions"}, nil)
			if err != nil || out.Error != "" || out.Scope.OrgID != "org-1" || out.Scope.ProjectID != "" {
				t.Fatalf("org diagnostic required project: %#v %v", out, err)
			}
		}
	}
}

func TestCapabilityAppAccessRejectsOversizedPage(t *testing.T) {
	for _, c := range platformapi.CapabilityReads {
		if c.Operation == "AppAccessDirectory" {
			_, out, err := handlePlatformCapabilityRead(context.Background(), c, PlatformCapabilityReadInput{AppID: "app", Limit: 101}, nil)
			if err != nil || out.Error == "" {
				t.Fatalf("oversized app access page accepted: %#v %v", out, err)
			}
		}
	}
}
