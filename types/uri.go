package types

// URIParser defines the interface for database-specific URI parsing
type URIParser interface {
	// ParseURI parses a database URI and returns a native database URI/DSN if the URI is supported by this driver
	// Returns an error if the URI format is not supported or invalid
	ParseURI(uri string) (string, error)

	// GetSupportedSchemes returns the URI schemes this parser supports (e.g., ["sqlite"])
	GetSupportedSchemes() []string

	// GetDriverType returns the driver type this parser is for
	GetDriverType() string
}
