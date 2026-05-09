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

package dbcatalog

import (
	"testing"

	"github.com/clidey/whodb/core/src/common/ssl"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/source"
)

func TestFindReturnsAliasPluginType(t *testing.T) {
	entry, ok := Find("FerretDB")
	if !ok {
		t.Fatal("expected FerretDB catalog entry")
	}

	if entry.PluginType != engine.DatabaseType_MongoDB {
		t.Fatalf("expected FerretDB to resolve to MongoDB plugin, got %q", entry.PluginType)
	}
}

func TestFindReturnsPromotedQuestDBPluginType(t *testing.T) {
	entry, ok := Find("QuestDB")
	if !ok {
		t.Fatal("expected QuestDB catalog entry")
	}

	if entry.PluginType != engine.DatabaseType_QuestDB {
		t.Fatalf("expected QuestDB to resolve to its own plugin, got %q", entry.PluginType)
	}
}

func TestFindReturnsPromotedYugabyteDBPluginType(t *testing.T) {
	entry, ok := Find("YugabyteDB")
	if !ok {
		t.Fatal("expected YugabyteDB catalog entry")
	}

	if entry.PluginType != engine.DatabaseType_YugabyteDB {
		t.Fatalf("expected YugabyteDB to resolve to its own plugin, got %q", entry.PluginType)
	}
}

func TestDefaultPortUsesCatalogOverrides(t *testing.T) {
	port, ok := DefaultPort("QuestDB")
	if !ok {
		t.Fatal("expected QuestDB default port")
	}

	if port != 8812 {
		t.Fatalf("expected QuestDB port 8812, got %d", port)
	}
}

func TestManagedServiceEntryRetainsFlags(t *testing.T) {
	entry, ok := Find("ElastiCache")
	if !ok {
		t.Fatal("expected ElastiCache catalog entry")
	}

	if !entry.IsAWSManaged {
		t.Fatal("expected ElastiCache to be marked as AWS managed")
	}

	if entry.Extra["TLS"].DefaultValue != "true" {
		t.Fatalf("expected ElastiCache TLS default, got %q", entry.Extra["TLS"].DefaultValue)
	}
}

func TestRegisterClonesEntriesAndReturnedValuesAreDefensiveCopies(t *testing.T) {
	originalRegisteredCatalog := registeredCatalog
	registeredCatalog = nil
	t.Cleanup(func() {
		registeredCatalog = originalRegisteredCatalog
	})

	entry := ConnectableDatabase{
		ID:         engine.DatabaseType("CustomDB"),
		Label:      "Custom DB",
		PluginType: engine.DatabaseType_Postgres,
		Extra: map[string]source.ConnectionExtraField{
			"Port": {
				DefaultValue:   "15432",
				Kind:           source.ConnectionFieldKindText,
				LabelKey:       "advancedFields.customPort",
				PlaceholderKey: "enterCustomPort",
			},
		},
		SSLModes: []source.SSLModeInfo{
			{Value: string(ssl.SSLModeRequired), Label: "Required", Description: "Require TLS"},
		},
	}

	Register(entry)

	entry.Extra["Port"] = source.ConnectionExtraField{DefaultValue: "9999"}
	entry.SSLModes[0].Label = "Mutated"

	found, ok := Find("CustomDB")
	if !ok {
		t.Fatal("expected custom database to be found after registration")
	}
	if found.Extra["Port"].DefaultValue != "15432" {
		t.Fatalf("expected Register to clone entry.Extra, got %#v", found.Extra)
	}
	if found.Extra["Port"].Kind != source.ConnectionFieldKindText {
		t.Fatalf("expected Register to clone entry.Extra, got %#v", found.Extra)
	}
	if found.SSLModes[0].Label != "Required" {
		t.Fatalf("expected Register to clone entry.SSLModes, got %#v", found.SSLModes)
	}

	found.Extra["Port"] = source.ConnectionExtraField{DefaultValue: "1111", Kind: source.ConnectionFieldKindBoolean}
	found.SSLModes[0].Label = "Changed"

	refetched, ok := Find("CustomDB")
	if !ok {
		t.Fatal("expected custom database to be refetchable")
	}
	if refetched.Extra["Port"].DefaultValue != "15432" {
		t.Fatalf("expected Find to return defensive copies of Extra, got %#v", refetched.Extra)
	}
	if refetched.Extra["Port"].Kind != source.ConnectionFieldKindText {
		t.Fatalf("expected Find to return defensive copies of Extra, got %#v", refetched.Extra)
	}
	if refetched.SSLModes[0].Label != "Required" {
		t.Fatalf("expected Find to return defensive copies of SSL modes, got %#v", refetched.SSLModes)
	}

	all := All()
	for i := range all {
		if all[i].ID != engine.DatabaseType("CustomDB") {
			continue
		}
		all[i].Extra["Port"] = source.ConnectionExtraField{DefaultValue: "2222", Kind: source.ConnectionFieldKindBoolean}
		all[i].SSLModes[0].Label = "Changed Again"
		break
	}

	refetched, ok = Find("CustomDB")
	if !ok {
		t.Fatal("expected custom database to still be available after All mutation")
	}
	if refetched.Extra["Port"].DefaultValue != "15432" {
		t.Fatalf("expected All to return defensive copies of Extra, got %#v", refetched.Extra)
	}
	if refetched.Extra["Port"].Kind != source.ConnectionFieldKindText {
		t.Fatalf("expected All to return defensive copies of Extra, got %#v", refetched.Extra)
	}
	if refetched.SSLModes[0].Label != "Required" {
		t.Fatalf("expected All to return defensive copies of SSL modes, got %#v", refetched.SSLModes)
	}
}

func TestDefaultPortReturnsFalseWhenCatalogPortIsInvalid(t *testing.T) {
	originalRegisteredCatalog := registeredCatalog
	registeredCatalog = nil
	t.Cleanup(func() {
		registeredCatalog = originalRegisteredCatalog
	})

	const customID = engine.DatabaseType("CustomPortDB")
	Register(ConnectableDatabase{
		ID:         customID,
		Label:      "Custom Port DB",
		PluginType: customID,
		Extra:      map[string]source.ConnectionExtraField{"Port": {DefaultValue: "invalid"}},
	})

	if port, ok := DefaultPort(string(customID)); ok {
		t.Fatalf("expected invalid catalog port to be rejected, got %d", port)
	}
}
