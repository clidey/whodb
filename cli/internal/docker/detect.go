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

package docker

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// DetectedContainer represents a running Docker container that hosts a known database.
type DetectedContainer struct {
	Name        string
	Type        string // WhoDB plugin name, e.g. "Postgres", "MySQL"
	Port        int    // Host port mapped to the database port
	Image       string
	ContainerID string
}

// imagePatterns maps regex patterns for Docker image names to WhoDB database type strings.
var imagePatterns = []struct {
	pattern *regexp.Regexp
	dbType  string
}{
	{regexp.MustCompile(`(?i)^postgres`), "Postgres"},
	{regexp.MustCompile(`(?i)^mysql`), "MySQL"},
	{regexp.MustCompile(`(?i)^mariadb`), "MariaDB"},
	{regexp.MustCompile(`(?i)^mongo`), "MongoDB"},
	{regexp.MustCompile(`(?i)^redis`), "Redis"},
	{regexp.MustCompile(`(?i)^clickhouse`), "ClickHouse"},
	{regexp.MustCompile(`(?i)^elasticsearch`), "ElasticSearch"},
	{regexp.MustCompile(`(?i)^docker\.elastic\.co/elasticsearch`), "ElasticSearch"},
	{regexp.MustCompile(`(?i)^duckdb`), "DuckDB"},
}

// dockerPSEntry represents the JSON output of a single line from `docker ps --format '{{json .}}'`.
type dockerPSEntry struct {
	ID    string `json:"ID"`
	Image string `json:"Image"`
	Names string `json:"Names"`
	Ports string `json:"Ports"`
	State string `json:"State"`
}

// DetectContainers discovers running Docker containers that match known database images.
// It shells out to the docker CLI rather than importing the Docker API client.
// Returns an empty slice if docker is not installed or returns an error.
func DetectContainers() []DetectedContainer {
	return detectContainersWithRunner(defaultRunner{})
}

// commandRunner abstracts command execution for testing.
type commandRunner interface {
	lookPath(file string) (string, error)
	run(name string, args ...string) ([]byte, error)
}

// defaultRunner uses the real OS exec.
type defaultRunner struct{}

func (defaultRunner) lookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (defaultRunner) run(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

// detectContainersWithRunner runs detection using the provided command runner.
func detectContainersWithRunner(runner commandRunner) []DetectedContainer {
	dockerPath, err := runner.lookPath("docker")
	if err != nil {
		return nil
	}

	out, err := runner.run(dockerPath, "ps", "--format", "{{json .}}")
	if err != nil {
		return nil
	}

	return parseDockerOutput(out)
}

// parseDockerOutput parses the line-delimited JSON output from docker ps.
func parseDockerOutput(data []byte) []DetectedContainer {
	var containers []DetectedContainer
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry dockerPSEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.State != "running" {
			continue
		}
		dbType := matchImageType(entry.Image)
		if dbType == "" {
			continue
		}
		port := extractHostPort(entry.Ports, dbType)
		if port == 0 {
			continue
		}
		name := entry.Names
		if name == "" {
			name = entry.ID[:12]
		}
		containers = append(containers, DetectedContainer{
			Name:        name,
			Type:        dbType,
			Port:        port,
			Image:       entry.Image,
			ContainerID: entry.ID,
		})
	}
	return containers
}

// matchImageType returns the WhoDB database type for a Docker image name,
// or an empty string if the image is not recognized.
func matchImageType(image string) string {
	// Strip registry prefix (e.g. "docker.io/library/postgres:16" -> "postgres:16")
	// but keep multi-segment paths like "docker.elastic.co/elasticsearch/elasticsearch"
	parts := strings.Split(image, "/")
	// Try matching the full image name first (for multi-segment images)
	for _, p := range imagePatterns {
		if p.pattern.MatchString(image) {
			return p.dbType
		}
	}
	// Then try the last segment (covers "library/postgres" -> "postgres")
	last := parts[len(parts)-1]
	// Remove the tag
	if idx := strings.Index(last, ":"); idx != -1 {
		last = last[:idx]
	}
	for _, p := range imagePatterns {
		if p.pattern.MatchString(last) {
			return p.dbType
		}
	}
	return ""
}

// defaultContainerPort returns the standard internal port for a given database type.
func defaultContainerPort(dbType string) int {
	switch dbType {
	case "Postgres":
		return 5432
	case "MySQL", "MariaDB":
		return 3306
	case "MongoDB":
		return 27017
	case "Redis":
		return 6379
	case "ClickHouse":
		return 9000
	case "ElasticSearch":
		return 9200
	default:
		return 0
	}
}

// extractHostPort parses Docker port mappings like "0.0.0.0:5432->5432/tcp, :::5432->5432/tcp"
// and returns the host port that maps to the expected container port for the database type.
func extractHostPort(ports string, dbType string) int {
	containerPort := defaultContainerPort(dbType)
	if containerPort == 0 {
		return 0
	}
	containerPortStr := strconv.Itoa(containerPort)
	for _, mapping := range strings.Split(ports, ", ") {
		mapping = strings.TrimSpace(mapping)
		// Format: "host:port->container/proto" or "0.0.0.0:host->container/proto"
		arrowIdx := strings.Index(mapping, "->")
		if arrowIdx == -1 {
			continue
		}
		right := mapping[arrowIdx+2:]
		// Strip protocol
		if slashIdx := strings.Index(right, "/"); slashIdx != -1 {
			right = right[:slashIdx]
		}
		if right != containerPortStr {
			continue
		}
		left := mapping[:arrowIdx]
		// The host port is the last colon-separated segment on the left side
		colonIdx := strings.LastIndex(left, ":")
		if colonIdx == -1 {
			continue
		}
		hostPortStr := left[colonIdx+1:]
		hostPort, err := strconv.Atoi(hostPortStr)
		if err != nil {
			continue
		}
		return hostPort
	}
	return 0
}
