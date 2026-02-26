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
	"testing"
	"time"

	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/router"
	"github.com/pkg/errors"
)

//go:embed build/*
var staticFilesTest embed.FS

const defaultPortTest = "8080"

var srv *http.Server

func TestMain(m *testing.M) {
	log.Info("Starting WhoDB in test mode (Ctrl+C to exit)...")

	src.InitializeEngine()
	r := router.InitializeRouter(staticFilesTest)

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPortTest
	}

	srv = &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      1 * time.Minute,
		IdleTimeout:       30 * time.Second,
	}

	go func() {
		log.Info("Server starting...")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s", err)
			os.Exit(1)
		}
	}()

	log.Infof("🎉 WhoDB test server running at http://localhost:%s 🎉", port)

	// Wait for SIGINT (Ctrl+C) or SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Received shutdown signal (Ctrl+C). Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Errorf("Graceful shutdown failed: %v", err)
	}

	log.Info("Test server shut down. Exiting and writing coverage.")
	os.Exit(m.Run())
}
