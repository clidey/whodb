// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package graph

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/go-chi/chi/v5"
)

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

	router.Post("/api/storage-units", addStorageUnitHandler)
	router.Post("/api/rows", addRowHandler)
	router.Delete("/api/rows", deleteRowHandler)
}

var resolver = mutationResolver{}

func getProfilesHandler(w http.ResponseWriter, r *http.Request) {
	profiles, err := resolver.Query().Profiles(r.Context())
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
	typeArg := r.URL.Query().Get("type")
	databases, err := resolver.Query().Database(r.Context(), typeArg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(databases)
	if err != nil {
		return
	}
}

func getSchemaHandler(w http.ResponseWriter, r *http.Request) {
	schemas, err := resolver.Query().Schema(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(schemas)
	if err != nil {
		return
	}
}

func getStorageUnitsHandler(w http.ResponseWriter, r *http.Request) {
	schema := r.URL.Query().Get("schema")
	storageUnits, err := resolver.Query().StorageUnit(r.Context(), schema)
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
	schema := r.URL.Query().Get("schema")
	storageUnit := r.URL.Query().Get("storageUnit")
	// where := r.URL.Query().Get("where")
	pageSize := parseQueryParamToInt(r.URL.Query().Get("pageSize"))
	pageOffset := parseQueryParamToInt(r.URL.Query().Get("pageOffset"))

	// todo: add where condition if necessary
	rowsResult, err := resolver.Query().Row(r.Context(), schema, storageUnit, &model.WhereCondition{}, pageSize, pageOffset)
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

	rowsResult, err := resolver.Query().RawExecute(r.Context(), req.Query)
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
	schema := r.URL.Query().Get("schema")
	graphUnits, err := resolver.Query().Graph(r.Context(), schema)
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
	models, err := resolver.Query().AIModel(r.Context(), modelType, &token)
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
		ModelType string          `json:"modelType"`
		Token     string          `json:"token"`
		Schema    string          `json:"schema"`
		Input     model.ChatInput `json:"input"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	messages, err := resolver.Query().AIChat(r.Context(), req.ModelType, &req.Token, req.Schema, req.Input)
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
		Schema      string               `json:"schema"`
		StorageUnit string               `json:"storageUnit"`
		Fields      []*model.RecordInput `json:"fields"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status, err := resolver.Mutation().AddStorageUnit(r.Context(), req.Schema, req.StorageUnit, req.Fields)
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
		Schema      string               `json:"schema"`
		StorageUnit string               `json:"storageUnit"`
		Values      []*model.RecordInput `json:"values"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status, err := resolver.Mutation().AddRow(r.Context(), req.Schema, req.StorageUnit, req.Values)
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
		Schema      string               `json:"schema"`
		StorageUnit string               `json:"storageUnit"`
		Values      []*model.RecordInput `json:"values"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status, err := resolver.Mutation().DeleteRow(r.Context(), req.Schema, req.StorageUnit, req.Values)
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
