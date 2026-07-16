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

package gcp

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2/google"

	"github.com/clidey/whodb/core/src/log"
)

// GenerateCloudSQLIAMAuthToken generates an OAuth2 access token for Cloud SQL IAM
// database authentication. The access token is used directly as the database password.
//
// If serviceAccountKeyPath is non-empty, credentials are loaded from that file.
// Otherwise, Application Default Credentials are used.
//
// See: https://cloud.google.com/sql/docs/mysql/iam-authentication
func GenerateCloudSQLIAMAuthToken(ctx context.Context, serviceAccountKeyPath, username string) (string, error) {
	log.Debugf("Cloud SQL IAM Auth: generating token for user=%s", username)

	scopes := []string{"https://www.googleapis.com/auth/sqlservice.login"}

	var creds *google.Credentials
	var err error

	if serviceAccountKeyPath != "" {
		data, readErr := os.ReadFile(serviceAccountKeyPath) // #nosec G304 -- reading the configured service-account key file is this function's purpose.
		if readErr != nil {
			return "", fmt.Errorf("failed to read service account key file: %w", readErr)
		}
		creds, err = google.CredentialsFromJSONWithTypeAndParams(ctx, data, google.ServiceAccount, google.CredentialsParams{Scopes: scopes})
	} else {
		creds, err = google.FindDefaultCredentials(ctx, scopes...)
	}

	if err != nil {
		log.Errorf("Cloud SQL IAM Auth: failed to find credentials: %v", err)
		return "", fmt.Errorf("failed to find GCP credentials for Cloud SQL IAM auth: %w", err)
	}

	token, err := creds.TokenSource.Token()
	if err != nil {
		log.Errorf("Cloud SQL IAM Auth: failed to obtain token: %v", err)
		return "", HandleGCPError(err)
	}

	log.Debugf("Cloud SQL IAM Auth: token generated successfully for user=%s (token length=%d)", username, len(token.AccessToken))
	return token.AccessToken, nil
}
