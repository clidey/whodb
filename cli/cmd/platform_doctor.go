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

package cmd

import (
	"context"
	"fmt"

	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var platformDoctorCmd = &cobra.Command{
	Use:           "platform",
	Short:         "Diagnose hosted WhoDB CLI and MCP readiness",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		format, err := output.ParseFormat(platformFormat)
		if err != nil {
			return err
		}
		report := runPlatformDoctor(ctx)
		if format == output.FormatJSON {
			if err := writeCommandJSON(cmd, report); err != nil {
				return err
			}
		} else if err := newCommandOutput(cmd, format, platformQuiet).WriteQueryResult(platformDoctorTable(report)); err != nil {
			return err
		}
		if !report.OK {
			return fmt.Errorf("hosted WhoDB doctor found %d failing check(s)", report.Failures)
		}
		return nil
	},
}

type platformDoctorReport struct {
	Host     string                `json:"host"`
	OK       bool                  `json:"ok"`
	Failures int                   `json:"failures"`
	Checks   []platformDoctorCheck `json:"checks"`
}

type platformDoctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func runPlatformDoctor(ctx context.Context) platformDoctorReport {
	report := platformDoctorReport{OK: true}
	add := func(name, status, detail string) {
		report.Checks = append(report.Checks, platformDoctorCheck{Name: name, Status: status, Detail: detail})
		if status == "fail" {
			report.OK = false
			report.Failures++
		}
	}
	normalizedHost, err := platform.NormalizeHost(platformHost)
	if err != nil {
		add("host", "fail", err.Error())
		return report
	}
	report.Host = normalizedHost
	add("host", "ok", normalizedHost)

	session, err := loadPlatformSession(ctx, platformHost)
	if err != nil {
		add("auth", "fail", err.Error())
		return report
	}
	add("auth", "ok", "local credentials loaded")

	user, err := session.Client.Me(ctx)
	if err != nil {
		add("identity", "fail", err.Error())
		return report
	}
	add("identity", "ok", user.Email)

	orgs, err := session.Client.Organizations(ctx)
	if err != nil {
		add("organizations", "fail", err.Error())
	} else {
		add("organizations", "ok", fmt.Sprintf("%d visible", len(orgs)))
	}

	selection, err := autoSelectPlatformWorkspaceWithOrgs(ctx, session.Client, &session.Host, orgs)
	if err != nil {
		add("workspace", "fail", err.Error())
	} else if session.Host.DefaultOrgID == "" || session.Host.DefaultProjectID == "" {
		add("workspace", "fail", "no selected organization/project; run whodb-cli use --org <org> --project <project>")
	} else {
		add("workspace", "ok", session.Host.DefaultOrgName+" / "+session.Host.DefaultProjectName)
	}

	manifest := manifestFromCache(session.Host.Manifest)
	if manifest == nil {
		manifest, err = refreshPlatformManifest(ctx, session.Config, &session.Host, session.Client)
		if err != nil {
			add("manifest", "fail", err.Error())
		} else {
			add("manifest", "ok", "refreshed "+manifest.PlatformVersion)
		}
	} else {
		add("manifest", "ok", "cached "+manifest.PlatformVersion)
	}
	if manifest != nil {
		capabilities := platformStatusCapabilities(manifest)
		missing := 0
		for _, capability := range capabilities {
			if !capability.Supported {
				missing++
			}
		}
		if missing > 0 {
			add("capabilities", "fail", fmt.Sprintf("%d required operation(s) missing", missing))
		} else {
			add("capabilities", "ok", "required operations available")
		}
	}
	if session.Host.DefaultProjectID != "" {
		session.Client.SetWorkspaceContext(session.Host.DefaultOrgID, session.Host.DefaultProjectID)
		if _, err := session.Client.ProjectSecrets(ctx, session.Host.DefaultProjectID); err != nil {
			add("project_read", "fail", err.Error())
		} else {
			add("project_read", "ok", "project-scoped requests work")
		}
	}
	if len(selection.Messages) > 0 {
		add("workspace_autoselect", "ok", selection.Messages[len(selection.Messages)-1])
	}
	return report
}

func platformDoctorTable(report platformDoctorReport) *output.QueryResult {
	rows := make([][]any, len(report.Checks))
	for i, check := range report.Checks {
		rows[i] = []any{check.Name, check.Status, check.Detail}
	}
	return tableResult([]string{"check", "status", "detail"}, rows)
}
