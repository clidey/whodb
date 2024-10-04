package mysql

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"os"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	mysql_driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	portKey                    = "Port"
	charsetKey                 = "Charset"
	parseTimeKey               = "Parse Time"
	locKey                     = "Loc"
	allowClearTextPasswordsKey = "Allow clear text passwords"
	sslModeKey                 = "SSL Mode"
	sslCAKey                   = "SSL CA"
	sslCertKey                 = "SSL Cert"
	sslKeyKey                  = "SSL Key"
)

func DB(config *engine.PluginConfig) (*gorm.DB, error) {
	port := common.GetRecordValueOrDefault(config.Credentials.Advanced, portKey, "3306")
	charset := common.GetRecordValueOrDefault(config.Credentials.Advanced, charsetKey, "utf8mb4")
	parseTime := common.GetRecordValueOrDefault(config.Credentials.Advanced, parseTimeKey, "True")
	loc := common.GetRecordValueOrDefault(config.Credentials.Advanced, locKey, "Local")
	allowClearTextPasswords := common.GetRecordValueOrDefault(config.Credentials.Advanced, allowClearTextPasswordsKey, "0")

	sslMode := common.GetRecordValueOrDefault(config.Credentials.Advanced, sslModeKey, "")
	sslCA := common.GetRecordValueOrDefault(config.Credentials.Advanced, sslCAKey, "")
	sslCert := common.GetRecordValueOrDefault(config.Credentials.Advanced, sslCertKey, "")
	sslKey := common.GetRecordValueOrDefault(config.Credentials.Advanced, sslKeyKey, "")

	tlsConfigName := ""
	if sslMode != "" && sslCA != "" && sslCert != "" && sslKey != "" {
		var err error
		tlsConfigName, err = registerTLSConfig(sslCA, sslCert, sslKey)
		if err != nil {
			return nil, fmt.Errorf("failed to register TLS config: %v", err)
		}
	}

	params := url.Values{}

	for _, record := range config.Credentials.Advanced {
		switch record.Key {
		case portKey, charsetKey, parseTimeKey, locKey, allowClearTextPasswordsKey:
			continue
		default:
			params.Add(record.Key, fmt.Sprintf("%v", record.Value))
		}
	}

	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?charset=%v&parseTime=%v&loc=%v&allowCleartextPasswords=%v&%v", config.Credentials.Username, config.Credentials.Password, config.Credentials.Hostname, port, config.Credentials.Database, charset, parseTime, loc, allowClearTextPasswords, params.Encode())

	if tlsConfigName != "" {
		dsn += fmt.Sprintf("&tls=%v", tlsConfigName)
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func registerTLSConfig(caFile, certFile, keyFile string) (string, error) {
	rootCertPool := x509.NewCertPool()

	caPem, err := os.ReadFile(caFile)
	if err != nil {
		return "", fmt.Errorf("failed to read CA certificate: %v", err)
	}

	if ok := rootCertPool.AppendCertsFromPEM(caPem); !ok {
		return "", fmt.Errorf("failed to append CA certificate")
	}

	clientCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return "", fmt.Errorf("failed to load client certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		RootCAs:      rootCertPool,
		Certificates: []tls.Certificate{clientCert},
	}

	mysql_driver.RegisterTLSConfig("custom", tlsConfig)

	return "custom", nil
}
