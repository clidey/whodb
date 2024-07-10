package main

import (
	"embed"

	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/router"
)

//go:embed build/*
var staticFiles embed.FS

func main() {
	src.InitializeEngine()
	router.InitializeRouter(staticFiles)
}
