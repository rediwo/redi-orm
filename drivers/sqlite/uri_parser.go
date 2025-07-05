package sqlite

import (
	"fmt"
	"net/url"
	"strings"
)

// SQLiteURIParser implements URIParser for SQLite databases
type SQLiteURIParser struct{}

// NewSQLiteURIParser creates a new SQLite URI parser
func NewSQLiteURIParser() *SQLiteURIParser {
	return &SQLiteURIParser{}
}

// ParseURI parses a SQLite URI and returns a native SQLite file path
// Supported formats:
//   - sqlite:///path/to/database.db
//   - sqlite://:memory:
//   - sqlite://file::memory:?cache=shared
//   - sqlite:///absolute/path/database.db
//   - sqlite://relative/path/database.db
func (p *SQLiteURIParser) ParseURI(uri string) (string, error) {
	// Parse the URI
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("invalid URI format: %w", err)
	}

	// Check if this is a SQLite URI
	if parsedURI.Scheme != "sqlite" && parsedURI.Scheme != "sqlite3" {
		return "", fmt.Errorf("unsupported URI scheme: %s", parsedURI.Scheme)
	}

	// Handle special case for in-memory database
	if parsedURI.Host == "" && strings.HasPrefix(parsedURI.Path, "/:memory:") {
		return ":memory:", nil
	}

	// Handle file::memory:?cache=shared format
	if parsedURI.Host == "file" && strings.HasPrefix(parsedURI.Path, "/:memory:") {
		return "file::memory:?cache=shared", nil
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

	// Add query parameters to file path if present
	if parsedURI.RawQuery != "" {
		path = path + "?" + parsedURI.RawQuery
	}

	return path, nil
}

// GetSupportedSchemes returns the URI schemes this parser supports
func (p *SQLiteURIParser) GetSupportedSchemes() []string {
	return []string{"sqlite", "sqlite3"}
}

// GetDriverType returns the driver type this parser is for
func (p *SQLiteURIParser) GetDriverType() string {
	return "sqlite"
}
