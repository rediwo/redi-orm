package orm

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rediwo/redi/runtime"
)

// JSTestRunner provides utilities for running JavaScript tests
type JSTestRunner struct {
	executor    *runtime.Executor
	t           *testing.T
	basePath    string
	databaseURI string
	tempDir     string
}

// NewJSTestRunner creates a new JavaScript test runner
func NewJSTestRunner(t *testing.T) (*JSTestRunner, error) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "orm-js-test-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	return &JSTestRunner{
		executor: runtime.NewExecutor(),
		t:        t,
		basePath: tempDir,
		tempDir:  tempDir,
	}, nil
}

// Cleanup removes temporary test files
func (r *JSTestRunner) Cleanup() {
	if r.tempDir != "" {
		os.RemoveAll(r.tempDir)
	}
}

// SetDatabaseURI sets the database URI for tests
func (r *JSTestRunner) SetDatabaseURI(uri string) {
	r.databaseURI = uri
}

// GetDatabaseURI returns the database URI for tests
func (r *JSTestRunner) GetDatabaseURI() string {
	return r.databaseURI
}

// RunScript executes a JavaScript script with the given content
func (r *JSTestRunner) RunScript(scriptContent string, args ...string) error {
	// Create temporary script file
	scriptPath := filepath.Join(r.tempDir, "test_script.js")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		return fmt.Errorf("failed to write script: %w", err)
	}

	// Create runtime config
	config := &runtime.Config{
		ScriptPath: scriptPath,
		BasePath:   r.basePath,
		Version:    "test",
		Args:       args,
		Timeout:    30 * time.Second, // Default 30 second timeout for tests
	}

	// Set database URI as environment variable
	if r.databaseURI != "" {
		os.Setenv("TEST_DATABASE_URI", r.databaseURI)
		defer os.Unsetenv("TEST_DATABASE_URI")
	}

	// Execute the script
	exitCode, err := r.executor.Execute(config)
	if err != nil {
		return fmt.Errorf("script execution failed: %w", err)
	}

	if exitCode != 0 {
		return fmt.Errorf("script exited with code %d", exitCode)
	}

	return nil
}

// RunTestFile executes a JavaScript test file from testdata directory
func (r *JSTestRunner) RunTestFile(t *testing.T, filename string) error {
	// Copy test file to temp directory
	testdataPath := filepath.Join("testdata", filename)
	content, err := os.ReadFile(testdataPath)
	if err != nil {
		// If file doesn't exist in testdata, try embedded testdata
		embeddedPath := filepath.Join(filepath.Dir(r.t.Name()), "testdata", filename)
		content, err = os.ReadFile(embeddedPath)
		if err != nil {
			return fmt.Errorf("failed to read test file %s: %w", filename, err)
		}
	}

	// Also copy assert.js if it exists
	assertPath := filepath.Join("testdata", "assert.js")
	assertContent, err := os.ReadFile(assertPath)
	if err == nil {
		assertDest := filepath.Join(r.tempDir, "assert.js")
		if err := os.WriteFile(assertDest, assertContent, 0644); err != nil {
			return fmt.Errorf("failed to write assert.js: %w", err)
		}
	}

	// Write test file to temp directory
	testPath := filepath.Join(r.tempDir, filename)
	if err := os.WriteFile(testPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write test file: %w", err)
	}

	// Create runtime config
	config := &runtime.Config{
		ScriptPath: testPath,
		BasePath:   r.tempDir,
		Version:    "test",
		Timeout:    30 * time.Second,
	}

	// Set database URI as environment variable
	if r.databaseURI != "" {
		os.Setenv("TEST_DATABASE_URI", r.databaseURI)
		defer os.Unsetenv("TEST_DATABASE_URI")
	}

	// Execute the test file
	exitCode, err := r.executor.Execute(config)
	if err != nil {
		return fmt.Errorf("test execution failed: %w", err)
	}

	if exitCode != 0 {
		return fmt.Errorf("test exited with code %d", exitCode)
	}

	return nil
}

// CopyTestAssets copies test assets (like assert.js, test schemas) to the temp directory
func (r *JSTestRunner) CopyTestAssets() error {
	// List of asset files to copy
	assets := []string{
		"assert.js",
		"test_schemas.js",
		"test_helpers.js",
	}

	// Try to find testdata directory relative to the module
	moduleDir := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "rediwo", "redi-orm", "modules", "orm")
	if os.Getenv("GOPATH") == "" {
		// Try relative to current working directory
		cwd, _ := os.Getwd()
		possiblePaths := []string{
			filepath.Join(cwd, "modules", "orm", "testdata"),
			filepath.Join(cwd, "..", "..", "modules", "orm", "testdata"),
			filepath.Join(cwd, "..", "..", "..", "modules", "orm", "testdata"),
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(filepath.Join(path, "assert.js")); err == nil {
				moduleDir = filepath.Dir(path)
				break
			}
		}
	}

	testdataDir := filepath.Join(moduleDir, "testdata")

	for _, asset := range assets {
		srcPath := filepath.Join(testdataDir, asset)
		content, err := os.ReadFile(srcPath)
		if os.IsNotExist(err) {
			// Skip if file doesn't exist
			continue
		}
		if err != nil {
			// Also skip on error (might not have found the right path)
			continue
		}

		dstPath := filepath.Join(r.tempDir, asset)
		if err := os.WriteFile(dstPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write asset %s: %w", asset, err)
		}
	}

	return nil
}

// RunCleanupScript runs a JavaScript cleanup script
func (r *JSTestRunner) RunCleanupScript(cleanupCode string) error {
	cleanupScript := fmt.Sprintf(`
const ormModule = require('redi/orm');

// Track all database connections created in cleanup
const dbConnections = [];

// Create wrapped fromUri function
const fromUri = function(uri) {
    const db = ormModule.fromUri(uri);
    dbConnections.push(db);
    return db;
};

async function cleanup() {
	try {
		%s
		
		// Close all database connections
		for (const db of dbConnections) {
			try {
				await db.close();
			} catch (err) {
				console.error('Warning: Failed to close database connection:', err.message);
			}
		}
	} catch (err) {
		console.error('Cleanup error:', err.message);
		
		// Try to close connections even on failure
		for (const db of dbConnections) {
			try {
				await db.close();
			} catch (closeErr) {
				// Ignore close errors
			}
		}
		
		process.exit(1);
	}
	process.exit(0);
}

cleanup();
`, cleanupCode)

	return r.RunScript(cleanupScript)
}

// RunInlineTest runs an inline JavaScript test
func (r *JSTestRunner) RunInlineTest(t *testing.T, testName string, testCode string) {
	t.Run(testName, func(t *testing.T) {
		// Wrap test code with proper setup
		fullScript := fmt.Sprintf(`
const ormModule = require('redi/orm');

// Inline assert module since we can't guarantee file location
const baseAssert = require('assert');

// Create custom assert with additional methods
const assert = function(condition, message) {
    if (!condition) {
        throw new Error(message || 'Assertion failed');
    }
};

// Copy over standard assert methods
Object.keys(baseAssert).forEach(key => {
    if (typeof baseAssert[key] === 'function') {
        assert[key] = baseAssert[key];
    }
});

// Add custom methods
assert.match = function(actual, regex, message) {
    if (!regex.test(actual)) {
        throw new Error(message || 'Expected "' + actual + '" to match ' + regex);
    }
};
assert.includes = function(array, value, message) {
    if (!array.includes(value)) {
        throw new Error(message || 'Expected array to include ' + JSON.stringify(value));
    }
};
assert.lengthOf = function(array, length, message) {
    if (array.length !== length) {
        throw new Error(message || 'Expected array length ' + length + ', got ' + array.length);
    }
};
assert.strictEqual = function(actual, expected, message) {
    if (actual !== expected) {
        throw new Error(message || 'Expected ' + JSON.stringify(actual) + ' to strictly equal ' + JSON.stringify(expected));
    }
};
assert.deepEqual = function(actual, expected, message) {
    if (JSON.stringify(actual) !== JSON.stringify(expected)) {
        throw new Error(message || 'Expected ' + JSON.stringify(actual) + ' to deep equal ' + JSON.stringify(expected));
    }
};

// Get database URI from environment
const TEST_DATABASE_URI = process.env.TEST_DATABASE_URI;
if (!TEST_DATABASE_URI) {
    console.error('TEST_DATABASE_URI not set');
    process.exit(1);
}

// Track all database connections created in the test
const dbConnections = [];

// Create wrapped fromUri function
const fromUri = function(uri) {
    const db = ormModule.fromUri(uri);
    dbConnections.push(db);
    return db;
};

async function runTest() {
    try {
        %s
        console.log('✓ %s passed');
        
        // Close all database connections
        for (const db of dbConnections) {
            try {
                await db.close();
            } catch (err) {
                console.error('Warning: Failed to close database connection:', err.message);
            }
        }
        
        process.exit(0);
    } catch (err) {
        console.error('✗ %s failed:', err.message);
        console.error(err.stack);
        
        // Try to close connections even on failure
        for (const db of dbConnections) {
            try {
                await db.close();
            } catch (closeErr) {
                // Ignore close errors on test failure
            }
        }
        
        process.exit(1);
    }
}

runTest().catch(err => {
    console.error('Unhandled error:', err);
    process.exit(1);
});
`, testCode, testName, testName)

		if err := r.RunScript(fullScript); err != nil {
			t.Fatalf("Test failed: %v", err)
		}
	})
}
