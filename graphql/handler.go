package graphql

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
)

// LogLevel represents logging verbosity
type LogLevel int

const (
	LogLevelNone LogLevel = iota
	LogLevelError
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

// Handler provides a generic HTTP handler for GraphQL requests
type Handler struct {
	schema            *graphql.Schema
	pretty            bool
	graphiQLEnabled   bool
	playgroundEnabled bool
	logLevel          LogLevel
}

// NewHandler creates a new GraphQL HTTP handler
func NewHandler(schema *graphql.Schema) *Handler {
	return &Handler{
		schema:            schema,
		pretty:            true,
		graphiQLEnabled:   false,
		playgroundEnabled: true,
		logLevel:          LogLevelInfo,
	}
}

// SetPretty enables or disables pretty printing of JSON responses
func (h *Handler) SetPretty(pretty bool) *Handler {
	h.pretty = pretty
	return h
}

// EnableGraphiQL enables the GraphiQL interface
func (h *Handler) EnableGraphiQL() *Handler {
	h.graphiQLEnabled = true
	h.playgroundEnabled = false
	return h
}

// EnablePlayground enables the GraphQL Playground interface
func (h *Handler) EnablePlayground() *Handler {
	h.playgroundEnabled = true
	h.graphiQLEnabled = false
	return h
}

// SetLogLevel sets the logging verbosity level
func (h *Handler) SetLogLevel(level LogLevel) *Handler {
	h.logLevel = level
	return h
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorGray   = "\033[90m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
)

// logf logs a message at the specified level
func (h *Handler) logf(level LogLevel, format string, args ...any) {
	if h.logLevel >= level {
		levelStr := ""
		colorCode := ""
		switch level {
		case LogLevelError:
			levelStr = "ERROR"
			colorCode = colorRed
		case LogLevelWarn:
			levelStr = "WARN"
			colorCode = colorYellow
		case LogLevelInfo:
			levelStr = "INFO"
			colorCode = colorGreen
		case LogLevelDebug:
			levelStr = "DEBUG"
			colorCode = colorGray
		}
		// Use custom logger without default timestamp
		message := fmt.Sprintf("[GraphQL] %s%s%s: %s", colorCode, levelStr, colorReset, fmt.Sprintf(format, args...))
		fmt.Println(message)
	}
}

// ServeHTTP implements the http.Handler interface
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle GraphQL queries
	if r.Method == "POST" || (r.Method == "GET" && r.URL.Query().Get("query") != "") {
		h.ServeGraphQL(w, r)
		return
	}

	// Handle playground/GraphiQL requests
	if r.Method == "GET" && h.acceptsHTML(r) {
		if h.playgroundEnabled {
			h.ServePlayground(w, r)
		} else if h.graphiQLEnabled {
			h.ServeGraphiQL(w, r)
		} else {
			http.Error(w, "GraphQL IDE not enabled", http.StatusNotFound)
		}
		return
	}

	// Method not allowed
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// ServeGraphQL handles GraphQL query execution
func (h *Handler) ServeGraphQL(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	var params graphQLParams

	// Parse request based on method
	if r.Method == "GET" {
		// Parse from query string
		query := r.URL.Query()
		params.Query = query.Get("query")
		params.OperationName = query.Get("operationName")

		// Parse variables if present
		if variables := query.Get("variables"); variables != "" {
			if err := json.Unmarshal([]byte(variables), &params.Variables); err != nil {
				h.writeError(w, "Invalid variables", http.StatusBadRequest)
				return
			}
		}
	} else if r.Method == "POST" {
		// Check content type
		contentType := r.Header.Get("Content-Type")

		if strings.Contains(contentType, "application/json") {
			// Parse JSON body
			body, err := io.ReadAll(r.Body)
			if err != nil {
				h.logf(LogLevelError, "Failed to read request body: %v", err)
				h.writeError(w, "Failed to read request body", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()

			if h.logLevel >= LogLevelDebug {
				h.logf(LogLevelDebug, "Request body: %s", h.truncateString(string(body), 100))
			}

			if err := json.Unmarshal(body, &params); err != nil {
				h.logf(LogLevelError, "JSON parse error: %v", err)
				if h.logLevel >= LogLevelDebug {
					h.logf(LogLevelDebug, "Raw body: %s", h.truncateString(string(body), 100))
				}
				h.writeError(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
				return
			}
		} else if strings.Contains(contentType, "application/graphql") {
			// Raw GraphQL query
			body, err := io.ReadAll(r.Body)
			if err != nil {
				h.logf(LogLevelError, "Failed to read request body: %v", err)
				h.writeError(w, "Failed to read request body", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()

			params.Query = string(body)
			if h.logLevel >= LogLevelDebug {
				h.logf(LogLevelDebug, "GraphQL query: %s", h.truncateString(params.Query, 100))
			}
		} else {
			h.logf(LogLevelWarn, "Unsupported content type: %s", contentType)
			h.writeError(w, "Unsupported content type", http.StatusBadRequest)
			return
		}
	}

	// Extract operation info for logging
	operationType := "query"
	if strings.Contains(strings.ToLower(params.Query), "mutation") {
		operationType = "mutation"
	}

	// Extract operation name from query if available
	operationName := params.OperationName
	if operationName == "" {
		// Try to extract from query string
		if strings.Contains(params.Query, "{") {
			parts := strings.Fields(strings.Split(params.Query, "{")[0])
			if len(parts) >= 2 {
				operationName = parts[1]
			}
		}
	}

	// Log the request at info level
	if operationName != "" {
		h.logf(LogLevelInfo, "%s %s", operationType, operationName)
	} else {
		h.logf(LogLevelInfo, "%s request", operationType)
	}

	// Debug level: show full query and variables
	if h.logLevel >= LogLevelDebug {
		h.logf(LogLevelDebug, "Query: %s", h.truncateString(params.Query, 100))
		if params.Variables != nil && len(params.Variables) > 0 {
			variablesJSON, _ := json.Marshal(params.Variables)
			h.logf(LogLevelDebug, "Variables: %s", h.truncateString(string(variablesJSON), 100))
		}
	}

	// Execute GraphQL query
	result := graphql.Do(graphql.Params{
		Schema:         *h.schema,
		RequestString:  params.Query,
		VariableValues: params.Variables,
		OperationName:  params.OperationName,
		Context:        r.Context(),
	})

	// Log execution results
	duration := time.Since(startTime)
	if len(result.Errors) > 0 {
		h.logf(LogLevelError, "Failed in %v - %d error(s)", duration, len(result.Errors))
		for i, err := range result.Errors {
			h.logf(LogLevelError, "  %d: %s", i+1, err.Message)
		}
	} else {
		h.logf(LogLevelInfo, "Success in %v", duration)
	}

	// Debug level: show full response
	if h.logLevel >= LogLevelDebug && result.Data != nil {
		responseJSON, _ := json.Marshal(result.Data)
		h.logf(LogLevelDebug, "Response: %s", h.truncateString(string(responseJSON), 100))
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")

	if h.pretty {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		encoder.Encode(result)
	} else {
		json.NewEncoder(w).Encode(result)
	}
}

// ServePlayground serves the GraphQL Playground interface
func (h *Handler) ServePlayground(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(playgroundHTML))
}

// ServeGraphiQL serves the GraphiQL interface
func (h *Handler) ServeGraphiQL(w http.ResponseWriter, r *http.Request) {
	// Use the graphql-go/handler package for GraphiQL
	graphiQLHandler := handler.New(&handler.Config{
		Schema:   h.schema,
		Pretty:   h.pretty,
		GraphiQL: true,
	})
	graphiQLHandler.ServeHTTP(w, r)
}

// writeError writes an error response
func (h *Handler) writeError(w http.ResponseWriter, message string, code int) {
	h.logf(LogLevelError, "HTTP %d: %s", code, message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	response := map[string]any{
		"errors": []map[string]any{
			{"message": message},
		},
	}

	json.NewEncoder(w).Encode(response)
}

// acceptsHTML checks if the client accepts HTML responses
func (h *Handler) acceptsHTML(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "text/html") || strings.Contains(accept, "*/*")
}

// truncateString truncates a string to the specified length, adding "..." if truncated
func (h *Handler) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}

// graphQLParams represents the parameters of a GraphQL request
type graphQLParams struct {
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables"`
	OperationName string         `json:"operationName"`
}

// playgroundHTML is the HTML for GraphQL Playground
const playgroundHTML = `
<!DOCTYPE html>
<html>
<head>
    <meta charset=utf-8/>
    <meta name="viewport" content="width=device-width, initial-scale=1"/>
    <title>GraphQL Playground</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/graphql-playground-react@1.7.26/build/static/css/index.css"/>
    <link rel="shortcut icon" href="https://cdn.jsdelivr.net/npm/graphql-playground-react@1.7.26/build/favicon.png"/>
    <script src="https://cdn.jsdelivr.net/npm/graphql-playground-react@1.7.26/build/static/js/middleware.js"></script>
</head>
<body>
    <div id="root"></div>
    <script>
        window.addEventListener('load', function (event) {
            GraphQLPlayground.init(document.getElementById('root'), {
                endpoint: window.location.href,
                settings: {
                    'request.credentials': 'same-origin',
                    'editor.theme': 'light',
                    'editor.fontSize': 14,
                    'editor.fontFamily': '"Fira Code", "Monaco", monospace',
                    'prettier.useTabs': false,
                    'prettier.tabWidth': 2,
                }
            })
        })
    </script>
</body>
</html>
`
