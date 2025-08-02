/*
 * Copyright 2025 Clidey, Inc.
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

package settings

type Settings struct {
	MetricsEnabled bool `json:"metricsEnabled"`
	PerformanceMonitoringEnabled bool `json:"performanceMonitoringEnabled"`
	PerformanceMetricsConfig map[string]bool `json:"performanceMetricsConfig"`
}

type ISettingsField interface {
	Apply(*Settings) bool
}

type MetricsEnabledField bool
type PerformanceMonitoringEnabledField bool
type PerformanceMetricsConfigField map[string]bool

var currentSettings = Settings{
	MetricsEnabled: false,
	PerformanceMonitoringEnabled: false,
	PerformanceMetricsConfig: map[string]bool{
		"query_latency":      true,
		"query_count":        true,
		"connection_count":   true,
		"error_count":        true,
		"cpu_usage":          false,
		"memory_usage":       false,
		"disk_io":            false,
		"cache_hit_ratio":    false,
		"transaction_count":  false,
		"lock_wait_time":     false,
	},
}

func Get() Settings {
	return currentSettings
}

func (m MetricsEnabledField) Apply(s *Settings) bool {
	if s.MetricsEnabled != bool(m) {
		s.MetricsEnabled = bool(m)
		return true
	}
	return false
}

func (p PerformanceMonitoringEnabledField) Apply(s *Settings) bool {
	if s.PerformanceMonitoringEnabled != bool(p) {
		s.PerformanceMonitoringEnabled = bool(p)
		return true
	}
	return false
}

func (p PerformanceMetricsConfigField) Apply(s *Settings) bool {
	changed := false
	for key, value := range p {
		if s.PerformanceMetricsConfig[key] != value {
			s.PerformanceMetricsConfig[key] = value
			changed = true
		}
	}
	return changed
}

// UpdateSettings todo: this isn't a good idea when your settings are larger. you'll end up pushing more data than is needed back and forth. refactor so it's more flexible
func UpdateSettings(fields ...ISettingsField) bool {
	changed := false
	for _, field := range fields {
		if field.Apply(&currentSettings) {
			changed = true
		}
	}
	return changed
}
