package postgresql

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

// PostgreSQLURIParser implements URIParser for PostgreSQL
type PostgreSQLURIParser struct{}

// NewPostgreSQLURIParser creates a new PostgreSQL URI parser
func NewPostgreSQLURIParser() *PostgreSQLURIParser {
	return &PostgreSQLURIParser{}
}

// ParseURI parses a PostgreSQL connection URI
func (p *PostgreSQLURIParser) ParseURI(uri string) (types.Config, error) {
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return types.Config{}, fmt.Errorf("invalid URI format: %w", err)
	}

	// Check scheme
	scheme := parsedURI.Scheme
	if scheme != "postgresql" && scheme != "postgres" {
		return types.Config{}, fmt.Errorf("invalid scheme: %s, expected postgresql or postgres", scheme)
	}

	// Extract host and port
	host := parsedURI.Hostname()
	if host == "" {
		return types.Config{}, fmt.Errorf("host is required")
	}

	port := 5432 // Default PostgreSQL port
	if parsedURI.Port() != "" {
		parsedPort, err := strconv.Atoi(parsedURI.Port())
		if err != nil {
			return types.Config{}, fmt.Errorf("invalid port: %s", parsedURI.Port())
		}
		port = parsedPort
	}

	// Extract user info
	var user, password string
	if parsedURI.User != nil {
		user = parsedURI.User.Username()
		password, _ = parsedURI.User.Password()
	}

	// Extract database name
	database := strings.TrimPrefix(parsedURI.Path, "/")
	if database == "" {
		return types.Config{}, fmt.Errorf("database name is required")
	}

	config := types.Config{
		Type:     "postgresql",
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Database: database,
	}

	// Parse query parameters for additional options
	// Note: Currently we don't have an Options field in types.Config
	// TODO: Add support for additional connection options when types.Config supports it

	return config, nil
}

// GetSupportedSchemes returns the URI schemes supported by this parser
func (p *PostgreSQLURIParser) GetSupportedSchemes() []string {
	return []string{"postgresql", "postgres"}
}

// GetDriverType returns the driver type this parser is for
func (p *PostgreSQLURIParser) GetDriverType() string {
	return "postgresql"
}