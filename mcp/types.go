package mcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// Server represents an MCP server instance
type Server struct {
	db        types.Database
	schemas   map[string]*schema.Schema
	transport Transport
	handler   *Handler
	logger    utils.Logger
	config    ServerConfig
	security  *SecurityManager
}

// ServerConfig holds MCP server configuration
type ServerConfig struct {
	DatabaseURI   string
	SchemaPath    string
	Transport     string   // "stdio" or "http"
	Port          int      // For HTTP transport
	ReadOnly      bool     // Default: true
	MaxQueryRows  int      // Default: 1000
	AllowedTables []string // Empty = all tables
	LogLevel      string
	
	// Security settings
	Security SecurityConfig
}

// Transport interface for different communication methods
type Transport interface {
	Start() error
	Stop() error
	Send(message json.RawMessage) error
	Receive() (json.RawMessage, error)
}

// JSON-RPC 2.0 message types
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
	ID      interface{}     `json:"id"`
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error implements the error interface
func (e *JSONRPCError) Error() string {
	return e.Message
}

// MCP protocol types
type InitializeParams struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ClientInfo      *ClientInfo  `json:"clientInfo,omitempty"`
}

type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
}

type Capabilities struct {
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
}

type ResourcesCapability struct{}
type ToolsCapability struct{}
type PromptsCapability struct{}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Resource types
type Resource struct {
	URI         string                 `json:"uri"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	MimeType    string                 `json:"mimeType,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

type ListResourcesResult struct {
	Resources []Resource `json:"resources"`
}

type ReadResourceParams struct {
	URI string `json:"uri"`
}

type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

// Tool types
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ToolContent struct {
	Type     string                 `json:"type"`
	Text     string                 `json:"text,omitempty"`
	Blob     string                 `json:"blob,omitempty"`
	Resource *Resource              `json:"resource,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Prompt types
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type ListPromptsResult struct {
	Prompts []Prompt `json:"prompts"`
}

type GetPromptParams struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

type GetPromptResult struct {
	Messages []PromptMessage `json:"messages"`
}

type PromptMessage struct {
	Role    string       `json:"role"`
	Content PromptContent `json:"content"`
}

type PromptContent struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	Resource *Resource `json:"resource,omitempty"`
}

// HandlerFunc represents a JSON-RPC method handler
type HandlerFunc func(ctx context.Context, params json.RawMessage) (interface{}, error)

// Error codes
const (
	ErrorCodeParseError     = -32700
	ErrorCodeInvalidRequest = -32600
	ErrorCodeMethodNotFound = -32601
	ErrorCodeInvalidParams  = -32602
	ErrorCodeInternalError  = -32603
)

// Resource URI prefixes
const (
	ResourceSchemaPrefix = "schema://"
	ResourceTablePrefix  = "table://"
	ResourceDataPrefix   = "data://"
	ResourceModelPrefix  = "model://"
)

// Core MCP methods
const (
	MethodInitialize    = "initialize"
	MethodResourcesList = "resources/list"
	MethodResourcesRead = "resources/read"
	MethodToolsList     = "tools/list"
	MethodToolsCall     = "tools/call"
	MethodPromptsList   = "prompts/list"
	MethodPromptsGet    = "prompts/get"
)

// Advanced tool types
type BatchQuery struct {
	SQL        string        `json:"sql"`
	Parameters []interface{} `json:"parameters,omitempty"`
	Label      string        `json:"label,omitempty"`
}

type BatchQueryResult struct {
	Index     int                      `json:"index"`
	Label     string                   `json:"label,omitempty"`
	SQL       string                   `json:"sql"`
	Results   []map[string]interface{} `json:"results,omitempty"`
	Count     int                      `json:"count"`
	Error     string                   `json:"error,omitempty"`
	Success   bool                     `json:"success"`
	Truncated bool                     `json:"truncated,omitempty"`
}

type TableAnalysis struct {
	Table      string                   `json:"table"`
	TotalRows  int64                    `json:"total_rows"`
	SampleSize int                      `json:"sample_size"`
	Schema     interface{}              `json:"schema"`
	Statistics map[string]ColumnStats   `json:"statistics"`
}

type ColumnStats struct {
	DataType    string      `json:"data_type"`
	NullCount   int         `json:"null_count"`
	UniqueCount int         `json:"unique_count"`
	MinValue    interface{} `json:"min_value,omitempty"`
	MaxValue    interface{} `json:"max_value,omitempty"`
	SampleValues []interface{} `json:"sample_values,omitempty"`
}

// ORM-based types for model management
type ModelDefinition struct {
	Name       string             `json:"name"`
	Fields     []FieldDefinition  `json:"fields"`
	Relations  []RelationDefinition `json:"relations,omitempty"`
	Indexes    []IndexDefinition    `json:"indexes,omitempty"`
	Attributes []string            `json:"attributes,omitempty"`
}

type FieldDefinition struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Attributes []string `json:"attributes,omitempty"`
	Default    interface{} `json:"default,omitempty"`
}

type RelationDefinition struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Model      string   `json:"model"`
	Fields     []string `json:"fields,omitempty"`
	References []string `json:"references,omitempty"`
	Attributes []string `json:"attributes,omitempty"`
}

type IndexDefinition struct {
	Name   string   `json:"name,omitempty"`
	Fields []string `json:"fields"`
	Type   string   `json:"type,omitempty"` // UNIQUE, INDEX, etc.
}

// ORM query types
type ORMQuery struct {
	Model   string                 `json:"model"`
	Action  string                 `json:"action"` // findMany, findUnique, create, update, delete, count, aggregate
	Query   map[string]interface{} `json:"query"`
}

type ORMWhereCondition struct {
	Field    string      `json:"field,omitempty"`
	Operator string      `json:"operator,omitempty"`
	Value    interface{} `json:"value,omitempty"`
	AND      []ORMWhereCondition `json:"AND,omitempty"`
	OR       []ORMWhereCondition `json:"OR,omitempty"`
	NOT      *ORMWhereCondition  `json:"NOT,omitempty"`
}

// Schema management types
type SchemaOperation struct {
	Type      string                 `json:"type"` // create, update, delete
	Model     string                 `json:"model,omitempty"`
	Changes   map[string]interface{} `json:"changes,omitempty"`
}

type MigrationInfo struct {
	Version   string    `json:"version"`
	Name      string    `json:"name"`
	AppliedAt time.Time `json:"applied_at,omitempty"`
	SQL       string    `json:"sql,omitempty"`
}