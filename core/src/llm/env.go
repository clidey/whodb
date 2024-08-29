package llm

import (
	"fmt"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/env"
)

func getOllamaEndpoint() string {
	host := "localhost"
	port := "11434"

	if common.IsRunningInsideDocker() {
		host = "host.docker.internal"
	}

	if env.OllamaHost != "" {
		host = env.OllamaHost
	}
	if env.OllamaPort != "" {
		port = env.OllamaPort
	}

	return fmt.Sprintf("http://%v:%v/api", host, port)
}
