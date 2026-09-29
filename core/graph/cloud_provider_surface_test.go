package graph

import "testing"

func TestCloudProviderOperationsAreNotExposed(t *testing.T) {
	schema := NewExecutableSchema(Config{Resolvers: &Resolver{}}).Schema()

	for _, field := range []string{
		"CloudProviders",
		"CloudProvider",
		"DiscoveredConnections",
		"ProviderConnections",
		"LocalAWSProfiles",
		"AWSRegions",
		"AzureProviders",
		"AzureProvider",
		"AzureSubscriptions",
		"AzureRegions",
		"GCPProviders",
		"GCPProvider",
		"LocalGCPProjects",
		"GCPRegions",
	} {
		if schema.Query.Fields.ForName(field) != nil {
			t.Errorf("cloud provider query %s remains exposed", field)
		}
	}

	for _, field := range []string{
		"AddAWSProvider",
		"UpdateAWSProvider",
		"TestAWSCredentials",
		"GenerateRDSAuthToken",
		"AddAzureProvider",
		"UpdateAzureProvider",
		"TestAzureCredentials",
		"RefreshAzureProvider",
		"GenerateAzureADToken",
		"AddGCPProvider",
		"UpdateGCPProvider",
		"TestGCPCredentials",
		"RefreshGCPProvider",
		"GenerateCloudSQLIAMAuthToken",
		"RemoveCloudProvider",
		"TestCloudProvider",
		"RefreshCloudProvider",
	} {
		if schema.Mutation.Fields.ForName(field) != nil {
			t.Errorf("cloud provider mutation %s remains exposed", field)
		}
	}
}
