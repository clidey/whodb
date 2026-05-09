package audit

import (
	"context"
	"testing"
)

func TestWithScopeMutatesExistingScopeState(t *testing.T) {
	ctx := WithIsolatedScope(context.Background(), Scope{OrgID: "org-1"})
	WithScope(ctx, Scope{ProjectID: "project-1"})

	scope := ScopeFromContext(ctx)
	if scope.OrgID != "org-1" {
		t.Fatalf("expected org id org-1, got %q", scope.OrgID)
	}
	if scope.ProjectID != "project-1" {
		t.Fatalf("expected project id project-1, got %q", scope.ProjectID)
	}
}

func TestWithIsolatedScopeDoesNotMutateParentScope(t *testing.T) {
	parent := WithIsolatedScope(context.Background(), Scope{OrgID: "org-1"})
	child := WithIsolatedScope(parent, Scope{ProjectID: "project-1"})

	WithScope(child, Scope{SourceID: "source-1"})

	parentScope := ScopeFromContext(parent)
	if parentScope.ProjectID != "" {
		t.Fatalf("expected parent project id to remain empty, got %q", parentScope.ProjectID)
	}
	if parentScope.SourceID != "" {
		t.Fatalf("expected parent source id to remain empty, got %q", parentScope.SourceID)
	}

	childScope := ScopeFromContext(child)
	if childScope.OrgID != "org-1" {
		t.Fatalf("expected child org id org-1, got %q", childScope.OrgID)
	}
	if childScope.ProjectID != "project-1" {
		t.Fatalf("expected child project id project-1, got %q", childScope.ProjectID)
	}
	if childScope.SourceID != "source-1" {
		t.Fatalf("expected child source id source-1, got %q", childScope.SourceID)
	}
}
