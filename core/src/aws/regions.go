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

// Partition constants for AWS region grouping.
const (
	PartitionStandard = "aws"
	PartitionGovCloud = "aws-us-gov"
	PartitionChina    = "aws-cn"
)

// Region represents an AWS region with its partition.
type Region struct {
	ID          string
	Description string
	Partition   string
}

// GetRegions returns the list of AWS regions across all supported partitions.
// Based on https://www.aws-services.info/regions.html
func GetRegions() []Region {
	return []Region{
		// US regions
		{ID: "us-east-1", Description: "US East (N. Virginia)", Partition: PartitionStandard},
		{ID: "us-east-2", Description: "US East (Ohio)", Partition: PartitionStandard},
		{ID: "us-west-1", Description: "US West (N. California)", Partition: PartitionStandard},
		{ID: "us-west-2", Description: "US West (Oregon)", Partition: PartitionStandard},

		// Canada
		{ID: "ca-central-1", Description: "Canada (Central)", Partition: PartitionStandard},
		{ID: "ca-west-1", Description: "Canada West (Calgary)", Partition: PartitionStandard},

		// Mexico
		{ID: "mx-central-1", Description: "Mexico (Central)", Partition: PartitionStandard},

		// Europe
		{ID: "eu-west-1", Description: "Europe (Ireland)", Partition: PartitionStandard},
		{ID: "eu-west-2", Description: "Europe (London)", Partition: PartitionStandard},
		{ID: "eu-west-3", Description: "Europe (Paris)", Partition: PartitionStandard},
		{ID: "eu-central-1", Description: "Europe (Frankfurt)", Partition: PartitionStandard},
		{ID: "eu-central-2", Description: "Europe (Zurich)", Partition: PartitionStandard},
		{ID: "eu-north-1", Description: "Europe (Stockholm)", Partition: PartitionStandard},
		{ID: "eu-south-1", Description: "Europe (Milan)", Partition: PartitionStandard},
		{ID: "eu-south-2", Description: "Europe (Spain)", Partition: PartitionStandard},

		// Asia Pacific
		{ID: "ap-east-1", Description: "Asia Pacific (Hong Kong)", Partition: PartitionStandard},
		{ID: "ap-east-2", Description: "Asia Pacific (Taipei)", Partition: PartitionStandard},
		{ID: "ap-northeast-1", Description: "Asia Pacific (Tokyo)", Partition: PartitionStandard},
		{ID: "ap-northeast-2", Description: "Asia Pacific (Seoul)", Partition: PartitionStandard},
		{ID: "ap-northeast-3", Description: "Asia Pacific (Osaka)", Partition: PartitionStandard},
		{ID: "ap-south-1", Description: "Asia Pacific (Mumbai)", Partition: PartitionStandard},
		{ID: "ap-south-2", Description: "Asia Pacific (Hyderabad)", Partition: PartitionStandard},
		{ID: "ap-southeast-1", Description: "Asia Pacific (Singapore)", Partition: PartitionStandard},
		{ID: "ap-southeast-2", Description: "Asia Pacific (Sydney)", Partition: PartitionStandard},
		{ID: "ap-southeast-3", Description: "Asia Pacific (Jakarta)", Partition: PartitionStandard},
		{ID: "ap-southeast-4", Description: "Asia Pacific (Melbourne)", Partition: PartitionStandard},
		{ID: "ap-southeast-5", Description: "Asia Pacific (Malaysia)", Partition: PartitionStandard},
		{ID: "ap-southeast-6", Description: "Asia Pacific (New Zealand)", Partition: PartitionStandard},
		{ID: "ap-southeast-7", Description: "Asia Pacific (Thailand)", Partition: PartitionStandard},

		// South America
		{ID: "sa-east-1", Description: "South America (São Paulo)", Partition: PartitionStandard},

		// Middle East
		{ID: "me-south-1", Description: "Middle East (Bahrain)", Partition: PartitionStandard},
		{ID: "me-central-1", Description: "Middle East (UAE)", Partition: PartitionStandard},

		// Africa
		{ID: "af-south-1", Description: "Africa (Cape Town)", Partition: PartitionStandard},

		// Israel
		{ID: "il-central-1", Description: "Israel (Tel Aviv)", Partition: PartitionStandard},

		// GovCloud
		{ID: "us-gov-west-1", Description: "AWS GovCloud (US-West)", Partition: PartitionGovCloud},
		{ID: "us-gov-east-1", Description: "AWS GovCloud (US-East)", Partition: PartitionGovCloud},

		// China
		{ID: "cn-north-1", Description: "China (Beijing)", Partition: PartitionChina},
		{ID: "cn-northwest-1", Description: "China (Ningxia)", Partition: PartitionChina},
	}
}
