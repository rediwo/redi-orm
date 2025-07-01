package sqlite

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

// SQLiteURIParser implements URIParser for SQLite databases
type SQLiteURIParser struct{}

// NewSQLiteURIParser creates a new SQLite URI parser
func NewSQLiteURIParser() *SQLiteURIParser {
	return &SQLiteURIParser{}
}

// ParseURI parses a SQLite URI and returns a Config
// Supported formats:
//   - sqlite:///path/to/database.db
//   - sqlite://:memory:
//   - sqlite://file::memory:?cache=shared
//   - sqlite:///absolute/path/database.db
//   - sqlite://relative/path/database.db
func (p *SQLiteURIParser) ParseURI(uri string) (types.Config, error) {
	// Parse the URI
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return types.Config{}, fmt.Errorf("invalid URI format: %w", err)
	}

	// Check if this is a SQLite URI
	if parsedURI.Scheme != "sqlite" && parsedURI.Scheme != "sqlite3" {
		return types.Config{}, fmt.Errorf("unsupported URI scheme: %s", parsedURI.Scheme)
	}

	config := types.Config{
		Type: "sqlite",
	}

	// Handle special case for in-memory database
	if parsedURI.Host == "" && strings.HasPrefix(parsedURI.Path, "/:memory:") {
		config.FilePath = ":memory:"
		return config, nil
	}

	// Handle file::memory:?cache=shared format
	if parsedURI.Host == "file" && strings.HasPrefix(parsedURI.Path, "/:memory:") {
		config.FilePath = "file::memory:?cache=shared"
		return config, nil
	}

	// Handle regular file paths
	path := parsedURI.Path
	
	// Remove leading slashes for absolute paths
	if strings.HasPrefix(path, "///") {
		// sqlite:///absolute/path -> /absolute/path
		path = path[2:]
	} else if strings.HasPrefix(path, "//") {
		// sqlite://relative/path -> relative/path
		path = path[2:]
	} else if strings.HasPrefix(path, "/") && parsedURI.Host == "" {
		// sqlite:/path -> path (relative)
		path = path[1:]
	}

	// Handle host part if present (for relative paths)
	if parsedURI.Host != "" && parsedURI.Host != "file" {
		// sqlite://relative/path/database.db -> relative/path/database.db
		path = parsedURI.Host + path
	}

	// Add query parameters if present
	if parsedURI.RawQuery != "" {
		path = path + "?" + parsedURI.RawQuery
	}

	config.FilePath = path
	return config, nil
}

// GetSupportedSchemes returns the URI schemes this parser supports
func (p *SQLiteURIParser) GetSupportedSchemes() []string {
	return []string{"sqlite", "sqlite3"}
}

// GetDriverType returns the driver type this parser is for
func (p *SQLiteURIParser) GetDriverType() string {
	return "sqlite"
}