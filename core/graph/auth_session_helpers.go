package graph

import (
	"fmt"
	"time"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
)

func loginCredentialsToEngineCredentials(credentials model.LoginCredentials) *engine.Credentials {
	advanced := make([]engine.Record, 0, len(credentials.Advanced))
	for _, recordInput := range credentials.Advanced {
		advanced = append(advanced, engine.Record{
			Key:   recordInput.Key,
			Value: recordInput.Value,
		})
	}

	return &engine.Credentials{
		Id:       credentials.ID,
		Type:     credentials.Type,
		Hostname: credentials.Hostname,
		Username: credentials.Username,
		Password: credentials.Password,
		Database: credentials.Database,
		Advanced: advanced,
	}
}

func authSessionPayload(token string, expiresAt time.Time, dbType, host, port, database string) *model.AuthSessionPayload {
	return &model.AuthSessionPayload{
		SessionToken: token,
		ExpiresAt:    expiresAt.Format(time.RFC3339),
		Type:         dbType,
		Hostname:     host,
		Port:         port,
		Database:     database,
		DisplayName:  standaloneDisplayName(dbType, host, database),
	}
}

func standaloneDisplayName(dbType, host, database string) string {
	if database == "" {
		return fmt.Sprintf("%s @ %s", dbType, host)
	}
	return fmt.Sprintf("%s @ %s/%s", dbType, host, database)
}

func advancedValue(records []engine.Record, key string) string {
	for _, record := range records {
		if record.Key == key {
			return record.Value
		}
	}
	return ""
}
