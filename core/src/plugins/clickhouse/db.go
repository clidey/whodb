package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

func DB(config *engine.PluginConfig) (driver.Conn, error) {
	port := common.GetRecordValueOrDefault(config.Credentials.Advanced, "Port", "9000")
	options := &clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", config.Credentials.Hostname, port)},
		Auth: clickhouse.Auth{
			Database: config.Credentials.Database,
			Username: config.Credentials.Username,
			Password: config.Credentials.Password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout:      time.Second * 30,
		MaxOpenConns:     5,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Hour,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	}

	return clickhouse.Open(options)
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
