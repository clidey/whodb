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

package graph

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/go-chi/chi/v5"
)

// SetupHTTPServer registers REST API endpoints that wrap GraphQL resolvers for clients
// that cannot use GraphQL directly (e.g., simple HTTP clients, legacy integrations).
func SetupHTTPServer(router chi.Router) {
	router.Get("/api/profiles", getProfilesHandler)
	router.Get("/api/databases", getDatabasesHandler)
	router.Get("/api/schema", getSchemaHandler)
	router.Get("/api/storage-units", getStorageUnitsHandler)
	router.Get("/api/rows", getRowsHandler)
	router.Post("/api/raw-execute", rawExecuteHandler)
	router.Get("/api/graph", getGraphHandler)
	router.Get("/api/ai-models", getAIModelsHandler)
	router.Post("/api/ai-chat", aiChatHandler)
	router.Post("/api/ai-chat/stream", aiChatStreamHandler)
	router.Post("/api/agent/stream", agentStreamHandler)
	router.Post("/api/agent/permit", agentPermitHandler)
	router.Post("/api/app/generate", appGenerateHandler)
	router.Post("/api/function/stream", functionStreamHandler)

	router.Post("/api/storage-units", addStorageUnitHandler)
	router.Post("/api/rows", addRowHandler)
	router.Delete("/api/rows", deleteRowHandler)

	router.Post("/api/export", HandleExport)

	// AI chat streaming endpoint is registered via build tags in http_ai_stream.go (!arm) / http_ai_stream_arm.go (arm)
}

var resolver = Resolver{}

func getProfilesHandler(w http.ResponseWriter, r *http.Request) {
	profiles, err := resolver.Query().SourceProfiles(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(profiles)
	if err != nil {
		return
	}
}

func getDatabasesHandler(w http.ResponseWriter, r *http.Request) {
	sourceType := r.URL.Query().Get("sourceType")
	if sourceType == "" {
		http.Error(w, "missing required query parameter: sourceType", http.StatusBadRequest)
		return
	}

	fieldKey := r.URL.Query().Get("fieldKey")
	if fieldKey == "" {
		fieldKey = "Database"
	}

	options, err := resolver.Query().SourceFieldOptions(r.Context(), sourceType, fieldKey, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(options)
	if err != nil {
		return
	}
}

func getSchemaHandler(w http.ResponseWriter, r *http.Request) {
	parent, err := querySourceObjectRef(r.URL.Query(), "parentKind", "parentPath")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	objects, err := resolver.Query().SourceObjects(r.Context(), parent, []model.SourceObjectKind{model.SourceObjectKindSchema})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	schemas := make([]string, 0, len(objects))
	for _, object := range objects {
		schemas = append(schemas, object.Name)
	}
	err = json.NewEncoder(w).Encode(schemas)
	if err != nil {
		return
	}
}

func getStorageUnitsHandler(w http.ResponseWriter, r *http.Request) {
	parent, err := querySourceObjectRef(r.URL.Query(), "parentKind", "parentPath")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	storageUnits, err := resolver.Query().SourceObjects(r.Context(), parent, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(storageUnits)
	if err != nil {
		return
	}
}

func getRowsHandler(w http.ResponseWriter, r *http.Request) {
	pageSize := parseQueryParamToInt(r.URL.Query().Get("pageSize"))
	pageOffset := parseQueryParamToInt(r.URL.Query().Get("pageOffset"))
	ref, err := querySourceObjectRef(r.URL.Query(), "kind", "path")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if ref == nil {
		http.Error(w, "missing required query parameter: kind", http.StatusBadRequest)
		return
	}

	// TODO: Add where condition parsing from query params if needed.
	rowsResult, err := resolver.Query().SourceRows(r.Context(), *ref, nil, []*model.SortCondition{}, pageSize, pageOffset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(rowsResult)
	if err != nil {
		return
	}
}

func rawExecuteHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rowsResult, err := resolver.Query().RunSourceQuery(r.Context(), req.Query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(rowsResult)
	if err != nil {
		return
	}
}

func getGraphHandler(w http.ResponseWriter, r *http.Request) {
	ref, err := querySourceObjectRef(r.URL.Query(), "kind", "path")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	graphUnits, err := resolver.Query().SourceGraph(r.Context(), ref)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(graphUnits)
	if err != nil {
		return
	}
}

func getAIModelsHandler(w http.ResponseWriter, r *http.Request) {
	modelType := r.URL.Query().Get("modelType")
	token := r.URL.Query().Get("token")
	models, err := resolver.Query().AIModel(r.Context(), nil, modelType, &token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(models)
	if err != nil {
		return
	}
}

func aiChatHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ModelType string                      `json:"modelType"`
		Token     string                      `json:"token"`
		Model     string                      `json:"model"`
		Endpoint  string                      `json:"endpoint"`
		Ref       *model.SourceObjectRefInput `json:"ref"`
		Input     model.ChatInput             `json:"input"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Copy top-level model to input.Model for GraphQL resolver compatibility.
	// Frontend sends model at top level (matching streaming endpoint format).
	if req.Model != "" && req.Input.Model == "" {
		req.Input.Model = req.Model
	}

	messages, err := resolver.Query().AIChat(r.Context(), nil, req.ModelType, &req.Token, req.Ref, req.Input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(messages)
	if err != nil {
		return
	}
}

func addStorageUnitHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Parent *model.SourceObjectRefInput `json:"parent"`
		Name   string                      `json:"name"`
		Fields []*model.RecordInput        `json:"fields"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status, err := resolver.Mutation().CreateSourceObject(r.Context(), req.Parent, req.Name, req.Fields)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(status)
	if err != nil {
		return
	}
}

func addRowHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Ref    *model.SourceObjectRefInput `json:"ref"`
		Values []*model.RecordInput        `json:"values"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Ref == nil {
		http.Error(w, "missing required request field: ref", http.StatusBadRequest)
		return
	}

	status, err := resolver.Mutation().AddSourceRow(r.Context(), *req.Ref, req.Values)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(status)
	if err != nil {
		return
	}
}

func deleteRowHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Ref    *model.SourceObjectRefInput `json:"ref"`
		Values []*model.RecordInput        `json:"values"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Ref == nil {
		http.Error(w, "missing required request field: ref", http.StatusBadRequest)
		return
	}

	status, err := resolver.Mutation().DeleteSourceRow(r.Context(), *req.Ref, req.Values)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(status)
	if err != nil {
		return
	}
}

func parseQueryParamToInt(queryParam string) int {
	if queryParam == "" {
		return 0
	}
	value, err := strconv.Atoi(queryParam)
	if err != nil {
		return 0
	}
	return value
}

func querySourceObjectRef(values url.Values, kindKey string, pathKey string) (*model.SourceObjectRefInput, error) {
	kind := values.Get(kindKey)
	path := values[pathKey]
	if kind == "" {
		if len(path) == 0 {
			return nil, nil
		}
		return nil, errors.New("missing required query parameter: " + kindKey)
	}

	return &model.SourceObjectRefInput{
		Kind: model.SourceObjectKind(kind),
		Path: path,
	}, nil
}
