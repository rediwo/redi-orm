package mysql

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// MySQLURIParser implements URIParser for MySQL databases
type MySQLURIParser struct{}

// NewMySQLURIParser creates a new MySQL URI parser
func NewMySQLURIParser() *MySQLURIParser {
	return &MySQLURIParser{}
}

// ParseURI parses a MySQL URI and returns a MySQL DSN
// Supported formats:
//   - mysql://user:password@host:port/database
//   - mysql://user:password@host/database (default port 3306)
//   - mysql://user@host/database (no password)
//   - mysql://host/database (no auth)
func (p *MySQLURIParser) ParseURI(uri string) (string, error) {
	// Parse the URI
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("invalid URI format: %w", err)
	}

	// Check if this is a MySQL URI
	if parsedURI.Scheme != "mysql" && parsedURI.Scheme != "mysql2" {
		return "", fmt.Errorf("unsupported URI scheme: %s", parsedURI.Scheme)
	}

	// Extract host and port
	host := parsedURI.Hostname()
	if host == "" {
		return "", fmt.Errorf("host is required in MySQL URI")
	}

	// Extract port (default to 3306)
	portStr := parsedURI.Port()
	port := 3306
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil {
			return "", fmt.Errorf("invalid port: %s", portStr)
		}
		port = p
	}

	// Extract database name
	if parsedURI.Path == "" || parsedURI.Path == "/" {
		return "", fmt.Errorf("database name is required in MySQL URI")
	}
	database := strings.TrimPrefix(parsedURI.Path, "/")

	// Extract user and password
	var user, password string
	if parsedURI.User != nil {
		user = parsedURI.User.Username()
		if p, hasPassword := parsedURI.User.Password(); hasPassword {
			password = p
		}
	}

	// Build DSN: user:password@tcp(host:port)/database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, port, database)

	// Parse query parameters and add to DSN
	var params []string
	query := parsedURI.Query()
	for key, values := range query {
		if len(values) > 0 {
			params = append(params, fmt.Sprintf("%s=%s", key, values[0]))
		}
	}

	// Set default charset if not specified
	hasCharset := false
	for _, param := range params {
		if strings.HasPrefix(param, "charset=") {
			hasCharset = true
			break
		}
	}
	if !hasCharset {
		params = append(params, "charset=utf8mb4")
	}

	// Set default parseTime if not specified
	hasParseTime := false
	for _, param := range params {
		if strings.HasPrefix(param, "parseTime=") {
			hasParseTime = true
			break
		}
	}
	if !hasParseTime {
		params = append(params, "parseTime=true")
	}

	if len(params) > 0 {
		dsn += "?" + strings.Join(params, "&")
	}

	return dsn, nil
}

// GetSupportedSchemes returns the URI schemes this parser supports
func (p *MySQLURIParser) GetSupportedSchemes() []string {
	return []string{"mysql", "mysql2"}
}

// GetDriverType returns the driver type this parser is for
func (p *MySQLURIParser) GetDriverType() string {
	return "mysql"
}
