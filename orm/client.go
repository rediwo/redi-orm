package orm

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// Client is the main entry point for the ORM API
type Client struct {
	db            types.Database
	typeConverter *TypeConverter
}

// ClientOption is a functional option for configuring the client
type ClientOption func(*Client)

// NewClient creates a new ORM client
func NewClient(db types.Database, opts ...ClientOption) *Client {
	client := &Client{
		db:            db,
		typeConverter: NewTypeConverter(db.GetCapabilities()),
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	return client
}

// WithTypeConverter sets a custom type converter
func WithTypeConverter(tc *TypeConverter) ClientOption {
	return func(c *Client) {
		c.typeConverter = tc
	}
}

// Model returns a model query builder for the specified model
func (c *Client) Model(modelName string) *Model {
	return &Model{
		client:    c,
		modelName: modelName,
		db:        c.db,
	}
}

// GetDB returns the underlying database
func (c *Client) GetDB() types.Database {
	return c.db
}

// Transaction executes a function within a database transaction
func (c *Client) Transaction(fn func(tx *Client) error) error {
	ctx := context.Background()

	// Use the Transaction method provided by the Database interface
	return c.db.Transaction(ctx, func(tx types.Transaction) error {
		// Create a new client with transaction-wrapped database
		// We need to create a wrapper that implements Database interface for the transaction
		txClient := &Client{
			db:            &transactionDatabase{tx: tx, originalDB: c.db},
			typeConverter: c.typeConverter,
		}

		return fn(txClient)
	})
}

// transactionDatabase wraps a Transaction to implement the Database interface
type transactionDatabase struct {
	tx         types.Transaction
	originalDB types.Database
}

// Implement Database interface by delegating to transaction
func (td *transactionDatabase) Model(modelName string) types.ModelQuery {
	return td.tx.Model(modelName)
}

func (td *transactionDatabase) Raw(sql string, args ...any) types.RawQuery {
	return td.tx.Raw(sql, args...)
}

// Delegate other methods to original database
func (td *transactionDatabase) Connect(ctx context.Context) error {
	return td.originalDB.Connect(ctx)
}

func (td *transactionDatabase) Close() error {
	return td.originalDB.Close()
}

func (td *transactionDatabase) Ping(ctx context.Context) error {
	return td.originalDB.Ping(ctx)
}

func (td *transactionDatabase) RegisterSchema(modelName string, schema *schema.Schema) error {
	return td.originalDB.RegisterSchema(modelName, schema)
}

func (td *transactionDatabase) GetSchema(modelName string) (*schema.Schema, error) {
	return td.originalDB.GetSchema(modelName)
}

func (td *transactionDatabase) CreateModel(ctx context.Context, modelName string) error {
	return td.originalDB.CreateModel(ctx, modelName)
}

func (td *transactionDatabase) DropModel(ctx context.Context, modelName string) error {
	return td.originalDB.DropModel(ctx, modelName)
}

func (td *transactionDatabase) LoadSchema(ctx context.Context, schemaContent string) error {
	return td.originalDB.LoadSchema(ctx, schemaContent)
}

func (td *transactionDatabase) LoadSchemaFrom(ctx context.Context, filename string) error {
	return td.originalDB.LoadSchemaFrom(ctx, filename)
}

func (td *transactionDatabase) SyncSchemas(ctx context.Context) error {
	return td.originalDB.SyncSchemas(ctx)
}

func (td *transactionDatabase) Begin(ctx context.Context) (types.Transaction, error) {
	// Nested transactions not supported in this wrapper
	return nil, fmt.Errorf("nested transactions not supported")
}

func (td *transactionDatabase) Transaction(ctx context.Context, fn func(tx types.Transaction) error) error {
	// Use the existing transaction
	return fn(td.tx)
}

func (td *transactionDatabase) GetModels() []string {
	return td.originalDB.GetModels()
}

func (td *transactionDatabase) GetModelSchema(modelName string) (*schema.Schema, error) {
	return td.originalDB.GetModelSchema(modelName)
}

func (td *transactionDatabase) GetDriverType() string {
	return td.originalDB.GetDriverType()
}

func (td *transactionDatabase) GetCapabilities() types.DriverCapabilities {
	return td.originalDB.GetCapabilities()
}

func (td *transactionDatabase) ResolveTableName(modelName string) (string, error) {
	return td.originalDB.ResolveTableName(modelName)
}

func (td *transactionDatabase) ResolveFieldName(modelName, fieldName string) (string, error) {
	return td.originalDB.ResolveFieldName(modelName, fieldName)
}

func (td *transactionDatabase) ResolveFieldNames(modelName string, fieldNames []string) ([]string, error) {
	return td.originalDB.ResolveFieldNames(modelName, fieldNames)
}

func (td *transactionDatabase) Exec(query string, args ...any) (sql.Result, error) {
	return td.originalDB.Exec(query, args...)
}

func (td *transactionDatabase) Query(query string, args ...any) (*sql.Rows, error) {
	return td.originalDB.Query(query, args...)
}

func (td *transactionDatabase) QueryRow(query string, args ...any) *sql.Row {
	return td.originalDB.QueryRow(query, args...)
}

func (td *transactionDatabase) GetMigrator() types.DatabaseMigrator {
	return td.originalDB.GetMigrator()
}

// parseJSON parses a JSON string into a map
func parseJSON(jsonStr string) (map[string]any, error) {
	var result map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return result, nil
}
