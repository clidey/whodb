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

package memcached

import (
	"strconv"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// UpdateStorageUnit updates a Memcached item's value and/or flags.
func (p *MemcachedPlugin) UpdateStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string, updatedColumns []string) (bool, error) {
	client, err := DB(config)
	if err != nil {
		log.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to connect to Memcached for update")
		return false, err
	}
	defer client.Close()

	// Get the current item to preserve unmodified fields
	current, err := client.Get(storageUnit)
	if err != nil {
		log.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to get current Memcached item for update")
		return false, err
	}
	if current == nil {
		return false, nil
	}

	item := &Item{
		Key:   storageUnit,
		Value: current.Value,
		Flags: current.Flags,
	}

	if val, ok := values["Value"]; ok {
		item.Value = []byte(val)
	}
	if flagStr, ok := values["Flags"]; ok {
		flags, err := strconv.ParseUint(flagStr, 10, 32)
		if err == nil {
			item.Flags = uint32(flags)
		}
	}

	if err := client.Replace(item); err != nil {
		log.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to update Memcached item")
		return false, err
	}

	return true, nil
}
