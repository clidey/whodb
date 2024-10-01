package main

import (
	"context"
	"embed"
	"fmt"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/router"
	"github.com/pkg/errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//go:embed build/*
var staticFiles embed.FS

const defaultPort = "8080"

func main() {
	log.Logger.Info("Starting WhoDB...")
	src.InitializeEngine()
	r := router.InitializeRouter(staticFiles)

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	serverStarted := make(chan bool, 1)

	go func() {
		log.Logger.Info("Almost there...")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Logger.Fatalf("listen: %s\n", err)
			serverStarted <- false
		}
	}()

	select {
	case success := <-serverStarted:
		if !success {
			log.Logger.Println("Server failed to start. Exiting...")
			os.Exit(1)
		}
	case <-time.After(2 * time.Second):
		log.Logger.Infof("🎉 Welcome to WhoDB! 🎉")
		log.Logger.Infof("Get started by visiting:")
		log.Logger.Infof("http://0.0.0.0:%s", port)
		log.Logger.Info("Explore and enjoy working with your databases!")
		if !env.IsAPIGatewayEnabled && !common.IsRunningInsideDocker() {
			common.OpenBrowser(fmt.Sprintf("http://localhost:%v", port))
		}
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Logger.Println("Shutting down server, 30 second timeout started...")

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	if err := srv.Shutdown(ctx); err != nil {
		log.Logger.Fatalf("Server forced to shutdown: %v, resources might be left hanging", err)
	}

	close(serverStarted)
	close(quit)

	log.Logger.Println("Server exiting")
}
