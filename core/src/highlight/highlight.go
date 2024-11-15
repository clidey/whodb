package highlight

import (
	"github.com/clidey/whodb/core/src/env"
	"github.com/highlight/highlight/sdk/highlight-go"
)

const highlightProjectId = "4d7z8oqe"

func InitializeHighlight() {
	environment := "production"
	if env.IsDevelopment {
		environment = "development"
	}
	highlight.SetProjectID(highlightProjectId)
	highlight.Start(
		highlight.WithServiceName("WhoDB-backend"),
		highlight.WithEnvironment(environment))
}

func StopHighlight() {
	highlight.Stop()
}
