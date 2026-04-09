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

package app

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/analytics"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"github.com/clidey/whodb/core/src/router"
	"github.com/clidey/whodb/core/src/settings"
)

const defaultPort = "8080"

// PopulateActiveDatabases sets env.ActiveDatabases from registered plugins.
func PopulateActiveDatabases() {
	for _, p := range engine.RegisteredPlugins() {
		env.ActiveDatabases = append(env.ActiveDatabases, string(p.Type))
	}
}

// AppConfig is the single point of dependency injection for the application.
// The entry point builds this config and passes it to Run().
type AppConfig struct {
	// Schema is the GraphQL executable schema.
	Schema graphql.ExecutableSchema

	// HTTPHandlers maps additional HTTP paths to handlers.
	HTTPHandlers map[string]http.HandlerFunc
}

// Run starts the WhoDB server with the given configuration.
// Called by the entry point.
func Run(config AppConfig, staticFiles embed.FS) {
	showVersion := flag.Bool("version", false, "Print the version and exit")
	flag.Parse()

	if *showVersion {
		version := env.ApplicationVersion
		if version == "" {
			version = "dev"
		}
		fmt.Println(version)
		return
	}

	PopulateActiveDatabases()

	defer log.CloseLogFile()
	appVersion := env.ApplicationVersion
	if appVersion == "" {
		appVersion = "dev"
	}
	log.Alwaysf("Starting WhoDB %s (log level: %s, set WHODB_LOG_LEVEL=warn or WHODB_LOG_LEVEL=error for quieter output)", appVersion, log.GetLevel())

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

	r := router.InitializeRouter(config.Schema, config.HTTPHandlers, staticFiles)

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

	go func() {
		log.Info("Almost there...")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %s\n", err)
		}
	}()

	// Brief pause to let ListenAndServe bind the port before printing the welcome banner
	time.Sleep(2 * time.Second)
	if env.IsEnterpriseEdition {
		log.Always("🎉 Welcome to WhoDB Enterprise! 🎉")
	} else {
		log.Always("🎉 Welcome to WhoDB! 🎉")
	}
	log.Always("Get started by visiting:")
	log.Alwaysf("http://0.0.0.0:%s", port)
	log.Always("Explore and enjoy working with your databases!")

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

	close(quit)

	log.Info("Server exiting")
}
