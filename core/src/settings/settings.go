package settings

import "github.com/clidey/whodb/core/src/highlight"

type Settings struct {
	MetricsEnabled bool `json:"metricsEnabled"`
}

type ISettingsField interface {
	Apply(*Settings) bool
}

type MetricsEnabledField bool

var currentSettings = Settings{MetricsEnabled: true}

func Get() Settings {
	return currentSettings
}

func (m MetricsEnabledField) Apply(s *Settings) bool {
	if s.MetricsEnabled != bool(m) {
		s.MetricsEnabled = bool(m)
		if s.MetricsEnabled {
			highlight.InitializeHighlight()
		} else {
			highlight.StopHighlight()
		}
		return true
	}
	return false
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
