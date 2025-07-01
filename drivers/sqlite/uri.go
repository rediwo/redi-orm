package drivers

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

// SQLiteURIParser handles URI parsing for SQLite databases
type SQLiteURIParser struct{}

// ParseURI parses SQLite URIs
// Supported formats:
// - sqlite:///path/to/database.db
// - sqlite://:memory:
// - sqlite://./relative/path.db
func (p *SQLiteURIParser) ParseURI(uri string) (types.Config, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return types.Config{}, fmt.Errorf("invalid URI: %w", err)
	}

	// Check if this is a SQLite URI
	if u.Scheme != "sqlite" {
		return types.Config{}, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	config := types.Config{
		Type: "sqlite",
	}

	// Parse SQLite file path
	if u.Host == ":memory:" || u.Path == "/:memory:" || u.Path == ":memory:" {
		// In-memory database
		config.FilePath = ":memory:"
	} else if u.Host == "" && u.Path != "" {
		// File path in the path component: sqlite:///path/to/file.db
		config.FilePath = u.Path
	} else if u.Host != "" && u.Path == "" {
		// File path in the host component: sqlite://file.db
		config.FilePath = u.Host
	} else {
		// Combination: sqlite://host/path
		config.FilePath = u.Host + u.Path
	}

	// Handle empty path
	if config.FilePath == "" || config.FilePath == "/" {
		return types.Config{}, fmt.Errorf("SQLite URI must specify a file path or :memory:")
	}

	// Clean up path
	if strings.HasPrefix(config.FilePath, "/") && !strings.HasPrefix(config.FilePath, "//") {
		// Absolute path, keep as-is
	} else if config.FilePath == ":memory:" {
		// Special case, keep as-is
	} else {
		// Relative path or other format
	}

	return config, nil
}

// GetSupportedSchemes returns the schemes this parser supports
func (p *SQLiteURIParser) GetSupportedSchemes() []string {
	return []string{"sqlite"}
}

// GetDriverType returns the driver type
func (p *SQLiteURIParser) GetDriverType() string {
	return "sqlite"
}
