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
	"fmt"
	"testing"
)

func TestParseDockerOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []DetectedContainer
	}{
		{
			name: "postgres container",
			input: `{"ID":"abc123","Image":"postgres:16","Names":"my-pg","Ports":"0.0.0.0:5432->5432/tcp, :::5432->5432/tcp","State":"running"}
`,
			expected: []DetectedContainer{
				{Name: "my-pg", Type: "Postgres", Port: 5432, Image: "postgres:16", ContainerID: "abc123"},
			},
		},
		{
			name: "mysql on custom port",
			input: `{"ID":"def456","Image":"mysql:8.0","Names":"test-mysql","Ports":"0.0.0.0:13306->3306/tcp","State":"running"}
`,
			expected: []DetectedContainer{
				{Name: "test-mysql", Type: "MySQL", Port: 13306, Image: "mysql:8.0", ContainerID: "def456"},
			},
		},
		{
			name: "mariadb container",
			input: `{"ID":"ghi789","Image":"mariadb:10","Names":"my-maria","Ports":"0.0.0.0:3306->3306/tcp","State":"running"}
`,
			expected: []DetectedContainer{
				{Name: "my-maria", Type: "MariaDB", Port: 3306, Image: "mariadb:10", ContainerID: "ghi789"},
			},
		},
		{
			name: "mongodb container",
			input: `{"ID":"jkl012","Image":"mongo:7","Names":"my-mongo","Ports":"0.0.0.0:27017->27017/tcp","State":"running"}
`,
			expected: []DetectedContainer{
				{Name: "my-mongo", Type: "MongoDB", Port: 27017, Image: "mongo:7", ContainerID: "jkl012"},
			},
		},
		{
			name: "redis container",
			input: `{"ID":"mno345","Image":"redis:7-alpine","Names":"cache","Ports":"0.0.0.0:6379->6379/tcp","State":"running"}
`,
			expected: []DetectedContainer{
				{Name: "cache", Type: "Redis", Port: 6379, Image: "redis:7-alpine", ContainerID: "mno345"},
			},
		},
		{
			name: "clickhouse container",
			input: `{"ID":"pqr678","Image":"clickhouse/clickhouse-server:latest","Names":"my-ch","Ports":"0.0.0.0:9000->9000/tcp, 0.0.0.0:8123->8123/tcp","State":"running"}
`,
			expected: []DetectedContainer{
				{Name: "my-ch", Type: "ClickHouse", Port: 9000, Image: "clickhouse/clickhouse-server:latest", ContainerID: "pqr678"},
			},
		},
		{
			name: "elasticsearch from docker.elastic.co",
			input: `{"ID":"stu901","Image":"docker.elastic.co/elasticsearch/elasticsearch:8.12.0","Names":"my-es","Ports":"0.0.0.0:9200->9200/tcp, 0.0.0.0:9300->9300/tcp","State":"running"}
`,
			expected: []DetectedContainer{
				{Name: "my-es", Type: "ElasticSearch", Port: 9200, Image: "docker.elastic.co/elasticsearch/elasticsearch:8.12.0", ContainerID: "stu901"},
			},
		},
		{
			name:     "non-running container is skipped",
			input:    `{"ID":"xyz","Image":"postgres:16","Names":"stopped-pg","Ports":"0.0.0.0:5432->5432/tcp","State":"exited"}`,
			expected: nil,
		},
		{
			name:     "unknown image is skipped",
			input:    `{"ID":"xyz","Image":"nginx:latest","Names":"web","Ports":"0.0.0.0:80->80/tcp","State":"running"}`,
			expected: nil,
		},
		{
			name:     "empty output",
			input:    "",
			expected: nil,
		},
		{
			name:     "invalid json is skipped",
			input:    `not json at all`,
			expected: nil,
		},
		{
			name: "multiple containers",
			input: `{"ID":"aaa","Image":"postgres:16","Names":"pg1","Ports":"0.0.0.0:5432->5432/tcp","State":"running"}
{"ID":"bbb","Image":"redis:7","Names":"redis1","Ports":"0.0.0.0:6379->6379/tcp","State":"running"}
{"ID":"ccc","Image":"nginx:latest","Names":"web","Ports":"0.0.0.0:80->80/tcp","State":"running"}
`,
			expected: []DetectedContainer{
				{Name: "pg1", Type: "Postgres", Port: 5432, Image: "postgres:16", ContainerID: "aaa"},
				{Name: "redis1", Type: "Redis", Port: 6379, Image: "redis:7", ContainerID: "bbb"},
			},
		},
		{
			name:     "no port mapping means container is skipped",
			input:    `{"ID":"xyz","Image":"postgres:16","Names":"no-ports","Ports":"","State":"running"}`,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDockerOutput([]byte(tt.input))
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d containers, got %d: %+v", len(tt.expected), len(result), result)
			}
			for i, c := range result {
				exp := tt.expected[i]
				if c.Name != exp.Name || c.Type != exp.Type || c.Port != exp.Port || c.Image != exp.Image || c.ContainerID != exp.ContainerID {
					t.Errorf("container %d mismatch:\n  got:  %+v\n  want: %+v", i, c, exp)
				}
			}
		})
	}
}

func TestExtractHostPort(t *testing.T) {
	tests := []struct {
		ports    string
		dbType   string
		expected int
	}{
		{"0.0.0.0:5432->5432/tcp", "Postgres", 5432},
		{"0.0.0.0:5432->5432/tcp, :::5432->5432/tcp", "Postgres", 5432},
		{"0.0.0.0:15432->5432/tcp", "Postgres", 15432},
		{"0.0.0.0:3306->3306/tcp", "MySQL", 3306},
		{"0.0.0.0:13306->3306/tcp", "MariaDB", 13306},
		{"0.0.0.0:9200->9200/tcp, 0.0.0.0:9300->9300/tcp", "ElasticSearch", 9200},
		{"", "Postgres", 0},
		{"0.0.0.0:80->80/tcp", "Postgres", 0},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%s", tt.dbType, tt.ports), func(t *testing.T) {
			result := extractHostPort(tt.ports, tt.dbType)
			if result != tt.expected {
				t.Errorf("extractHostPort(%q, %q) = %d, want %d", tt.ports, tt.dbType, result, tt.expected)
			}
		})
	}
}

func TestMatchImageType(t *testing.T) {
	tests := []struct {
		image    string
		expected string
	}{
		{"postgres:16", "Postgres"},
		{"postgres", "Postgres"},
		{"mysql:8.0", "MySQL"},
		{"mariadb:10", "MariaDB"},
		{"mongo:7", "MongoDB"},
		{"mongodb/mongodb-community-server:latest", "MongoDB"},
		{"redis:7-alpine", "Redis"},
		{"clickhouse/clickhouse-server:latest", "ClickHouse"},
		{"docker.elastic.co/elasticsearch/elasticsearch:8.12.0", "ElasticSearch"},
		{"elasticsearch:7.17", "ElasticSearch"},
		{"duckdb/duckdb:latest", "DuckDB"},
		{"nginx:latest", ""},
		{"library/postgres:16", "Postgres"},
	}

	for _, tt := range tests {
		t.Run(tt.image, func(t *testing.T) {
			result := matchImageType(tt.image)
			if result != tt.expected {
				t.Errorf("matchImageType(%q) = %q, want %q", tt.image, result, tt.expected)
			}
		})
	}
}

type fakeRunner struct {
	lookPathErr error
	output      []byte
	runErr      error
}

func (f fakeRunner) lookPath(string) (string, error) {
	if f.lookPathErr != nil {
		return "", f.lookPathErr
	}
	return "/usr/bin/docker", nil
}

func (f fakeRunner) run(string, ...string) ([]byte, error) {
	return f.output, f.runErr
}

func TestDetectContainersWithRunner_DockerNotFound(t *testing.T) {
	runner := fakeRunner{lookPathErr: fmt.Errorf("not found")}
	result := detectContainersWithRunner(runner)
	if result != nil {
		t.Errorf("expected nil when docker not found, got %+v", result)
	}
}

func TestDetectContainersWithRunner_DockerError(t *testing.T) {
	runner := fakeRunner{runErr: fmt.Errorf("docker error")}
	result := detectContainersWithRunner(runner)
	if result != nil {
		t.Errorf("expected nil on docker error, got %+v", result)
	}
}

func TestDetectContainersWithRunner_Success(t *testing.T) {
	output := `{"ID":"abc123","Image":"postgres:16","Names":"my-pg","Ports":"0.0.0.0:5432->5432/tcp","State":"running"}
`
	runner := fakeRunner{output: []byte(output)}
	result := detectContainersWithRunner(runner)
	if len(result) != 1 {
		t.Fatalf("expected 1 container, got %d", len(result))
	}
	if result[0].Type != "Postgres" || result[0].Port != 5432 {
		t.Errorf("unexpected container: %+v", result[0])
	}
}
