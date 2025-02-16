// Licensed to Clidey Limited under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Clidey Limited licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package clickhouse

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

const (
	portKey         = "Port"
	sslModeKey      = "SSL Mode"
	httpProtocolKey = "HTTP Protocol"
	readOnlyKey     = "Readonly"
	debugKey        = "Debug"
)

func DB(config *engine.PluginConfig) (*sql.DB, error) {
	port, err := strconv.Atoi(common.GetRecordValueOrDefault(config.Credentials.Advanced, portKey, "9000"))
	if err != nil {
		return nil, err
	}
	sslMode := common.GetRecordValueOrDefault(config.Credentials.Advanced, sslModeKey, "disable")
	httpProtocol := common.GetRecordValueOrDefault(config.Credentials.Advanced, httpProtocolKey, "disable")
	readOnly := common.GetRecordValueOrDefault(config.Credentials.Advanced, readOnlyKey, "disable")
	debug := common.GetRecordValueOrDefault(config.Credentials.Advanced, debugKey, "disable")

	auth := clickhouse.Auth{
		Database: config.Credentials.Database,
		Username: config.Credentials.Username,
		Password: config.Credentials.Password,
	}
	address := []string{net.JoinHostPort(config.Credentials.Hostname, strconv.Itoa(port))}
	options := &clickhouse.Options{
		Addr:             address,
		Auth:             auth,
		DialTimeout:      time.Second * 30,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	}

	if httpProtocol != "disable" {
		options.Protocol = clickhouse.HTTP
		options.Compression = &clickhouse.Compression{
			Method: clickhouse.CompressionGZIP,
		}
	}

	if debug != "disable" {
		options.Debug = true
	}
	if readOnly == "disable" {
		options.Settings = clickhouse.Settings{
			"max_execution_time": 60,
		}
	}
	if sslMode != "disable" {
		options.TLS = &tls.Config{InsecureSkipVerify: sslMode == "relaxed" || sslMode == "none"}
	}

	conn := clickhouse.OpenDB(options)

	conn.SetMaxOpenConns(5)
	conn.SetMaxOpenConns(5)
	conn.SetConnMaxLifetime(time.Hour)

	err = conn.PingContext(context.Background())
	if err != nil {
		return nil, err
	}
	return conn, err
}

func getTableColumns(conn driver.Conn, schema, table string) ([]engine.Record, error) {
	query := fmt.Sprintf("DESCRIBE TABLE %s.%s", schema, table)
	rows, err := conn.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []engine.Record
	for rows.Next() {
		var name, typ, defaultType, defaultExpression string
		var comment *string
		if err := rows.Scan(&name, &typ, &defaultType, &defaultExpression, &comment); err != nil {
			return nil, err
		}
		columns = append(columns, engine.Record{Key: name, Value: typ})
	}

	return columns, nil
}
