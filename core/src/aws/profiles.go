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
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/clidey/whodb/core/src/log"
)

// LocalProfile represents an AWS profile discovered from local configuration.
type LocalProfile struct {
	Name      string
	Region    string
	Source    string
	IsDefault bool
}

// DiscoverLocalProfiles scans ~/.aws/credentials and ~/.aws/config for available profiles.
// It also checks environment variables for AWS credentials.
func DiscoverLocalProfiles() ([]LocalProfile, error) {
	profiles := make(map[string]*LocalProfile)

	if hasEnvCredentials() {
		envProfile := &LocalProfile{
			Name:      "environment",
			Source:    "environment",
			IsDefault: os.Getenv("AWS_PROFILE") == "",
		}
		if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
			envProfile.Region = region
		} else if region := os.Getenv("AWS_REGION"); region != "" {
			envProfile.Region = region
		}
		profiles["environment"] = envProfile
	}

	awsDir := getAWSConfigDir()

	credentialsPath := filepath.Join(awsDir, "credentials")
	credProfiles, err := parseINIFile(credentialsPath, "credentials", false)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Logger.Warnf("Failed to parse AWS credentials file %s: %v", credentialsPath, err)
		}
	} else {
		for name, profile := range credProfiles {
			profiles[name] = profile
		}
	}

	configPath := filepath.Join(awsDir, "config")
	configProfiles, err := parseINIFile(configPath, "config", true)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Logger.Warnf("Failed to parse AWS config file %s: %v", configPath, err)
		}
	} else {
		for name, profile := range configProfiles {
			if existing, ok := profiles[name]; ok {
				if existing.Region == "" && profile.Region != "" {
					existing.Region = profile.Region
				}
			} else {
				profiles[name] = profile
			}
		}
	}

	result := make([]LocalProfile, 0, len(profiles))
	for _, profile := range profiles {
		result = append(result, *profile)
	}

	return result, nil
}

func hasEnvCredentials() bool {
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	return accessKey != "" && secretKey != ""
}

func getAWSConfigDir() string {
	if dir := os.Getenv("AWS_CONFIG_FILE"); dir != "" {
		return filepath.Dir(dir)
	}
	if dir := os.Getenv("AWS_SHARED_CREDENTIALS_FILE"); dir != "" {
		return filepath.Dir(dir)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Logger.Warnf("Unable to determine home directory for AWS config: %v", err)
		return ""
	}
	return filepath.Join(homeDir, ".aws")
}

func parseINIFile(path, source string, isConfigFile bool) (map[string]*LocalProfile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	profiles := make(map[string]*LocalProfile)
	var currentProfile *LocalProfile

	sectionRegex := regexp.MustCompile(`^\s*\[([^\]]+)\]\s*$`)
	keyValueRegex := regexp.MustCompile(`^\s*([^=]+?)\s*=\s*(.+?)\s*$`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
			continue
		}

		if matches := sectionRegex.FindStringSubmatch(line); matches != nil {
			sectionName := matches[1]

			profileName := sectionName
			if isConfigFile && strings.HasPrefix(sectionName, "profile ") {
				profileName = strings.TrimPrefix(sectionName, "profile ")
			}

			currentProfile = &LocalProfile{
				Name:      profileName,
				Source:    source,
				IsDefault: profileName == "default",
			}
			profiles[profileName] = currentProfile
			continue
		}

		if currentProfile != nil {
			if matches := keyValueRegex.FindStringSubmatch(line); matches != nil {
				key := strings.ToLower(matches[1])
				value := matches[2]

				if key == "region" {
					currentProfile.Region = value
				}
			}
		}
	}

	return profiles, scanner.Err()
}
