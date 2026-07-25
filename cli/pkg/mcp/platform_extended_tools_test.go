package mcp

import (
	"context"
	"reflect"
	"testing"
)

func TestHandlePlatformExtendedQueryInjectsSelectedWorkspace(t *testing.T) {
	client := &fakePlatformClient{platformQueryResult: []any{map[string]any{"id": "app-1"}}}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	_, output, err := handlePlatformExtendedQuery(context.Background(), "platform_apps", "ProjectApps", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("handlePlatformExtendedQuery() error = %v", err)
	}
	if output.Error != "" || output.Count != 1 {
		t.Fatalf("output = %#v, want one app without error", output)
	}
	if client.platformQueryOperation != "ProjectApps" {
		t.Fatalf("operation = %q, want ProjectApps", client.platformQueryOperation)
	}
	if client.platformQueryVariables["projectId"] != "proj-1" || client.platformQueryVariables["orgId"] != "org-1" {
		t.Fatalf("variables = %#v, want selected workspace ids", client.platformQueryVariables)
	}
}

func TestHandlePlatformExtendedQueryProjectsFieldsAndScope(t *testing.T) {
	client := &fakePlatformClient{platformQueryResult: map[string]any{
		"id": "app-1", "name": "Customer App", "private": "must-not-be-returned",
	}}
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(client), nil
	})

	fields := []string{"id", "name"}
	_, output, err := handlePlatformExtendedQuery(context.Background(), "platform_app", "AppDetail", map[string]any{"id": "app-1"}, fields)
	if err != nil {
		t.Fatalf("handlePlatformExtendedQuery() error = %v", err)
	}
	if output.Error != "" {
		t.Fatalf("output error = %q", output.Error)
	}
	if output.RequestID == "" {
		t.Fatal("output request_id is empty")
	}
	if output.Scope == nil || output.Scope.OrgID != "org-1" || output.Scope.ProjectID != "proj-1" {
		t.Fatalf("output scope = %#v, want selected org/project", output.Scope)
	}
	if !reflect.DeepEqual(output.Fields, fields) {
		t.Fatalf("output fields = %#v, want %#v", output.Fields, fields)
	}
	data, ok := output.Data.(map[string]any)
	if !ok {
		t.Fatalf("output data = %T, want map[string]any", output.Data)
	}
	if !reflect.DeepEqual(data, map[string]any{"id": "app-1", "name": "Customer App"}) {
		t.Fatalf("projected data = %#v, want only requested fields", data)
	}
}

func TestHandlePlatformExtendedQueryRejectsInvalidInputsBeforeRequest(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		variables map[string]any
		wantError string
	}{
		{name: "missing app id", operation: "AppDetail", variables: map[string]any{"id": ""}, wantError: "id is required"},
		{name: "missing version object type", operation: "ObjectVersions", variables: map[string]any{"objectId": "dataset-1"}, wantError: "object_type is required"},
		{name: "missing access resource type", operation: "WhatCanIAccess", variables: map[string]any{}, wantError: "resource_type is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakePlatformClient{}
			withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
				return testPlatformSession(client), nil
			})
			_, output, err := handlePlatformExtendedQuery(context.Background(), "platform_test", tt.operation, tt.variables, nil)
			if err != nil {
				t.Fatalf("handlePlatformExtendedQuery() error = %v", err)
			}
			if output.Error != tt.wantError {
				t.Fatalf("output error = %q, want %q", output.Error, tt.wantError)
			}
			if client.platformQueryOperation != "" {
				t.Fatalf("platform query operation = %q, want no request", client.platformQueryOperation)
			}
		})
	}
}

func TestBuildPlatformGenericWriteAcceptsStructuredPayload(t *testing.T) {
	session := testPlatformSession(&fakePlatformClient{})
	spec, variables, err := buildPlatformGenericWrite(session, PlatformGenericWriteInput{
		Resource: "dataset",
		Payload:  map[string]any{"name": "customers", "schemaMode": "manual"},
	}, "create")
	if err != nil {
		t.Fatalf("buildPlatformGenericWrite() error = %v", err)
	}
	if spec.Mutation != "CreateDataset" {
		t.Fatalf("mutation = %q, want CreateDataset", spec.Mutation)
	}
	input, ok := variables["input"].(map[string]any)
	if !ok {
		t.Fatalf("variables[input] = %T, want map[string]any", variables["input"])
	}
	if input["name"] != "customers" || input["projectId"] != "proj-1" {
		t.Fatalf("input = %#v, want structured payload with selected project", input)
	}
}

func TestBuildPlatformGenericWriteRejectsAmbiguousPayloads(t *testing.T) {
	_, _, err := buildPlatformGenericWrite(testPlatformSession(&fakePlatformClient{}), PlatformGenericWriteInput{
		Resource:    "dataset",
		Payload:     map[string]any{"name": "customers"},
		PayloadJSON: `{"name":"customers"}`,
	}, "create")
	if err == nil || err.Error() != "provide payload or payload_json, not both" {
		t.Fatalf("error = %v, want ambiguous payload error", err)
	}
}
