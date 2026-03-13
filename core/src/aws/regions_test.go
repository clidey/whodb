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

import (
	"testing"
)

func TestGetRegions_HasPartitions(t *testing.T) {
	regions := GetRegions()

	partitions := make(map[string]int)
	for _, r := range regions {
		if r.Partition == "" {
			t.Errorf("region %s has empty partition", r.ID)
		}
		partitions[r.Partition]++
	}

	if partitions[PartitionStandard] == 0 {
		t.Error("expected standard partition regions")
	}
	if partitions[PartitionGovCloud] != 2 {
		t.Errorf("expected 2 GovCloud regions, got %d", partitions[PartitionGovCloud])
	}
	if partitions[PartitionChina] != 2 {
		t.Errorf("expected 2 China regions, got %d", partitions[PartitionChina])
	}
}

func TestGetRegions_GovCloudRegions(t *testing.T) {
	regions := GetRegions()

	govRegions := make(map[string]bool)
	for _, r := range regions {
		if r.Partition == PartitionGovCloud {
			govRegions[r.ID] = true
		}
	}

	if !govRegions["us-gov-west-1"] {
		t.Error("expected us-gov-west-1 in GovCloud regions")
	}
	if !govRegions["us-gov-east-1"] {
		t.Error("expected us-gov-east-1 in GovCloud regions")
	}
}

func TestGetRegions_ChinaRegions(t *testing.T) {
	regions := GetRegions()

	chinaRegions := make(map[string]bool)
	for _, r := range regions {
		if r.Partition == PartitionChina {
			chinaRegions[r.ID] = true
		}
	}

	if !chinaRegions["cn-north-1"] {
		t.Error("expected cn-north-1 in China regions")
	}
	if !chinaRegions["cn-northwest-1"] {
		t.Error("expected cn-northwest-1 in China regions")
	}
}

func TestGetRegions_NoDuplicateIDs(t *testing.T) {
	regions := GetRegions()
	seen := make(map[string]bool)
	for _, r := range regions {
		if seen[r.ID] {
			t.Errorf("duplicate region ID: %s", r.ID)
		}
		seen[r.ID] = true
	}
}
