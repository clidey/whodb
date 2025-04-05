package clickhouse

import (
	"context"
	"crypto/tls"
	"github.com/ClickHouse/clickhouse-go/v2"
	"gorm.io/gorm"
	"net"
	"strconv"
	"time"

	"github.com/clidey/whodb/core/src/engine"
	gorm_clickhouse "gorm.io/driver/clickhouse"
)

func (p *ClickHousePlugin) DB(config *engine.PluginConfig) (*gorm.DB, error) {
	connectionInput, err := p.ParseConnectionConfig(config)
	if err != nil {
		return nil, err
	}

	auth := clickhouse.Auth{
		Database: connectionInput.Database,
		Username: connectionInput.Username,
		Password: connectionInput.Password,
	}

	address := []string{net.JoinHostPort(connectionInput.Hostname, strconv.Itoa(connectionInput.Port))}
	options := &clickhouse.Options{
		Addr:             address,
		Auth:             auth,
		DialTimeout:      time.Second * 30,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	}

	if connectionInput.HTTPProtocol != "disable" {
		options.Protocol = clickhouse.HTTP
		options.Compression = &clickhouse.Compression{
			Method: clickhouse.CompressionGZIP,
		}
	}

	if connectionInput.Debug != "disable" {
		options.Debug = true
	}
	if connectionInput.ReadOnly == "disable" {
		options.Settings = clickhouse.Settings{
			"max_execution_time": 60,
		}
	}
	if connectionInput.SSLMode != "disable" {
		options.TLS = &tls.Config{InsecureSkipVerify: connectionInput.SSLMode == "relaxed" || connectionInput.SSLMode == "none"}
	}

	conn := clickhouse.OpenDB(options)

	conn.SetMaxOpenConns(5)
	conn.SetMaxOpenConns(5)
	conn.SetConnMaxLifetime(time.Hour)

	err = conn.PingContext(context.Background())
	if err != nil {
		return nil, err
	}

	return gorm.Open(gorm_clickhouse.New(gorm_clickhouse.Config{
		Conn: conn,
	}))
}
