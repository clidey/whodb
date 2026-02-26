/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	_ "github.com/clidey/whodb/core/src/bamlconfig" // Must be first - sets BAML_LOG before native library loads

	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/analytics"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"github.com/clidey/whodb/core/src/router"
	"github.com/clidey/whodb/core/src/settings"
	"errors"
)

const defaultPort = "8080"

func main() {
	defer log.CloseLogFile()
	log.Alwaysf("Starting WhoDB... (log level: %s, set WHODB_LOG_LEVEL=warn or WHODB_LOG_LEVEL=error for quieter output)", log.GetLevel())

	settingsCfg := settings.Get()

	if err := analytics.Initialize(analytics.Config{
		APIKey:      env.PosthogAPIKey,
		Host:        env.PosthogHost,
		Environment: env.ApplicationEnvironment,
		AppVersion:  env.ApplicationVersion,
	}); err != nil {
		// analytics init failure is non-fatal
	} else {
		defer analytics.Shutdown()
	}
	analytics.SetEnabled(settingsCfg.MetricsEnabled)

	src.InitializeEngine()

	// Load persisted AWS providers from disk (if any)
	if err := settings.LoadProvidersFromFile(); err != nil {
		log.Warnf("Failed to load persisted AWS providers: %v", err)
	}

	// Initialize AWS providers from environment variables (may add or override persisted)
	if err := settings.InitAWSProvidersFromEnv(); err != nil {
		log.Warnf("Failed to initialize AWS providers from environment: %v", err)
	}

	r := router.InitializeRouter(staticFiles)

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      1 * time.Minute,
		IdleTimeout:       30 * time.Second,
	}

	serverStarted := make(chan bool, 1)

	go func() {
		log.Info("Almost there...")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %s\n", err)
		}
	}()

	select {
	case success := <-serverStarted:
		if !success {
			log.Error("Server failed to start. Exiting...")
			os.Exit(1)
		}
	case <-time.After(2 * time.Second):
		if env.IsEnterpriseEdition {
			log.Always("🎉 Welcome to WhoDB Enterprise! 🎉")
		} else {
			log.Always("🎉 Welcome to WhoDB! 🎉")
		}
		log.Always("Get started by visiting:")
		log.Alwaysf("http://0.0.0.0:%s", port)
		log.Always("Explore and enjoy working with your databases!")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server and close DB connections in parallel
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := srv.Shutdown(ctx); err != nil {
			log.Errorf("HTTP server shutdown error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		plugins.CloseAllConnections(ctx)
	}()

	wg.Wait()

	close(serverStarted)
	close(quit)

	log.Info("Server exiting")
}
