package types

import (
	"database/sql"
)

// DatabaseType represents supported database types
type DatabaseType string

const (
	SQLite     DatabaseType = "sqlite"
	MySQL      DatabaseType = "mysql"
	PostgreSQL DatabaseType = "postgresql"
)

// Config holds database connection configuration
type Config struct {
	Type     DatabaseType
	Host     string
	Port     int
	Database string
	User     string
	Password string
	FilePath string // for SQLite
}

// QueryBuilder interface for building database queries
type QueryBuilder interface {
	Where(field string, operator string, value interface{}) QueryBuilder
	WhereIn(field string, values []interface{}) QueryBuilder
	OrderBy(field string, direction string) QueryBuilder
	Limit(limit int) QueryBuilder
	Offset(offset int) QueryBuilder
	Execute() ([]map[string]interface{}, error)
	First() (map[string]interface{}, error)
	Count() (int64, error)
}

// Transaction interface for database transactions
type Transaction interface {
	Commit() error
	Rollback() error
	Insert(tableName string, data map[string]interface{}) (int64, error)
	Update(tableName string, id interface{}, data map[string]interface{}) error
	Delete(tableName string, id interface{}) error
}

// Database interface defines all database operations
type Database interface {
	Connect() error
	Close() error
	CreateTable(schema interface{}) error // Using interface{} to avoid circular dependency with schema
	DropTable(tableName string) error

	// CRUD operations
	Insert(tableName string, data map[string]interface{}) (int64, error)
	FindByID(tableName string, id interface{}) (map[string]interface{}, error)
	Find(tableName string, conditions map[string]interface{}, limit, offset int) ([]map[string]interface{}, error)
	Update(tableName string, id interface{}, data map[string]interface{}) error
	Delete(tableName string, id interface{}) error

	// Query builder
	Select(tableName string, columns []string) QueryBuilder

	// Transaction
	Begin() (Transaction, error)

	// Raw query
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}
