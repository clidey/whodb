/*
 * Copyright 2025 Clidey, Inc.
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
	"context"
	"embed"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/router"
	"github.com/clidey/whodb/core/src/settings"
	"github.com/pkg/errors"
)

//go:embed build/*
var staticFiles embed.FS

const defaultPort = "8080"

func main() {
	log.Logger.Info("Starting WhoDB...")

	settingsCfg := settings.Get()
	if settingsCfg.MetricsEnabled {
	}
	src.InitializeEngine()
	log.Logger.Infof("Auth configured: sources=[Authorization header, Cookie]; keyring service=%s", auth.GetKeyringServiceName())
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
		if !env.IsAPIGatewayEnabled && !common.IsRunningInsideDocker() {
			//common.OpenBrowser(fmt.Sprintf("http://localhost:%v", port))
		}
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Logger.Info("Shutting down server, 30 second timeout started...")

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

	log.Logger.Info("Server exiting")
}
