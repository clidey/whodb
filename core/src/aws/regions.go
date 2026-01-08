/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package aws

type Region struct {
	ID          string
	Description string
}

// GetRegions returns the list of AWS regions.
// Based on https://www.aws-services.info/regions.html
// Excludes China (cn-*), GovCloud (us-gov-*), and Sovereign Cloud (eusc-*) partitions.
func GetRegions() []Region {
	return []Region{
		// US regions
		{ID: "us-east-1", Description: "US East (N. Virginia)"},
		{ID: "us-east-2", Description: "US East (Ohio)"},
		{ID: "us-west-1", Description: "US West (N. California)"},
		{ID: "us-west-2", Description: "US West (Oregon)"},

		// Canada
		{ID: "ca-central-1", Description: "Canada (Central)"},
		{ID: "ca-west-1", Description: "Canada West (Calgary)"},

		// Mexico
		{ID: "mx-central-1", Description: "Mexico (Central)"},

		// Europe
		{ID: "eu-west-1", Description: "Europe (Ireland)"},
		{ID: "eu-west-2", Description: "Europe (London)"},
		{ID: "eu-west-3", Description: "Europe (Paris)"},
		{ID: "eu-central-1", Description: "Europe (Frankfurt)"},
		{ID: "eu-central-2", Description: "Europe (Zurich)"},
		{ID: "eu-north-1", Description: "Europe (Stockholm)"},
		{ID: "eu-south-1", Description: "Europe (Milan)"},
		{ID: "eu-south-2", Description: "Europe (Spain)"},

		// Asia Pacific
		{ID: "ap-east-1", Description: "Asia Pacific (Hong Kong)"},
		{ID: "ap-east-2", Description: "Asia Pacific (Taipei)"},
		{ID: "ap-northeast-1", Description: "Asia Pacific (Tokyo)"},
		{ID: "ap-northeast-2", Description: "Asia Pacific (Seoul)"},
		{ID: "ap-northeast-3", Description: "Asia Pacific (Osaka)"},
		{ID: "ap-south-1", Description: "Asia Pacific (Mumbai)"},
		{ID: "ap-south-2", Description: "Asia Pacific (Hyderabad)"},
		{ID: "ap-southeast-1", Description: "Asia Pacific (Singapore)"},
		{ID: "ap-southeast-2", Description: "Asia Pacific (Sydney)"},
		{ID: "ap-southeast-3", Description: "Asia Pacific (Jakarta)"},
		{ID: "ap-southeast-4", Description: "Asia Pacific (Melbourne)"},
		{ID: "ap-southeast-5", Description: "Asia Pacific (Malaysia)"},
		{ID: "ap-southeast-6", Description: "Asia Pacific (New Zealand)"},
		{ID: "ap-southeast-7", Description: "Asia Pacific (Thailand)"},

		// South America
		{ID: "sa-east-1", Description: "South America (São Paulo)"},

		// Middle East
		{ID: "me-south-1", Description: "Middle East (Bahrain)"},
		{ID: "me-central-1", Description: "Middle East (UAE)"},

		// Africa
		{ID: "af-south-1", Description: "Africa (Cape Town)"},

		// Israel
		{ID: "il-central-1", Description: "Israel (Tel Aviv)"},
	}
}
