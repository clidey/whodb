package platform

import (
	"context"
	"strings"
	"testing"
)

func TestPlatformQuerySpecsAreBoundedAndScoped(t *testing.T) {
	for operation, query := range platformQuerySpecs {
		if !strings.Contains(query, operation) {
			t.Errorf("%s query does not select its operation", operation)
		}
		if strings.Contains(query, "IntrospectionQuery") || strings.Contains(query, "__schema") {
			t.Errorf("%s query exposes GraphQL introspection", operation)
		}
	}
}

func TestPlatformQueryRejectsUnknownOperation(t *testing.T) {
	client := &Client{}
	if _, err := client.PlatformQuery(context.Background(), "UnknownOperation", nil); err == nil {
		t.Fatal("PlatformQuery() accepted an unknown operation")
	}
}
