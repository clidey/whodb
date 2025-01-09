package env

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/log"
)

var IsDevelopment = os.Getenv("ENVIRONMENT") == "dev"
var Tokens = common.FilterList(strings.Split(os.Getenv("WHODB_TOKENS"), ","), func(item string) bool {
	return item != ""
})
var IsAPIGatewayEnabled = len(Tokens) > 0
var OllamaHost = os.Getenv("WHODB_OLLAMA_HOST")
var OllamaPort = os.Getenv("WHODB_OLLAMA_PORT")
var AllowedOrigins = common.FilterList(strings.Split(os.Getenv("WHODB_ALLOWED_ORIGINS"), ","), func(item string) bool {
	return item != ""
})

var KeeperToken = os.Getenv("WHODB_KEEPER_TOKEN")

func GetClideyQuickContainerImage() string {
	image := os.Getenv("CLIDEY_QUICK_CONTAINER_IMAGE")
	if len(image) == 0 {
		return ""
	}
	splitImage := strings.Split(image, ":")
	if len(splitImage) != 2 {
		return ""
	}
	return splitImage[1]
}

type DatabaseCredentials struct {
	Alias    string            `json:"alias"`
	Hostname string            `json:"host"`
	Username string            `json:"user"`
	Password string            `json:"password"`
	Database string            `json:"database"`
	Port     string            `json:"port"`
	Config   map[string]string `json:"config"`
	Type     string            `json:"type"`
}

func GetDefaultDatabaseCredentials(databaseType string) []DatabaseCredentials {
	uppercaseDatabaseType := strings.ToUpper(databaseType)
	credEnvVar := fmt.Sprintf("WHODB_%s", uppercaseDatabaseType)
	credEnvValue := os.Getenv(credEnvVar)

	if credEnvValue == "" {
		return findAllDatabaseCredentials(databaseType)
	}

	var creds []DatabaseCredentials
	err := json.Unmarshal([]byte(credEnvValue), &creds)
	if err != nil {
		log.Logger.Warn("Unable to parse database credentials from environment variable: ", err)
		return nil
	}

	return creds
}

func findAllDatabaseCredentials(databaseType string) []DatabaseCredentials {
	uppercaseDatabaseType := strings.ToUpper(databaseType)
	i := 1
	profiles := []DatabaseCredentials{}

	for {
		databaseProfile := os.Getenv(fmt.Sprintf("WHODB_%s_%d", uppercaseDatabaseType, i))
		if databaseProfile == "" {
			break
		}

		var creds DatabaseCredentials
		err := json.Unmarshal([]byte(databaseProfile), &creds)
		if err != nil {
			log.Logger.Warn("Unable to parse database credential: ", err)
			break
		}

		profiles = append(profiles, creds)
		i++
	}

	return profiles
}
