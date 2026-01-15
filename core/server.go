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
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"github.com/clidey/whodb/core/src/router"
	"github.com/clidey/whodb/core/src/settings"
	"github.com/pkg/errors"
)

const defaultPort = "8080"

func main() {
	log.Logger.Info("Starting WhoDB...")

	settingsCfg := settings.Get()

	if err := analytics.Initialize(analytics.Config{
		APIKey:      env.PosthogAPIKey,
		Host:        env.PosthogHost,
		Environment: env.ApplicationEnvironment,
		AppVersion:  env.ApplicationVersion,
	}); err != nil {
		log.Logger.WithError(err).Warn("Analytics: PostHog initialization failed, metrics disabled")
	} else {
		defer analytics.Shutdown()
	}
	analytics.SetEnabled(settingsCfg.MetricsEnabled)

	src.InitializeEngine()
	log.Logger.Infof("Auth configured: sources=[Authorization header, Cookie]; keyring service=%s", auth.GetKeyringServiceName())

	// Load persisted AWS providers from disk (if any)
	if err := settings.LoadProvidersFromFile(); err != nil {
		log.Logger.Warnf("Failed to load persisted AWS providers: %v", err)
	}

	// Initialize AWS providers from environment variables (may add or override persisted)
	if err := settings.InitAWSProvidersFromEnv(); err != nil {
		log.Logger.Warnf("Failed to initialize AWS providers from environment: %v", err)
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
		log.Logger.Info("Almost there...")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Logger.Fatalf("listen: %s\n", err)
			serverStarted <- false
		}
	}()

	select {
	case success := <-serverStarted:
		if !success {
			log.Logger.Error("Server failed to start. Exiting...")
			os.Exit(1)
		}
	case <-time.After(2 * time.Second):
		if env.IsEnterpriseEdition {
			log.Logger.Info("🎉 Welcome to WhoDB Enterprise! 🎉")
		} else {
			log.Logger.Info("🎉 Welcome to WhoDB! 🎉")
		}
		log.Logger.Info("Get started by visiting:")
		log.Logger.Infof("http://0.0.0.0:%s", port)
		log.Logger.Info("Explore and enjoy working with your databases!")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Logger.Info("Shutting down server...")

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server and close DB connections in parallel
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := srv.Shutdown(ctx); err != nil {
			log.Logger.Errorf("HTTP server shutdown error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		plugins.CloseAllConnections(ctx)
	}()

	wg.Wait()

	close(serverStarted)
	close(quit)

	log.Logger.Info("Server exiting")
}
