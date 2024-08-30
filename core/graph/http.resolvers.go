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
	json.NewEncoder(w).Encode(profiles)
}

func getDatabasesHandler(w http.ResponseWriter, r *http.Request) {
	dbType := model.DatabaseType(r.URL.Query().Get("type"))
	databases, err := resolver.Query().Database(r.Context(), dbType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(databases)
}

func getSchemaHandler(w http.ResponseWriter, r *http.Request) {
	dbType := model.DatabaseType(r.URL.Query().Get("type"))
	schemas, err := resolver.Query().Schema(r.Context(), dbType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(schemas)
}

func getStorageUnitsHandler(w http.ResponseWriter, r *http.Request) {
	dbType := model.DatabaseType(r.URL.Query().Get("type"))
	schema := r.URL.Query().Get("schema")
	storageUnits, err := resolver.Query().StorageUnit(r.Context(), dbType, schema)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(storageUnits)
}

func getRowsHandler(w http.ResponseWriter, r *http.Request) {
	dbType := model.DatabaseType(r.URL.Query().Get("type"))
	schema := r.URL.Query().Get("schema")
	storageUnit := r.URL.Query().Get("storageUnit")
	where := r.URL.Query().Get("where")
	pageSize := parseQueryParamToInt(r.URL.Query().Get("pageSize"))
	pageOffset := parseQueryParamToInt(r.URL.Query().Get("pageOffset"))

	rowsResult, err := resolver.Query().Row(r.Context(), dbType, schema, storageUnit, where, pageSize, pageOffset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(rowsResult)
}

func rawExecuteHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type  model.DatabaseType `json:"type"`
		Query string             `json:"query"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rowsResult, err := resolver.Query().RawExecute(r.Context(), req.Type, req.Query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(rowsResult)
}

func getGraphHandler(w http.ResponseWriter, r *http.Request) {
	dbType := model.DatabaseType(r.URL.Query().Get("type"))
	schema := r.URL.Query().Get("schema")
	graphUnits, err := resolver.Query().Graph(r.Context(), dbType, schema)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(graphUnits)
}

func getAIModelsHandler(w http.ResponseWriter, r *http.Request) {
	modelType := r.URL.Query().Get("modelType")
	token := r.URL.Query().Get("token")
	models, err := resolver.Query().AIModel(r.Context(), modelType, &token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(models)
}

func aiChatHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ModelType string             `json:"modelType"`
		Token     string             `json:"token"`
		Type      model.DatabaseType `json:"type"`
		Schema    string             `json:"schema"`
		Input     model.ChatInput    `json:"input"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	messages, err := resolver.Query().AIChat(r.Context(), req.ModelType, &req.Token, req.Type, req.Schema, req.Input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(messages)
}

func addStorageUnitHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type        model.DatabaseType   `json:"type"`
		Schema      string               `json:"schema"`
		StorageUnit string               `json:"storageUnit"`
		Fields      []*model.RecordInput `json:"fields"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status, err := resolver.Mutation().AddStorageUnit(r.Context(), req.Type, req.Schema, req.StorageUnit, req.Fields)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(status)
}

func addRowHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type        model.DatabaseType   `json:"type"`
		Schema      string               `json:"schema"`
		StorageUnit string               `json:"storageUnit"`
		Values      []*model.RecordInput `json:"values"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status, err := resolver.Mutation().AddRow(r.Context(), req.Type, req.Schema, req.StorageUnit, req.Values)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(status)
}

func deleteRowHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type        model.DatabaseType   `json:"type"`
		Schema      string               `json:"schema"`
		StorageUnit string               `json:"storageUnit"`
		Values      []*model.RecordInput `json:"values"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status, err := resolver.Mutation().DeleteRow(r.Context(), req.Type, req.Schema, req.StorageUnit, req.Values)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(status)
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
