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
	"context"
	"fmt"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/clidey/whodb/core/src/log"
)

// GenerateRDSAuthToken builds a short-lived IAM authentication token for RDS.
// The token acts as a password and is valid for ~15 minutes.
func GenerateRDSAuthToken(ctx context.Context, cfg awssdk.Config, endpoint string, port int, region, username string) (string, error) {
	addr := fmt.Sprintf("%s:%d", endpoint, port)
	log.Infof("RDS IAM Auth: generating token for %s (region=%s, user=%s)", addr, region, username)
	token, err := auth.BuildAuthToken(ctx, addr, region, username, cfg.Credentials)
	if err != nil {
		log.Errorf("RDS IAM Auth: token generation failed for %s: %v", addr, err)
		return "", HandleAWSError(err)
	}
	log.Infof("RDS IAM Auth: token generated successfully for %s (token length=%d)", addr, len(token))
	return token, nil
}
