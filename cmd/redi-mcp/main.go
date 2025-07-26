package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/rediwo/redi-orm/drivers/mongodb"    // Import MongoDB driver
	_ "github.com/rediwo/redi-orm/drivers/mysql"      // Import MySQL driver
	_ "github.com/rediwo/redi-orm/drivers/postgresql" // Import PostgreSQL driver
	_ "github.com/rediwo/redi-orm/drivers/sqlite"     // Import SQLite driver
	"github.com/rediwo/redi-orm/mcp"
)

// Version is injected at build time via -ldflags
var version = "dev"

const (
	usage = `RediORM MCP Server - Model Context Protocol for AI Assistants

Usage:
  redi-mcp [flags]

Description:
  MCP (Model Context Protocol) server that allows AI assistants to interact
  with databases through RediORM. By default uses stdio for local AI assistants.
  Specify --port to enable HTTP server for remote access.

Flags:
  --db              Database URI (required)
                    Examples:
                    - sqlite://./myapp.db
                    - mysql://user:pass@localhost:3306/dbname
                    - postgresql://user:pass@localhost:5432/dbname
                    - mongodb://localhost:27017/dbname
  
  --schema          Path to schema file or directory (default: ./schema.prisma)
                    Supports Prisma-style schema definitions
                    If directory: loads all .prisma files in the directory
  
  --port            Enable HTTP server on specified port
                    If not specified, uses stdio mode for local AI assistants
  
  --log-level       Logging level (debug|info|warn|error|none)
                    Default: info
  
  Security Flags:
  --api-key         API key for HTTP transport authentication
  --enable-auth     Enable authentication for HTTP transport
  --read-only       Enable read-only mode (default: true)
  --rate-limit      Requests per minute rate limit (default: 60)
  
  --help            Show help message
  --version         Show version information

Examples:
  # Start MCP server with stdio (default for local AI assistants)
  redi-mcp --db=sqlite://./myapp.db
  
  # Start MCP server with HTTP transport
  redi-mcp --db=postgresql://user:pass@localhost/db --port=3000
  
  # Start with security settings
  redi-mcp --db=mysql://user:pass@localhost/db --enable-auth --api-key=secret --rate-limit=100
  
  # Start with write access enabled
  redi-mcp --db=sqlite://./myapp.db --read-only=false

AI Assistant Integration:
  For stdio (default), configure your AI assistant to run:
    redi-mcp --db=<your-database-uri>
  
  For HTTP transport, specify port and point your AI assistant to:
    redi-mcp --db=<your-database-uri> --port=3000
    Connect to: http://localhost:3000
`
)

func main() {
	// Define flags
	var (
		dbURI       string
		schemaPath  string
		port        int
		logLevel    string
		help        bool
		showVersion bool

		// Security flags
		apiKey       string
		enableAuth   bool
		readOnlyMode bool
		rateLimit    int
	)

	flag.StringVar(&dbURI, "db", "", "Database URI")
	flag.StringVar(&schemaPath, "schema", "./schema.prisma", "Path to schema file or directory")
	flag.IntVar(&port, "port", 0, "Enable HTTP server on specified port (0 = stdio mode)")
	flag.StringVar(&logLevel, "log-level", "info", "Logging level")
	flag.BoolVar(&help, "help", false, "Show help message")
	flag.BoolVar(&showVersion, "version", false, "Show version information")

	// Security flags
	flag.StringVar(&apiKey, "api-key", "", "API key for HTTP transport authentication")
	flag.BoolVar(&enableAuth, "enable-auth", false, "Enable authentication for HTTP transport")
	flag.BoolVar(&readOnlyMode, "read-only", false, "Enable read-only mode (default: false)")
	flag.IntVar(&rateLimit, "rate-limit", 60, "Requests per minute rate limit")

	// Custom usage
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}

	// Parse flags
	flag.Parse()

	// Handle help
	if help {
		flag.Usage()
		os.Exit(0)
	}

	// Handle version
	if showVersion {
		fmt.Printf("RediORM MCP Server %s\n", version)
		os.Exit(0)
	}

	// Validate required flags
	if dbURI == "" {
		fmt.Fprintln(os.Stderr, "Error: --db flag is required")
		fmt.Fprintln(os.Stderr, "")
		flag.Usage()
		os.Exit(1)
	}

	// Determine transport mode based on port
	transport := "stdio"
	if port > 0 {
		transport = "http"
	}

	// Run MCP server
	ctx := context.Background()
	runMCP(ctx, dbURI, schemaPath, port, transport, logLevel, apiKey, enableAuth, readOnlyMode, rateLimit)
}

func runMCP(ctx context.Context, dbURI, schemaPath string, port int, transport, logLevel, apiKey string, enableAuth, readOnlyMode bool, rateLimit int) {

	// Create MCP server configuration
	config := mcp.ServerConfig{
		DatabaseURI: dbURI,
		SchemaPath:  schemaPath,
		Transport:   transport,
		Port:        port,
		LogLevel:    logLevel,
		ReadOnly:    readOnlyMode,
		Security: mcp.SecurityConfig{
			EnableAuth:      enableAuth,
			APIKey:          apiKey,
			EnableRateLimit: rateLimit > 0,
			RequestsPerMin:  rateLimit,
			ReadOnlyMode:    readOnlyMode,
		},
		Version: version,
	}

	// Create MCP server using SDK
	server, err := mcp.NewSDKServer(config)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	// Use stderr for stdio transport to avoid polluting JSON-RPC stream
	var output = os.Stdout
	if transport == "stdio" {
		output = os.Stderr
	}

	// Start server
	fmt.Fprintf(output, "Starting MCP server v%s\n", version)
	fmt.Fprintf(output, "  Database: %s\n", dbURI)
	fmt.Fprintf(output, "  Transport: %s\n", transport)
	if transport == "http" {
		fmt.Fprintf(output, "  Port: %d\n", port)
		fmt.Fprintf(output, "  Authentication: %t\n", enableAuth)
		if enableAuth {
			fmt.Fprintf(output, "  API Key: %s\n", "***")
		}
		// Rate limiting only applies to HTTP transport
		if rateLimit > 0 {
			fmt.Fprintf(output, "  Rate Limit: %d req/min\n", rateLimit)
		}
	}
	// Show write mode warning with red color when enabled
	if !readOnlyMode {
		fmt.Fprintf(output, "  Mode: Read/Write \033[31m(CAUTION: Write operations enabled)\033[0m\n")
	}
	fmt.Fprintf(output, "  Log level: %s\n", logLevel)
	fmt.Fprintln(output)

	if transport == "stdio" {
		fmt.Fprintln(output, "MCP server is ready for JSON-RPC communication via stdio")
		fmt.Fprintln(output, "Connect your AI assistant to this process")
	} else {
		fmt.Fprintf(output, "MCP server is ready at http://localhost:%d\n", port)
		fmt.Fprintln(output, "Connect with MCP Inspector or compatible client")
	}

	// Start the server (blocking)
	if err := server.Start(); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}
