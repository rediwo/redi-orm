package mysql

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

// MySQLURIParser implements URIParser for MySQL databases
type MySQLURIParser struct{}

// NewMySQLURIParser creates a new MySQL URI parser
func NewMySQLURIParser() *MySQLURIParser {
	return &MySQLURIParser{}
}

// ParseURI parses a MySQL URI and returns a Config
// Supported formats:
//   - mysql://user:password@host:port/database
//   - mysql://user:password@host/database (default port 3306)
//   - mysql://user@host/database (no password)
//   - mysql://host/database (no auth)
func (p *MySQLURIParser) ParseURI(uri string) (types.Config, error) {
	// Parse the URI
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return types.Config{}, fmt.Errorf("invalid URI format: %w", err)
	}

	// Check if this is a MySQL URI
	if parsedURI.Scheme != "mysql" && parsedURI.Scheme != "mysql2" {
		return types.Config{}, fmt.Errorf("unsupported URI scheme: %s", parsedURI.Scheme)
	}

	config := types.Config{
		Type: "mysql",
	}

	// Extract host and port
	host := parsedURI.Hostname()
	if host == "" {
		return types.Config{}, fmt.Errorf("host is required in MySQL URI")
	}
	config.Host = host

	// Extract port (default to 3306)
	portStr := parsedURI.Port()
	if portStr == "" {
		config.Port = 3306
	} else {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return types.Config{}, fmt.Errorf("invalid port: %s", portStr)
		}
		config.Port = port
	}

	// Extract database name
	if parsedURI.Path == "" || parsedURI.Path == "/" {
		return types.Config{}, fmt.Errorf("database name is required in MySQL URI")
	}
	config.Database = strings.TrimPrefix(parsedURI.Path, "/")

	// Extract user and password
	if parsedURI.User != nil {
		config.User = parsedURI.User.Username()
		if password, hasPassword := parsedURI.User.Password(); hasPassword {
			config.Password = password
		}
	}

	// Initialize Options map
	config.Options = make(map[string]string)

	// Parse query parameters for additional options
	query := parsedURI.Query()
	for key, values := range query {
		if len(values) > 0 {
			// Common MySQL connection parameters
			switch key {
			case "charset", "collation", "parseTime", "loc", "timeout",
			     "readTimeout", "writeTimeout", "allowNativePasswords",
			     "allowOldPasswords", "clientFoundRows", "columnsWithAlias",
			     "interpolateParams", "multiStatements", "tls":
				config.Options[key] = values[0]
			default:
				// Store any other parameters as well
				config.Options[key] = values[0]
			}
		}
	}

	// Set default charset if not specified
	if _, ok := config.Options["charset"]; !ok {
		config.Options["charset"] = "utf8mb4"
	}

	// Set default parseTime if not specified
	if _, ok := config.Options["parseTime"]; !ok {
		config.Options["parseTime"] = "true"
	}

	return config, nil
}

// GetSupportedSchemes returns the URI schemes this parser supports
func (p *MySQLURIParser) GetSupportedSchemes() []string {
	return []string{"mysql", "mysql2"}
}

// GetDriverType returns the driver type this parser is for
func (p *MySQLURIParser) GetDriverType() string {
	return "mysql"
}
