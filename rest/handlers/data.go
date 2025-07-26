package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rediwo/redi-orm/logger"
	"github.com/rediwo/redi-orm/rest/services"
	"github.com/rediwo/redi-orm/rest/types"
	ormTypes "github.com/rediwo/redi-orm/types"
)

// DataHandler handles data operations
type DataHandler struct {
	connHandler  *ConnectionHandler
	queryBuilder *services.QueryBuilder
	logger       logger.Logger
}

// NewDataHandler creates a new data handler
func NewDataHandler(connHandler *ConnectionHandler, l logger.Logger) *DataHandler {
	return &DataHandler{
		connHandler:  connHandler,
		queryBuilder: services.NewQueryBuilder(),
		logger:       l,
	}
}

// Find handles finding multiple records
func (h *DataHandler) Find(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, types.NewErrorResponse("METHOD_NOT_ALLOWED", "Only GET method is allowed"))
		return
	}

	start := time.Now()
	modelName := extractModelName(r.URL.Path)
	connectionName := r.Header.Get("X-Connection-Name")

	db, err := h.connHandler.GetConnection(connectionName)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, types.NewErrorResponse("NO_CONNECTION", "No database connection available"))
		return
	}

	// Parse query parameters
	params, err := types.ParseQueryParams(r.URL.Query())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, types.NewErrorResponse("INVALID_PARAMS", "Invalid query parameters", err.Error()))
		return
	}

	// Build query
	query, err := h.queryBuilder.BuildFindQuery(db, modelName, params)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, types.NewErrorResponse("QUERY_BUILD_ERROR", "Failed to build query", err.Error()))
		return
	}

	// Execute query
	var results []map[string]any
	if err := query.FindMany(r.Context(), &results); err != nil {
		writeJSON(w, http.StatusInternalServerError, types.NewErrorResponse("QUERY_ERROR", "Failed to execute query", err.Error()))
		return
	}

	// Get total count for pagination
	var total int
	if params.Page > 0 {
		countQuery := h.queryBuilder.BuildCountQuery(db, modelName, params)
		count, err := countQuery.Count(r.Context())
		if err != nil {
			h.logger.Error("Failed to get count: %v", err)
			total = len(results)
		} else {
			total = int(count)
		}
	}

	// Build response
	var response *types.Response
	if params.Page > 0 {
		response = types.NewPaginatedResponse(results, params.Page, params.Limit, total)
	} else {
		response = types.NewSuccessResponse(results)
	}

	response.WithExecutionTime(time.Since(start))
	writeJSON(w, http.StatusOK, response)
}

// FindOne handles finding a single record
func (h *DataHandler) FindOne(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, types.NewErrorResponse("METHOD_NOT_ALLOWED", "Only GET method is allowed"))
		return
	}

	start := time.Now()
	modelName, id := extractModelAndID(r.URL.Path)
	connectionName := r.Header.Get("X-Connection-Name")

	db, err := h.connHandler.GetConnection(connectionName)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, types.NewErrorResponse("NO_CONNECTION", "No database connection available"))
		return
	}

	// Parse query parameters for includes
	params, _ := types.ParseQueryParams(r.URL.Query())

	// Build query
	query := db.Model(modelName).Select()

	// Add where condition for ID
	idValue := parseID(id)
	query = query.WhereCondition(query.Where("id").Equals(idValue))

	// Add includes if specified
	if params.Include != nil {
		query = h.queryBuilder.ApplyIncludes(query, params.Include)
	}

	// Execute query
	var result map[string]any
	if err := query.FindFirst(r.Context(), &result); err != nil {
		if err == sql.ErrNoRows || strings.Contains(err.Error(), "no rows") {
			writeJSON(w, http.StatusNotFound, types.NewErrorResponse("NOT_FOUND", "Record not found"))
		} else {
			writeJSON(w, http.StatusInternalServerError, types.NewErrorResponse("QUERY_ERROR", "Failed to execute query", err.Error()))
		}
		return
	}

	response := types.NewSuccessResponse(result).WithExecutionTime(time.Since(start))
	writeJSON(w, http.StatusOK, response)
}

// Create handles creating a new record
func (h *DataHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, types.NewErrorResponse("METHOD_NOT_ALLOWED", "Only POST method is allowed"))
		return
	}

	start := time.Now()
	modelName := extractModelName(r.URL.Path)
	connectionName := r.Header.Get("X-Connection-Name")

	db, err := h.connHandler.GetConnection(connectionName)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, types.NewErrorResponse("NO_CONNECTION", "No database connection available"))
		return
	}

	// Parse request body
	var req types.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, types.NewErrorResponse("INVALID_REQUEST", "Invalid request body", err.Error()))
		return
	}

	// Execute insert
	result, err := db.Model(modelName).Insert(req.Data).Exec(r.Context())
	if err != nil {
		h.logger.Error("Failed to create record: %v", err)
		writeJSON(w, http.StatusInternalServerError, types.NewErrorResponse("CREATE_ERROR", "Failed to create record", err.Error()))
		return
	}

	// Fetch the created record
	if result.LastInsertID > 0 {
		query := db.Model(modelName).Select()
		query = query.WhereCondition(query.Where("id").Equals(result.LastInsertID))

		var created map[string]any
		if err := query.FindFirst(r.Context(), &created); err == nil {
			response := types.NewSuccessResponse(created).WithExecutionTime(time.Since(start))
			writeJSON(w, http.StatusCreated, response)
			return
		}
	}

	// Fallback response
	response := types.NewSuccessResponse(map[string]any{
		"id":      result.LastInsertID,
		"created": true,
	}).WithExecutionTime(time.Since(start))
	writeJSON(w, http.StatusCreated, response)
}

// Update handles updating a record
func (h *DataHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, types.NewErrorResponse("METHOD_NOT_ALLOWED", "Only PUT or PATCH methods are allowed"))
		return
	}

	start := time.Now()
	modelName, id := extractModelAndID(r.URL.Path)
	connectionName := r.Header.Get("X-Connection-Name")

	db, err := h.connHandler.GetConnection(connectionName)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, types.NewErrorResponse("NO_CONNECTION", "No database connection available"))
		return
	}

	// Parse request body
	var req types.UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, types.NewErrorResponse("INVALID_REQUEST", "Invalid request body", err.Error()))
		return
	}

	// Build update query
	idValue := parseID(id)
	query := db.Model(modelName).Update(req.Data)
	query = query.WhereCondition(query.Where("id").Equals(idValue))

	// Execute update
	result, err := query.Exec(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, types.NewErrorResponse("UPDATE_ERROR", "Failed to update record", err.Error()))
		return
	}

	if result.RowsAffected == 0 {
		writeJSON(w, http.StatusNotFound, types.NewErrorResponse("NOT_FOUND", "Record not found"))
		return
	}

	// Fetch the updated record
	selectQuery := db.Model(modelName).Select()
	selectQuery = selectQuery.WhereCondition(selectQuery.Where("id").Equals(idValue))

	var updated map[string]any
	if err := selectQuery.FindFirst(r.Context(), &updated); err == nil {
		response := types.NewSuccessResponse(updated).WithExecutionTime(time.Since(start))
		writeJSON(w, http.StatusOK, response)
		return
	}

	// Fallback response
	response := types.NewSuccessResponse(map[string]any{
		"id":      idValue,
		"updated": true,
	}).WithExecutionTime(time.Since(start))
	writeJSON(w, http.StatusOK, response)
}

// Delete handles deleting a record
func (h *DataHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, types.NewErrorResponse("METHOD_NOT_ALLOWED", "Only DELETE method is allowed"))
		return
	}

	start := time.Now()
	modelName, id := extractModelAndID(r.URL.Path)
	connectionName := r.Header.Get("X-Connection-Name")

	db, err := h.connHandler.GetConnection(connectionName)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, types.NewErrorResponse("NO_CONNECTION", "No database connection available"))
		return
	}

	// Build delete query
	idValue := parseID(id)
	query := db.Model(modelName).Delete()
	query = query.WhereCondition(query.Where("id").Equals(idValue))

	// Execute delete
	result, err := query.Exec(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, types.NewErrorResponse("DELETE_ERROR", "Failed to delete record", err.Error()))
		return
	}

	if result.RowsAffected == 0 {
		writeJSON(w, http.StatusNotFound, types.NewErrorResponse("NOT_FOUND", "Record not found"))
		return
	}

	response := types.NewSuccessResponse(map[string]any{
		"id":      idValue,
		"deleted": true,
	}).WithExecutionTime(time.Since(start))
	writeJSON(w, http.StatusOK, response)
}

// BatchCreate handles creating multiple records
func (h *DataHandler) BatchCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, types.NewErrorResponse("METHOD_NOT_ALLOWED", "Only POST method is allowed"))
		return
	}

	start := time.Now()
	modelName := extractModelName(r.URL.Path)
	connectionName := r.Header.Get("X-Connection-Name")

	db, err := h.connHandler.GetConnection(connectionName)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, types.NewErrorResponse("NO_CONNECTION", "No database connection available"))
		return
	}

	// Parse request body
	var req types.BatchCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, types.NewErrorResponse("INVALID_REQUEST", "Invalid request body", err.Error()))
		return
	}

	// Execute batch insert in transaction
	var createdCount int
	err = db.Transaction(r.Context(), func(tx ormTypes.Transaction) error {
		result, err := tx.CreateMany(r.Context(), modelName, req.Data)
		if err != nil {
			return err
		}
		createdCount = int(result.RowsAffected)
		return nil
	})

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, types.NewErrorResponse("BATCH_CREATE_ERROR", "Failed to create records", err.Error()))
		return
	}

	response := types.NewSuccessResponse(map[string]any{
		"created": createdCount,
	}).WithExecutionTime(time.Since(start))
	writeJSON(w, http.StatusCreated, response)
}

// extractModelName extracts model name from URL path
func extractModelName(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "api" {
		return parts[1]
	}
	return ""
}

// extractModelAndID extracts model name and ID from URL path
func extractModelAndID(path string) (string, string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 && parts[0] == "api" {
		return parts[1], parts[2]
	}
	return "", ""
}

// parseID attempts to parse ID as integer or returns as string
func parseID(id string) any {
	if intID, err := strconv.Atoi(id); err == nil {
		return intID
	}
	return id
}
