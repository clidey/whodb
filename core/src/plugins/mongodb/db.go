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

package mongodb

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func DB(config *engine.PluginConfig) (*mongo.Client, error) {
	ctx := context.Background()
	port, err := strconv.Atoi(common.GetRecordValueOrDefault(config.Credentials.Advanced, "Port", "27017"))
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]interface{}{
			"hostname":  config.Credentials.Hostname,
			"portValue": common.GetRecordValueOrDefault(config.Credentials.Advanced, "Port", "27017"),
		}).Error("Failed to parse MongoDB port number")
		return nil, err
	}
	queryParams := common.GetRecordValueOrDefault(config.Credentials.Advanced, "URL Params", "")
	dnsEnabled, err := strconv.ParseBool(common.GetRecordValueOrDefault(config.Credentials.Advanced, "DNS Enabled", "false"))
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]interface{}{
			"hostname":        config.Credentials.Hostname,
			"dnsEnabledValue": common.GetRecordValueOrDefault(config.Credentials.Advanced, "DNS Enabled", "false"),
		}).Error("Failed to parse MongoDB DNS enabled flag")
		return nil, err
	}

	connectionURI := strings.Builder{}
	clientOptions := options.Client()

	if dnsEnabled {
		connectionURI.WriteString("mongodb+srv://")
		connectionURI.WriteString(fmt.Sprintf("%s/", config.Credentials.Hostname))
	} else {
		connectionURI.WriteString("mongodb://")
		connectionURI.WriteString(fmt.Sprintf("%s:%d/", config.Credentials.Hostname, port))
	}

	connectionURI.WriteString(config.Credentials.Database)
	connectionURI.WriteString(queryParams)

	clientOptions.ApplyURI(connectionURI.String())
	clientOptions.SetAuth(options.Credential{
		Username: url.QueryEscape(config.Credentials.Username),
		Password: url.QueryEscape(config.Credentials.Password),
	})

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]interface{}{
			"hostname":   config.Credentials.Hostname,
			"database":   config.Credentials.Database,
			"username":   config.Credentials.Username,
			"dnsEnabled": dnsEnabled,
			"port":       port,
		}).Error("Failed to connect to MongoDB")
		return nil, err
	}
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]interface{}{
			"hostname": config.Credentials.Hostname,
			"database": config.Credentials.Database,
			"username": config.Credentials.Username,
		}).Error("Failed to ping MongoDB server")
		return nil, err
	}
	return client, nil
}
