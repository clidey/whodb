package settings

import "testing"

func TestUpdateSettingsTogglesMetricsEnabled(t *testing.T) {
	original := currentSettings
	t.Cleanup(func() {
		currentSettings = original
	})

	// Enable metrics from default false
	changed := UpdateSettings(MetricsEnabledField(true))
	if !changed {
		t.Fatalf("expected settings change when enabling metrics")
	}
	if !Get().MetricsEnabled {
		t.Fatalf("expected metrics to be enabled after update")
	}

	// Calling with same value should not report change
	if changed := UpdateSettings(MetricsEnabledField(true)); changed {
		t.Fatalf("expected no change when value remains the same")
	}

	// Disable metrics again
	changed = UpdateSettings(MetricsEnabledField(false))
	if !changed || Get().MetricsEnabled {
		t.Fatalf("expected metrics to be disabled and change reported")
	}
}
