package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rediwo/redi-orm/database"
	_ "github.com/rediwo/redi-orm/drivers/mongodb"    // Import MongoDB driver
	_ "github.com/rediwo/redi-orm/drivers/mysql"      // Import MySQL driver
	_ "github.com/rediwo/redi-orm/drivers/postgresql" // Import PostgreSQL driver
	_ "github.com/rediwo/redi-orm/drivers/sqlite"     // Import SQLite driver
	"github.com/rediwo/redi-orm/graphql"
	"github.com/rediwo/redi-orm/mcp"
	"github.com/rediwo/redi-orm/migration"
	_ "github.com/rediwo/redi-orm/modules/orm" // Import ORM module
	"github.com/rediwo/redi-orm/prisma"
	"github.com/rediwo/redi-orm/rest"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi/runtime"
)

// Version is injected at build time via -ldflags
var version = "dev"

const (
	usage = `RediORM CLI - Database migration tool

Usage:
  redi-orm <command> [flags]

Commands:
  run               Execute a JavaScript file with ORM support
  server            Start GraphQL and REST API server
  mcp               Start MCP (Model Context Protocol) server
  migrate           Run pending migrations
  migrate:generate  Generate new migration file
  migrate:apply     Apply pending migrations from directory
  migrate:rollback  Rollback last applied migration
  migrate:status    Show migration status
  migrate:reset     Reset all migrations (drop all tables)
  migrate:dry-run   Preview migration changes without applying them
  version           Show version information

Flags:
  --db          Database URI (required)
                Examples:
                - sqlite://./myapp.db
                - mysql://user:pass@localhost:3306/dbname
                - postgresql://user:pass@localhost:5432/dbname
  
  --schema      Path to schema file (default: ./schema.prisma)
                Supports Prisma-style schema definitions
  
  --migrations  Path to migrations directory (default: ./migrations)
                Used for file-based migrations in production
  
  --mode        Migration mode: auto|file (default: auto)
                auto - Auto-migrate (development)
                file - File-based migrations (production)
  
  --name        Migration name (for generate command)
  
  --force       Force destructive changes without confirmation
                Use with caution! This will drop columns/tables
  
  --timeout     Execution timeout in milliseconds (for run command)
                Example: --timeout 30000 (30 seconds)
                Default: 0 (no timeout)
  
  --port        Server port (for server command)
                Default: 4000
                GraphQL endpoint: http://localhost:4000/graphql
                REST API endpoint: http://localhost:4000/api
  
  --playground  Enable GraphQL playground (for server command)
                Default: true
  
  --cors        Enable CORS (for server command)
                Default: true
  
  --log-level   Logging level for server (debug|info|warn|error|none)
                Default: info
                Controls both GraphQL/REST operation logging and database SQL logging
                - debug: Shows operations + SQL queries with execution times
                - info: Shows operations only (no SQL)
                - warn/error: Shows warnings/errors only  
                - none: Disables all logging
                Example: --log-level debug
  
  --transport   MCP transport mode (stdio|http) (for mcp command)
                Default: http
                - stdio: Standard I/O (for local AI assistants)
                - http: HTTP server with SSE (for remote access)
  
  --help        Show help message

Examples:
  # Run JavaScript file with ORM
  redi-orm run app.js
  redi-orm run scripts/migrate-data.js
  redi-orm run --timeout 60000 long-running-script.js
  
  # Start GraphQL and REST API server
  redi-orm server --db=sqlite://./myapp.db --schema=./schema.prisma
  redi-orm server --db=postgresql://user:pass@localhost/db --port=8080
  
  # Start MCP server
  redi-orm mcp --db=sqlite://./myapp.db --schema=./schema.prisma
  redi-orm mcp --db=postgresql://user:pass@localhost/db --transport=http --port=3000
  
  # Auto-migrate (development)
  redi-orm migrate --db=sqlite://./myapp.db --schema=./schema.prisma
  
  # Generate migration file (production)
  redi-orm migrate:generate --db=sqlite://./myapp.db --schema=./schema.prisma --name="add_user_table"
  
  # Apply migrations from directory (production)
  redi-orm migrate:apply --db=sqlite://./myapp.db --migrations=./migrations
  
  # Rollback last migration
  redi-orm migrate:rollback --db=sqlite://./myapp.db --migrations=./migrations
  
  # Check migration status
  redi-orm migrate:status --db=sqlite://./myapp.db
  
  # Preview changes (dry run)
  redi-orm migrate:dry-run --db=sqlite://./myapp.db --schema=./schema.prisma
  
  # Reset database (dangerous!)
  redi-orm migrate:reset --db=sqlite://./myapp.db --force
`
)

func main() {
	// Define flags
	var (
		dbURI         string
		schemaPath    string
		migrationsDir string
		mode          string
		name          string
		force         bool
		help          bool
		timeout       int
		port          int
		playground    bool
		cors          bool
		logLevel      string
		transport     string
		
		// Security flags
		apiKey        string
		enableAuth    bool
		readOnlyMode  bool
		rateLimit     int
		allowedTables string
	)

	flag.StringVar(&dbURI, "db", "", "Database URI")
	flag.StringVar(&schemaPath, "schema", "./schema.prisma", "Path to schema file")
	flag.StringVar(&migrationsDir, "migrations", "./migrations", "Path to migrations directory")
	flag.StringVar(&mode, "mode", "auto", "Migration mode: auto|file")
	flag.StringVar(&name, "name", "", "Migration name")
	flag.BoolVar(&force, "force", false, "Force destructive changes")
	flag.BoolVar(&help, "help", false, "Show help message")
	flag.IntVar(&timeout, "timeout", 0, "Execution timeout in milliseconds (for run command)")
	flag.IntVar(&port, "port", 4000, "Server port (for server command)")
	flag.BoolVar(&playground, "playground", true, "Enable GraphQL playground (for server command)")
	flag.BoolVar(&cors, "cors", true, "Enable CORS (for server command)")
	flag.StringVar(&logLevel, "log-level", "info", "Logging level for GraphQL server")
	flag.StringVar(&transport, "transport", "http", "MCP transport mode (stdio|http)")
	
	// Security flags
	flag.StringVar(&apiKey, "api-key", "", "API key for HTTP transport authentication")
	flag.BoolVar(&enableAuth, "enable-auth", false, "Enable authentication for HTTP transport")
	flag.BoolVar(&readOnlyMode, "read-only", true, "Enable read-only mode (default: true)")
	flag.IntVar(&rateLimit, "rate-limit", 60, "Requests per minute rate limit")
	flag.StringVar(&allowedTables, "allowed-tables", "", "Comma-separated list of allowed tables")

	// Custom usage
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}

	// Check if any arguments provided
	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(0)
	}

	// Get command first
	command := os.Args[1]

	// Handle version command before parsing flags
	if command == "version" {
		fmt.Printf("RediORM CLI v%s\n", version)
		os.Exit(0)
	}

	// Handle help
	if command == "help" || command == "--help" || command == "-h" {
		flag.Usage()
		os.Exit(0)
	}

	// For the run command, handle timeout flag manually to support Node.js-style syntax
	if command == "run" {
		args := os.Args[2:]
		var filteredArgs []string

		for i := 0; i < len(args); i++ {
			arg := args[i]
			if arg == "--timeout" && i+1 < len(args) {
				// --timeout 5000 style
				if val, err := strconv.Atoi(args[i+1]); err == nil {
					timeout = val
					i++ // skip next arg
					continue
				}
			} else if strings.HasPrefix(arg, "--timeout=") {
				// --timeout=5000 style
				if val, err := strconv.Atoi(strings.TrimPrefix(arg, "--timeout=")); err == nil {
					timeout = val
					continue
				}
			}
			filteredArgs = append(filteredArgs, arg)
		}

		// Now parse remaining flags
		flag.CommandLine.Parse(filteredArgs)
	} else {
		// For other commands, use normal flag parsing
		flag.CommandLine.Parse(os.Args[2:])
	}

	// Execute command
	ctx := context.Background()
	switch command {
	case "run":
		// For run command, we need the script file as the next argument
		if len(flag.Args()) < 1 {
			log.Fatal("Error: JavaScript file path required\nUsage: redi-orm run <script.js>")
		}
		scriptPath := flag.Args()[0]
		// Pass remaining args after the script path as script arguments
		scriptArgs := flag.Args()[1:]
		runScript(scriptPath, scriptArgs, timeout)
		return
	case "server":
		// Validate required flags
		if dbURI == "" {
			log.Fatal("Error: --db flag is required")
		}
		runServer(ctx, dbURI, schemaPath, port, playground, cors, logLevel)
		return
	case "mcp":
		// Validate required flags
		if dbURI == "" {
			log.Fatal("Error: --db flag is required")
		}
		runMCP(ctx, dbURI, schemaPath, port, transport, logLevel, apiKey, enableAuth, readOnlyMode, rateLimit, allowedTables)
		return
	}

	// Validate required flags for other commands
	if dbURI == "" {
		log.Fatal("Error: --db flag is required")
	}

	switch command {
	case "migrate":
		runMigrate(ctx, dbURI, schemaPath, migrationsDir, mode, false, force)
	case "migrate:generate":
		runMigrateGenerate(ctx, dbURI, schemaPath, migrationsDir, name)
	case "migrate:apply":
		runMigrateApply(ctx, dbURI, migrationsDir)
	case "migrate:rollback":
		runMigrateRollback(ctx, dbURI, migrationsDir)
	case "migrate:dry-run":
		runMigrate(ctx, dbURI, schemaPath, migrationsDir, mode, true, force)
	case "migrate:status":
		runMigrateStatus(ctx, dbURI)
	case "migrate:reset":
		runMigrateReset(ctx, dbURI, force)
	default:
		log.Fatalf("Unknown command: %s\n\nRun 'redi-orm --help' for usage", command)
	}
}

func runMigrate(ctx context.Context, dbURI, schemaPath, migrationsDir, mode string, dryRun, force bool) {
	// Create database connection
	db, err := database.NewFromURI(dbURI)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}

	// Connect to database
	if err := db.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Load schema from file
	schemas, err := loadSchemaFromFile(schemaPath)
	if err != nil {
		log.Fatalf("Failed to load schema: %v", err)
	}

	// Register schemas with database
	for _, schema := range schemas {
		if err := db.RegisterSchema(schema.Name, schema); err != nil {
			log.Fatalf("Failed to register schema %s: %v", schema.Name, err)
		}
	}

	// Create migration manager
	options := types.MigrationOptions{
		DryRun:        dryRun,
		Force:         force,
		Mode:          types.MigrationMode(mode),
		MigrationsDir: migrationsDir,
	}
	manager, err := migration.NewManager(db, options)
	if err != nil {
		log.Fatalf("Failed to create migration manager: %v", err)
	}

	// Run migrations
	if err := manager.Migrate(schemas); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	if dryRun {
		fmt.Println("\nDry run completed. No changes were applied.")
	} else {
		fmt.Println("\nMigration completed successfully.")
	}
}

func runMigrateStatus(ctx context.Context, dbURI string) {
	// Create database connection
	db, err := database.NewFromURI(dbURI)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}

	// Connect to database
	if err := db.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create migration manager
	options := types.MigrationOptions{}
	manager, err := migration.NewManager(db, options)
	if err != nil {
		log.Fatalf("Failed to create migration manager: %v", err)
	}

	// Get migration status
	status, err := manager.GetMigrationStatus()
	if err != nil {
		log.Fatalf("Failed to get migration status: %v", err)
	}

	// Display status
	fmt.Println("=== Migration Status ===")
	fmt.Printf("Database: %s\n", dbURI)
	fmt.Printf("Tables: %d\n", status.TableCount)

	if len(status.Tables) > 0 {
		fmt.Println("\nExisting tables:")
		for _, table := range status.Tables {
			fmt.Printf("  - %s\n", table)
		}
	}

	if status.LastMigration != nil {
		fmt.Printf("\nLast migration:\n")
		fmt.Printf("  Version: %s\n", status.LastMigration.Version)
		fmt.Printf("  Name: %s\n", status.LastMigration.Name)
		fmt.Printf("  Applied: %s\n", status.LastMigration.AppliedAt.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Println("\nNo migrations have been applied yet.")
	}

	if len(status.AppliedMigrations) > 0 {
		fmt.Printf("\nTotal migrations applied: %d\n", len(status.AppliedMigrations))
	}
}

func runMigrateReset(ctx context.Context, dbURI string, force bool) {
	if !force {
		fmt.Println("WARNING: This will drop all tables and clear migration history!")
		fmt.Println("Use --force flag to confirm this destructive operation.")
		os.Exit(1)
	}

	// Create database connection
	db, err := database.NewFromURI(dbURI)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}

	// Connect to database
	if err := db.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create migration manager
	options := types.MigrationOptions{
		Force: force,
	}
	manager, err := migration.NewManager(db, options)
	if err != nil {
		log.Fatalf("Failed to create migration manager: %v", err)
	}

	// Reset migrations
	if err := manager.ResetMigrations(); err != nil {
		log.Fatalf("Failed to reset migrations: %v", err)
	}

	fmt.Println("Migration reset completed successfully.")
}

func runMigrateGenerate(ctx context.Context, dbURI, schemaPath, migrationsDir, name string) {
	if name == "" {
		log.Fatal("Error: --name flag is required for generate command")
	}

	// Create database connection
	db, err := database.NewFromURI(dbURI)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}

	// Connect to database
	if err := db.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Load schema from file
	schemas, err := loadSchemaFromFile(schemaPath)
	if err != nil {
		log.Fatalf("Failed to load schema: %v", err)
	}

	// Create migration manager
	options := types.MigrationOptions{
		Mode:          types.MigrationModeFile,
		MigrationsDir: migrationsDir,
	}
	manager, err := migration.NewManager(db, options)
	if err != nil {
		log.Fatalf("Failed to create migration manager: %v", err)
	}

	// Generate migration
	if err := manager.GenerateMigration(name, schemas); err != nil {
		log.Fatalf("Failed to generate migration: %v", err)
	}
}

func runMigrateApply(ctx context.Context, dbURI, migrationsDir string) {
	// Create database connection
	db, err := database.NewFromURI(dbURI)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}

	// Connect to database
	if err := db.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create migration manager
	options := types.MigrationOptions{
		Mode:          types.MigrationModeFile,
		MigrationsDir: migrationsDir,
	}
	manager, err := migration.NewManager(db, options)
	if err != nil {
		log.Fatalf("Failed to create migration manager: %v", err)
	}

	// Run migrations
	if err := manager.Migrate(nil); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
}

func runMigrateRollback(ctx context.Context, dbURI, migrationsDir string) {
	// Create database connection
	db, err := database.NewFromURI(dbURI)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}

	// Connect to database
	if err := db.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create migration manager
	options := types.MigrationOptions{
		Mode:          types.MigrationModeFile,
		MigrationsDir: migrationsDir,
	}
	manager, err := migration.NewManager(db, options)
	if err != nil {
		log.Fatalf("Failed to create migration manager: %v", err)
	}

	// Rollback migration
	if err := manager.RollbackMigration(); err != nil {
		log.Fatalf("Rollback failed: %v", err)
	}
}

func runScript(scriptPath string, args []string, timeoutMs int) {
	// Check if script file exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		log.Fatalf("Script file not found: %s", scriptPath)
	}

	// Get absolute path
	absPath, err := filepath.Abs(scriptPath)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}

	// Create executor
	executor := runtime.NewExecutor()

	// Create runtime config
	config := &runtime.Config{
		ScriptPath: absPath,
		BasePath:   filepath.Dir(absPath),
		Version:    version,
		Args:       args,
		Timeout:    time.Duration(timeoutMs) * time.Millisecond,
	}

	// Execute the script
	exitCode, err := executor.Execute(config)
	if err != nil {
		log.Fatalf("Script execution failed: %v", err)
	}

	// Exit with the same code as the script
	os.Exit(exitCode)
}

func loadSchemaFromFile(path string) (map[string]*schema.Schema, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("schema file not found: %s", path)
	}

	// Read file contents
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	// Parse schema
	schemas, err := prisma.ParseSchema(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	// Log loaded models
	fmt.Printf("Loaded %d models from %s:\n", len(schemas), filepath.Base(path))
	for name := range schemas {
		fmt.Printf("  - %s\n", name)
	}
	fmt.Println()

	return schemas, nil
}

func runServer(ctx context.Context, dbURI, schemaPath string, port int, playground, cors bool, logLevel string) {
	// Create database connection
	db, err := database.NewFromURI(dbURI)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}

	// Connect to database
	if err := db.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Load schema from file if provided
	if schemaPath != "" {
		if _, err := os.Stat(schemaPath); err == nil {
			schemas, err := loadSchemaFromFile(schemaPath)
			if err != nil {
				log.Fatalf("Failed to load schema: %v", err)
			}

			// Register schemas with database
			for _, schema := range schemas {
				if err := db.RegisterSchema(schema.Name, schema); err != nil {
					log.Fatalf("Failed to register schema %s: %v", schema.Name, err)
				}
			}

			// Sync schemas
			if err := db.SyncSchemas(ctx); err != nil {
				log.Fatalf("Failed to sync schemas: %v", err)
			}
		}
	}

	// Create GraphQL server configuration
	graphqlConfig := graphql.ServerConfig{
		DatabaseURI: dbURI,
		SchemaPath:  schemaPath,
		Port:        port,
		Playground:  playground,
		CORS:        cors,
		LogLevel:    logLevel,
	}

	// Create GraphQL server
	graphqlServer, err := graphql.NewServer(graphqlConfig)
	if err != nil {
		log.Fatalf("Failed to create GraphQL server: %v", err)
	}

	// Create REST server configuration
	restConfig := rest.ServerConfig{
		Database:   db,
		Port:       port,
		LogLevel:   logLevel,
		SchemaFile: schemaPath,
	}

	// Create REST server
	restServer, err := rest.NewServer(restConfig)
	if err != nil {
		log.Fatalf("Failed to create REST server: %v", err)
	}

	// Create a multiplexer to handle both GraphQL and REST
	mux := http.NewServeMux()

	// Mount GraphQL at /graphql
	mux.Handle("/graphql", graphqlServer.Handler())
	if playground {
		mux.Handle("/", graphqlServer.Handler()) // Playground at root
	}

	// Mount REST API at /api
	mux.Handle("/api/", restServer.Router)

	// Apply CORS if enabled
	var handler http.Handler = mux
	if cors {
		handler = applyCORS(handler)
	}

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler,
	}

	// Start server
	fmt.Printf("Starting server on http://localhost:%d\n", port)
	fmt.Printf("  GraphQL endpoint: http://localhost:%d/graphql\n", port)
	if playground {
		fmt.Printf("  GraphQL Playground: http://localhost:%d/\n", port)
	}
	fmt.Printf("  REST API endpoint: http://localhost:%d/api\n", port)
	fmt.Println()

	// Start the server (blocking)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

func applyCORS(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Connection-Name")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

func runMCP(ctx context.Context, dbURI, schemaPath string, port int, transport, logLevel, apiKey string, enableAuth, readOnlyMode bool, rateLimit int, allowedTables string) {
	// Parse allowed tables
	var allowedTablesList []string
	if allowedTables != "" {
		allowedTablesList = strings.Split(allowedTables, ",")
		for i, table := range allowedTablesList {
			allowedTablesList[i] = strings.TrimSpace(table)
		}
	}

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
			APIKey:         apiKey,
			EnableRateLimit: rateLimit > 0,
			RequestsPerMin:  rateLimit,
			BurstLimit:     rateLimit / 4, // 25% of rate limit as burst
			AllowedTables:  allowedTablesList,
			ReadOnlyMode:   readOnlyMode,
			MaxQueryRows:   1000,
		},
	}

	// Create MCP server
	server, err := mcp.NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	// Start server
	fmt.Printf("Starting MCP server\n")
	fmt.Printf("  Database: %s\n", dbURI)
	fmt.Printf("  Transport: %s\n", transport)
	if transport == "http" {
		fmt.Printf("  Port: %d\n", port)
		fmt.Printf("  Authentication: %t\n", enableAuth)
		if enableAuth {
			fmt.Printf("  API Key: %s\n", "***")
		}
		if rateLimit > 0 {
			fmt.Printf("  Rate Limit: %d req/min\n", rateLimit)
		}
	}
	fmt.Printf("  Read-only Mode: %t\n", readOnlyMode)
	fmt.Printf("  Log level: %s\n", logLevel)
	if len(allowedTablesList) > 0 {
		fmt.Printf("  Allowed Tables: %s\n", allowedTables)
	}
	fmt.Println()

	if transport == "stdio" {
		fmt.Println("MCP server is ready for JSON-RPC communication via stdio")
		fmt.Println("Connect your AI assistant to this process")
	} else {
		fmt.Printf("MCP server is ready at http://localhost:%d\n", port)
		fmt.Println("  - POST / for JSON-RPC requests")
		fmt.Println("  - GET /events for Server-Sent Events")
	}

	// Start the server (blocking)
	if err := server.Start(); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}
