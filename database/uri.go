package database

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"github.com/rediwo/redi-orm/types"
)

// ParseURI parses a database URI and returns a Config
// Supported formats:
// - sqlite:///path/to/database.db
// - sqlite://:memory:
// - mysql://user:pass@host:port/database
// - postgresql://user:pass@host:port/database
// - postgres://user:pass@host:port/database
func ParseURI(uri string) (types.Config, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return types.Config{}, fmt.Errorf("invalid URI: %w", err)
	}

	config := types.Config{}

	// Determine database type from scheme
	switch u.Scheme {
	case "sqlite":
		config.Type = types.SQLite
		// For SQLite, the path is the file path
		if u.Host == "" && u.Path != "" {
			config.FilePath = u.Path
		} else if u.Host == ":memory:" || u.Path == "/:memory:" {
			config.FilePath = ":memory:"
		} else {
			config.FilePath = u.Host + u.Path
		}

	case "mysql":
		config.Type = types.MySQL
		if err := parseNetworkURI(u, &config); err != nil {
			return config, err
		}

	case "postgresql", "postgres":
		config.Type = types.PostgreSQL
		if err := parseNetworkURI(u, &config); err != nil {
			return config, err
		}

	default:
		return config, fmt.Errorf("unsupported database type: %s", u.Scheme)
	}

	return config, nil
}

// parseNetworkURI parses network-based database URIs (MySQL, PostgreSQL)
func parseNetworkURI(u *url.URL, config *types.Config) error {
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
			return fmt.Errorf("invalid port: %s", portStr)
		}
		config.Port = port
	} else {
		// Default ports
		switch config.Type {
		case types.MySQL:
			config.Port = 3306
		case types.PostgreSQL:
			config.Port = 5432
		}
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

	return nil
}