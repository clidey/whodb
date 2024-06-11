package main

import (
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/router"
)

func main() {
	src.InitializeEngine()
	router.InitializeRouter()
}
