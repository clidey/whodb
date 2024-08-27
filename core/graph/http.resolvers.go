package graph

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/go-chi/chi/v5"
)

func SetupHTTPServer(router chi.Router) {
	router.Get("/profiles", getProfilesHandler)
	router.Get("/databases/{type}", getDatabasesHandler)
	router.Get("/schema/{type}", getSchemaHandler)
	router.Get("/storage-units/{type}/{schema}", getStorageUnitsHandler)
	router.Get("/rows", getRowsHandler)
	router.Post("/raw-execute", rawExecuteHandler)
	router.Get("/graph/{type}/{schema}", getGraphHandler)
	router.Get("/ai-models", getAIModelsHandler)
	router.Post("/ai-chat", aiChatHandler)

	router.Post("/auth/login", loginHandler)
	router.Post("/auth/login-with-profile", loginWithProfileHandler)
	router.Post("/auth/logout", logoutHandler)

	router.Post("/storage-units", addStorageUnitHandler)
	router.Post("/rows", addRowHandler)
	router.Delete("/rows", deleteRowHandler)
}

var resolver = mutationResolver{}

func getProfilesHandler(w http.ResponseWriter, r *http.Request) {
	profiles, err := resolver.Query().Profiles(context.Background())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(profiles)
}

func getDatabasesHandler(w http.ResponseWriter, r *http.Request) {
	dbType := model.DatabaseType(chi.URLParam(r, "type"))
	databases, err := resolver.Query().Database(context.Background(), dbType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(databases)
}

func getSchemaHandler(w http.ResponseWriter, r *http.Request) {
	dbType := model.DatabaseType(chi.URLParam(r, "type"))
	schemas, err := resolver.Query().Schema(context.Background(), dbType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(schemas)
}

func getStorageUnitsHandler(w http.ResponseWriter, r *http.Request) {
	dbType := model.DatabaseType(chi.URLParam(r, "type"))
	schema := chi.URLParam(r, "schema")
	storageUnits, err := resolver.Query().StorageUnit(context.Background(), dbType, schema)
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

	rowsResult, err := resolver.Query().Row(context.Background(), dbType, schema, storageUnit, where, pageSize, pageOffset)
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

	rowsResult, err := resolver.Query().RawExecute(context.Background(), req.Type, req.Query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(rowsResult)
}

func getGraphHandler(w http.ResponseWriter, r *http.Request) {
	dbType := model.DatabaseType(chi.URLParam(r, "type"))
	schema := chi.URLParam(r, "schema")
	graphUnits, err := resolver.Query().Graph(context.Background(), dbType, schema)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(graphUnits)
}

func getAIModelsHandler(w http.ResponseWriter, r *http.Request) {
	modelType := r.URL.Query().Get("modelType")
	token := r.URL.Query().Get("token")
	models, err := resolver.Query().AIModel(context.Background(), modelType, &token)
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

	messages, err := resolver.Query().AIChat(context.Background(), req.ModelType, &req.Token, req.Type, req.Schema, req.Input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(messages)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var credentials model.LoginCredentials
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status, err := resolver.Mutation().Login(context.Background(), credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	json.NewEncoder(w).Encode(status)
}

func loginWithProfileHandler(w http.ResponseWriter, r *http.Request) {
	var profile model.LoginProfileInput
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status, err := resolver.Mutation().LoginWithProfile(context.Background(), profile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	json.NewEncoder(w).Encode(status)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	status, err := resolver.Mutation().Logout(context.Background())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(status)
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

	status, err := resolver.Mutation().AddStorageUnit(context.Background(), req.Type, req.Schema, req.StorageUnit, req.Fields)
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

	status, err := resolver.Mutation().AddRow(context.Background(), req.Type, req.Schema, req.StorageUnit, req.Values)
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

	status, err := resolver.Mutation().DeleteRow(context.Background(), req.Type, req.Schema, req.StorageUnit, req.Values)
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
