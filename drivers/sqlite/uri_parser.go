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
		Type:    "sqlite",
		Options: make(map[string]string),
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

	// For SQLite URIs:
	// - sqlite:///absolute/path -> /absolute/path (absolute)
	// - sqlite://relative/path -> relative/path (relative)
	// - sqlite:/path -> path (relative)

	// When using sqlite:/// (three slashes), url.Parse returns path as "/absolute/path"
	// We want to keep this as an absolute path
	// When using sqlite:// with a host, we get the host separately
	// When using sqlite:/ (one slash), we want relative paths

	if parsedURI.Host == "" && strings.HasPrefix(path, "/") {
		// Check if original URI had three slashes (indicating absolute path)
		// by looking at the original URI string
		if strings.HasPrefix(uri, "sqlite:///") || strings.HasPrefix(uri, "sqlite3:///") {
			// Keep the path as-is (absolute)
			// path already has leading slash from url.Parse
		} else {
			// sqlite:/path -> path (relative)
			path = path[1:]
		}
	}

	// Handle host part if present (for relative paths)
	if parsedURI.Host != "" && parsedURI.Host != "file" {
		// sqlite://relative/path/database.db -> relative/path/database.db
		path = parsedURI.Host + path
	}

	// Parse query parameters for SQLite options
	query := parsedURI.Query()
	for key, values := range query {
		if len(values) > 0 {
			// Common SQLite connection parameters
			switch key {
			case "mode", "cache", "psow", "nolock", "immutable", "_mutex":
				config.Options[key] = values[0]
			default:
				// Store any other parameters as well
				config.Options[key] = values[0]
			}
		}
	}

	// Add query parameters to file path if present
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
