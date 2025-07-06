package mongodb

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

// MongoDBURIParser handles MongoDB URI parsing
type MongoDBURIParser struct{}

// NewMongoDBURIParser creates a new MongoDB URI parser
func NewMongoDBURIParser() *MongoDBURIParser {
	return &MongoDBURIParser{}
}

// ParseURI parses a MongoDB URI and returns the native MongoDB connection string
func (p *MongoDBURIParser) ParseURI(uri string) (string, error) {
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("invalid URI format: %w", err)
	}

	// Validate scheme
	if !p.isValidScheme(parsedURI.Scheme) {
		return "", fmt.Errorf("unsupported scheme: %s", parsedURI.Scheme)
	}

	// MongoDB URIs are already in the correct format for the official driver
	// Just validate required components
	if parsedURI.Host == "" {
		return "", fmt.Errorf("host is required in MongoDB URI")
	}

	// Ensure the URI starts with mongodb:// or mongodb+srv://
	if !strings.HasPrefix(uri, "mongodb://") && !strings.HasPrefix(uri, "mongodb+srv://") {
		return "", fmt.Errorf("MongoDB URI must start with mongodb:// or mongodb+srv://")
	}

	// Return the URI as-is since MongoDB driver expects standard MongoDB URI format
	return uri, nil
}

// GetSupportedSchemes returns the URI schemes supported by this parser
func (p *MongoDBURIParser) GetSupportedSchemes() []string {
	return []string{"mongodb", "mongodb+srv"}
}

// GetDriverType returns the driver type for this parser
func (p *MongoDBURIParser) GetDriverType() string {
	return string(types.DriverMongoDB)
}

// isValidScheme checks if the scheme is supported
func (p *MongoDBURIParser) isValidScheme(scheme string) bool {
	for _, s := range p.GetSupportedSchemes() {
		if s == scheme {
			return true
		}
	}
	return false
}
