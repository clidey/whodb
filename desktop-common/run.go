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

package common

import (
	"embed"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/router"
)

// RunApp starts the Wails application with the given configuration
func RunApp(edition string, title string, assets embed.FS) error {
	// Set desktop mode for backend (SQLite path handling, auth, etc.)
	os.Setenv("WHODB_DESKTOP", "true")

	// Initialize WhoDB engine (same as server.go)
	src.InitializeEngine()
	log.Logger.Infof("Running WhoDB Desktop %s Edition", strings.ToUpper(edition))

	// Get the Chi router with embedded assets
	r := router.InitializeRouter(assets)

	// Create an instance of the app structure using common package
	app := NewApp(edition)

	// Create application with options
	err := wails.Run(&options.App{
		Title:     title,
		Width:     1400,
		Height:    900,
		MinWidth:  1024,
		MinHeight: 768,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: r, // Pass entire Chi router - handles GraphQL and all routes
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.Startup,
		OnShutdown:       app.Shutdown,
		OnDomReady:       app.DomReady,
		Bind: []any{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarDefault(),
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}

	return err
}