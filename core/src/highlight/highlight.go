package highlight

import (
	"github.com/clidey/whodb/core/src/env"
	"github.com/highlight/highlight/sdk/highlight-go"
)

func InitializeHighlight() {
	environment := "production"
	if env.IsDevelopment {
		environment = "development"
	}
	highlight.SetProjectID("")
	highlight.Start(
		highlight.WithServiceName("WhoDB-backend"),
		highlight.WithEnvironment(environment))
}

func StopHighlight() {
	highlight.Stop()
}
