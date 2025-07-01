package drivers

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

// MySQLURIParser handles URI parsing for MySQL databases
type MySQLURIParser struct{}

// ParseURI parses MySQL URIs
// Supported formats:
// - mysql://user:pass@host:port/database
// - mysql://user@host/database
// - mysql://host/database
func (p *MySQLURIParser) ParseURI(uri string) (types.Config, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return types.Config{}, fmt.Errorf("invalid URI: %w", err)
	}

	// Check if this is a MySQL URI
	if u.Scheme != "mysql" {
		return types.Config{}, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	config := types.Config{
		Type: "mysql",
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
		// Default MySQL port
		config.Port = 3306
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
func (p *MySQLURIParser) GetSupportedSchemes() []string {
	return []string{"mysql"}
}

// GetDriverType returns the driver type
func (p *MySQLURIParser) GetDriverType() string {
	return "mysql"
}
