package postgresql

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// PostgreSQLURIParser implements URIParser for PostgreSQL
type PostgreSQLURIParser struct{}

// NewPostgreSQLURIParser creates a new PostgreSQL URI parser
func NewPostgreSQLURIParser() *PostgreSQLURIParser {
	return &PostgreSQLURIParser{}
}

// ParseURI parses a PostgreSQL connection URI and returns a PostgreSQL DSN
func (p *PostgreSQLURIParser) ParseURI(uri string) (string, error) {
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("invalid URI format: %w", err)
	}

	// Check scheme
	scheme := parsedURI.Scheme
	if scheme != "postgresql" && scheme != "postgres" {
		return "", fmt.Errorf("invalid scheme: %s, expected postgresql or postgres", scheme)
	}

	// Extract host and port
	host := parsedURI.Hostname()
	if host == "" {
		return "", fmt.Errorf("host is required")
	}

	port := 5432 // Default PostgreSQL port
	if parsedURI.Port() != "" {
		parsedPort, err := strconv.Atoi(parsedURI.Port())
		if err != nil {
			return "", fmt.Errorf("invalid port: %s", parsedURI.Port())
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
		return "", fmt.Errorf("database name is required")
	}

	// Build PostgreSQL DSN: key=value pairs
	var dsnParts []string

	if host != "" {
		dsnParts = append(dsnParts, fmt.Sprintf("host=%s", host))
	}
	if port != 5432 {
		dsnParts = append(dsnParts, fmt.Sprintf("port=%d", port))
	}
	if user != "" {
		dsnParts = append(dsnParts, fmt.Sprintf("user=%s", user))
	}
	if password != "" {
		dsnParts = append(dsnParts, fmt.Sprintf("password=%s", password))
	}
	if database != "" {
		dsnParts = append(dsnParts, fmt.Sprintf("dbname=%s", database))
	}

	// Parse query parameters for additional options
	query := parsedURI.Query()
	for key, values := range query {
		if len(values) > 0 {
			dsnParts = append(dsnParts, fmt.Sprintf("%s=%s", key, values[0]))
		}
	}

	dsn := strings.Join(dsnParts, " ")
	return dsn, nil
}

// GetSupportedSchemes returns the URI schemes supported by this parser
func (p *PostgreSQLURIParser) GetSupportedSchemes() []string {
	return []string{"postgresql", "postgres"}
}

// GetDriverType returns the driver type this parser is for
func (p *PostgreSQLURIParser) GetDriverType() string {
	return "postgresql"
}
