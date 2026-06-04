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

package src

import (
	"maps"

	"github.com/clidey/whodb/core/src/common/ssl"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/types"
)

// GetSourceProfiles returns saved and environment-defined profiles in the
// source-first shape consumed by the public API.
func GetSourceProfiles() []source.Profile {
	loginProfiles := GetLoginProfiles()
	profiles := make([]source.Profile, 0, len(loginProfiles))
	for i := range loginProfiles {
		profile := loginProfiles[i]
		id := GetLoginProfileId(i, profile)
		displayName := id
		if profile.Alias != "" {
			displayName = profile.Alias
		}
		if profile.CustomId != "" {
			id = profile.CustomId
		}

		sslConfigured := false
		if mode, ok := profile.Advanced[ssl.KeySSLMode]; ok && mode != "" && mode != string(ssl.SSLModeDisabled) {
			sslConfigured = true
		}

		profiles = append(profiles, source.Profile{
			ID:                   id,
			DisplayName:          displayName,
			SourceType:           profile.Type,
			Values:               displayValuesForProfile(profile),
			IsEnvironmentDefined: profile.Source == "environment",
			Source:               profile.Source,
			SSLConfigured:        sslConfigured,
		})
	}
	return profiles
}

// FindSourceProfile resolves a source profile and its full credentials by id.
func FindSourceProfile(id string) (*source.Profile, *source.Credentials, bool) {
	loginProfiles := GetLoginProfiles()
	for i := range loginProfiles {
		profile := loginProfiles[i]
		profileID := GetLoginProfileId(i, profile)
		if profile.CustomId != "" {
			profileID = profile.CustomId
		}
		if profileID != id {
			continue
		}

		displayName := profileID
		if profile.Alias != "" {
			displayName = profile.Alias
		}

		sourceProfile := &source.Profile{
			ID:                   profileID,
			DisplayName:          displayName,
			SourceType:           profile.Type,
			Values:               displayValuesForProfile(profile),
			IsEnvironmentDefined: profile.Source == "environment",
			Source:               profile.Source,
		}
		credentials := GetSourceCredentials(profile)
		credentials.ID = &sourceProfile.ID
		return sourceProfile, credentials, true
	}

	return nil, nil, false
}

// GetSourceCredentials converts stored login-profile credentials to the source-first shape.
func GetSourceCredentials(profile types.DatabaseCredentials) *source.Credentials {
	values := make(map[string]string, len(profile.Advanced)+4)
	if profile.Hostname != "" {
		values["Hostname"] = profile.Hostname
	}
	if profile.Username != "" {
		values["Username"] = profile.Username
	}
	if profile.Password != "" {
		values["Password"] = profile.Password
	}
	if profile.Database != "" {
		values["Database"] = profile.Database
	}
	if profile.Port != "" {
		values["Port"] = profile.Port
	}
	maps.Copy(values, profile.Advanced)
	return &source.Credentials{
		SourceType: profile.Type,
		Values:     values,
		IsProfile:  profile.IsProfile,
	}
}

func displayValuesForProfile(profile types.DatabaseCredentials) map[string]string {
	values := map[string]string{}
	if profile.Hostname != "" {
		values["Hostname"] = profile.Hostname
	}
	if profile.Database != "" {
		values["Database"] = profile.Database
	}
	if profile.Port != "" {
		values["Port"] = profile.Port
	}
	for key, value := range profile.Advanced {
		if key == ssl.KeySSLCACertContent || key == ssl.KeySSLClientCertContent || key == ssl.KeySSLClientKeyContent {
			continue
		}
		values[key] = value
	}
	return values
}
