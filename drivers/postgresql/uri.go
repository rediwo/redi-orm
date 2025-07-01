package drivers

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

// PostgreSQLURIParser handles URI parsing for PostgreSQL databases
type PostgreSQLURIParser struct{}

// ParseURI parses PostgreSQL URIs
// Supported formats:
// - postgresql://user:pass@host:port/database
// - postgres://user:pass@host:port/database
// - postgresql://user@host/database
// - postgresql://host/database
func (p *PostgreSQLURIParser) ParseURI(uri string) (types.Config, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return types.Config{}, fmt.Errorf("invalid URI: %w", err)
	}

	// Check if this is a PostgreSQL URI
	if u.Scheme != "postgresql" && u.Scheme != "postgres" {
		return types.Config{}, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	config := types.Config{
		Type: "postgresql",
	}

	// Parse host and port
	host := u.Hostname()
	if host == "" {
		host = "localhost"
	}
	config.Host = host

	portStr := u.Port()
	if portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return types.Config{}, fmt.Errorf("invalid port: %s", portStr)
		}
		config.Port = port
	} else {
		// Default PostgreSQL port
		config.Port = 5432
	}

	// Parse user info
	if u.User != nil {
		config.User = u.User.Username()
		if pass, ok := u.User.Password(); ok {
			config.Password = pass
		}
	}

	// Parse database name from path
	if u.Path != "" && u.Path != "/" {
		config.Database = strings.TrimPrefix(u.Path, "/")
	}

	return config, nil
}

// GetSupportedSchemes returns the schemes this parser supports
func (p *PostgreSQLURIParser) GetSupportedSchemes() []string {
	return []string{"postgresql", "postgres"}
}

// GetDriverType returns the driver type
func (p *PostgreSQLURIParser) GetDriverType() string {
	return "postgresql"
}
