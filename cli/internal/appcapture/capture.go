/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package appcapture

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

//go:embed capture.mjs
var browserScript string

// App is the small part of a hosted app needed to select a capture target.
type App struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// View describes a page that the hosted app runtime can render.
type View struct {
	Page string `json:"page"`
	URL  string `json:"url"`
}

// Action is one browser interaction performed inside the rendered app frame.
type Action struct {
	Kind         string `json:"kind"`
	Selector     string `json:"selector,omitempty"`
	Value        string `json:"value,omitempty"`
	Milliseconds int    `json:"milliseconds,omitempty"`
}

// Diagnostics reports browser errors observed during an app capture.
type Diagnostics struct {
	ConsoleErrors  []string `json:"console_errors,omitempty"`
	PageErrors     []string `json:"page_errors,omitempty"`
	FailedRequests []string `json:"failed_requests,omitempty"`
	HTTPResponses  []string `json:"http_error_responses,omitempty"`
	ActionErrors   []string `json:"action_errors,omitempty"`
	ScriptErrors   []string `json:"script_errors,omitempty"`
}

// Options select one rendered app view for a local browser capture.
type Options struct {
	Host        string
	OrgSlug     string
	ProjectSlug string
	OrgID       string
	ProjectID   string
	AppID       string
	Env         string
	Page        string
	Tab         string
	Actions     []Action
	Script      string
	Token       string
	Width       int
	Height      int
}

// Result contains the screenshot and visible app content observed at capture time.
type Result struct {
	Image            []byte      `json:"-"`
	MIMEType         string      `json:"mime_type"`
	URL              string      `json:"url"`
	Page             string      `json:"page,omitempty"`
	Tab              string      `json:"tab,omitempty"`
	Width            int         `json:"width"`
	Height           int         `json:"height"`
	Text             string      `json:"visible_text,omitempty"`
	Diagnostics      Diagnostics `json:"diagnostics"`
	ScriptResultJSON string      `json:"script_result_json,omitempty"`
}

// URL builds the actual end-user app route without credentials.
func (o Options) URL() (string, error) {
	base, err := url.Parse(o.Host)
	if err != nil || (base.Scheme != "https" && base.Scheme != "http") || base.Host == "" {
		return "", errors.New("valid hosted WhoDB URL is required")
	}
	if o.OrgSlug == "" || o.ProjectSlug == "" || o.AppID == "" {
		return "", errors.New("organization, project, and app are required")
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/" + url.PathEscape(o.OrgSlug) + "/" + url.PathEscape(o.ProjectSlug) + "/apps/" + url.PathEscape(o.AppID)
	query := base.Query()
	if o.Env != "" {
		if o.Env != "dev" && o.Env != "published" {
			return "", errors.New("env must be dev or published")
		}
		if o.Env == "dev" {
			query.Set("env", "dev")
		}
	}
	if o.Page != "" {
		query.Set("page", o.Page)
	}
	base.RawQuery = query.Encode()
	return base.String(), nil
}

type browserInput struct {
	Options           Options        `json:"options"`
	URL               string         `json:"url"`
	Session           map[string]any `json:"session"`
	PlaywrightPackage string         `json:"playwrightPackage"`
}

type browserOutput struct {
	ImageBase64      string      `json:"imageBase64"`
	Text             string      `json:"text"`
	Diagnostics      Diagnostics `json:"diagnostics"`
	ScriptResultJSON string      `json:"scriptResultJSON"`
}

// Capture renders the actual app in an isolated browser and returns WebP bytes.
// The browser helper only permits hosted GraphQL reads and never publishes media.
func Capture(parent context.Context, options Options) (*Result, error) {
	appURL, err := options.URL()
	if err != nil {
		return nil, err
	}
	if options.Token == "" || options.OrgID == "" || options.ProjectID == "" {
		return nil, errors.New("authenticated organization and project are required")
	}
	if options.Width == 0 {
		options.Width = 1440
	}
	if options.Height == 0 {
		options.Height = 900
	}
	if options.Width < 640 || options.Width > 3840 || options.Height < 400 || options.Height > 2160 {
		return nil, errors.New("viewport must be between 640x400 and 3840x2160")
	}
	if len(options.Actions) > 20 || len(options.Script) > 16000 {
		return nil, errors.New("capture allows at most 20 actions and 16000 JavaScript characters")
	}
	for index, action := range options.Actions {
		if err := validateAction(action); err != nil {
			return nil, fmt.Errorf("action %d: %w", index+1, err)
		}
	}
	ctx, cancel := context.WithTimeout(parent, 150*time.Second)
	defer cancel()
	session, err := readSession(ctx, options.Host, options.Token)
	if err != nil {
		return nil, err
	}
	playwrightPackage, err := findPlaywrightPackage()
	if err != nil {
		return nil, err
	}
	input, err := json.Marshal(browserInput{Options: options, URL: appURL, Session: session, PlaywrightPackage: playwrightPackage})
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "node", "--input-type=module", "--eval", browserScript)
	if path := managedBrowserPath(playwrightPackage); path != "" {
		cmd.Env = append(os.Environ(), "PLAYWRIGHT_BROWSERS_PATH="+path)
	}
	cmd.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if strings.Contains(message, "Executable doesn't exist") || strings.Contains(message, "Please run the following command") {
			return nil, errors.New("Chromium is missing; run whodb apps setup-capture, or retry whodb apps screenshot --install")
		}
		if len(message) > 700 {
			message = message[:700]
		}
		return nil, fmt.Errorf("render app: %w: %s", err, message)
	}
	var output browserOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return nil, fmt.Errorf("decode browser capture: %w", err)
	}
	image, err := base64.StdEncoding.DecodeString(output.ImageBase64)
	if err != nil || len(image) < 1000 {
		return nil, errors.New("browser returned no usable screenshot")
	}
	return &Result{Image: image, MIMEType: "image/webp", URL: appURL, Page: options.Page, Tab: options.Tab, Width: options.Width, Height: options.Height, Text: output.Text, Diagnostics: output.Diagnostics, ScriptResultJSON: output.ScriptResultJSON}, nil
}

func validateAction(action Action) error {
	if len(action.Selector) > 500 || len(action.Value) > 2000 {
		return errors.New("selector or value is too long")
	}
	switch action.Kind {
	case "click", "fill", "press", "wait_for":
		if strings.TrimSpace(action.Selector) == "" {
			return errors.New("selector is required")
		}
	case "wait":
		if action.Milliseconds < 0 || action.Milliseconds > 30000 {
			return errors.New("wait must be between 0 and 30000 milliseconds")
		}
	default:
		return errors.New("kind must be click, fill, press, wait_for, or wait")
	}
	return nil
}

func readSession(ctx context.Context, host, token string) (map[string]any, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(host, "/")+"/api/auth/session", nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("read hosted session: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hosted session returned %d", response.StatusCode)
	}
	var session map[string]any
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&session); err != nil {
		return nil, fmt.Errorf("decode hosted session: %w", err)
	}
	if session["authenticated"] != true {
		return nil, errors.New("hosted WhoDB login is not authenticated")
	}
	if session["csrfToken"] == nil || session["csrfToken"] == "" {
		session["csrfToken"] = "capture-readonly"
	}
	return session, nil
}
