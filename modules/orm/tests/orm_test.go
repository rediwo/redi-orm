package tests

import (
	"context"
	"os"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"testing"

	_ "github.com/rediwo/redi-orm/drivers/mysql"      // Import MySQL driver
	_ "github.com/rediwo/redi-orm/drivers/postgresql" // Import PostgreSQL driver
	_ "github.com/rediwo/redi-orm/drivers/sqlite"     // Import SQLite driver
	_ "github.com/rediwo/redi-orm/modules/orm"        // Import to register the module
	"github.com/rediwo/redi/runtime"
)

// TestSuite runs all JavaScript test files
func TestSuite(t *testing.T) {
	// Get the absolute path to the test directory (where this test file is located)
	_, filename, _, ok := goruntime.Caller(0)
	if !ok {
		t.Fatalf("Failed to get test file location")
	}
	testDir := filepath.Dir(filename)

	// Find all *_test.js files
	testFiles, err := filepath.Glob(filepath.Join(testDir, "*_test.js"))
	if err != nil {
		t.Fatalf("Failed to find test files: %v", err)
	}

	// Extract just the filenames for cleaner test names
	var testFileNames []string
	for _, fullPath := range testFiles {
		testFileNames = append(testFileNames, filepath.Base(fullPath))
	}

	// Sort for consistent order
	sort.Strings(testFileNames)

	// Create executor
	executor := runtime.NewExecutor()

	for _, testFile := range testFileNames {
		t.Run(testFile, func(t *testing.T) {
			scriptPath := filepath.Join(testDir, testFile)

			// Check if test file exists
			if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
				t.Skipf("Test file %s not found", testFile)
				return
			}

			// Run the test script
			config := &runtime.Config{
				ScriptPath: scriptPath,
				BasePath:   testDir,
				Version:    "dev",
			}

			exitCode, err := executor.Execute(config)
			if err != nil {
				t.Fatalf("Script execution failed: %v", err)
			}

			if exitCode != 0 {
				t.Fatalf("Script exited with code %d", exitCode)
			}
		})
	}
}

// TestFromUri specifically tests the fromUri functionality
func TestFromUri(t *testing.T) {
	ctx := context.Background()

	// Create a temporary directory for test databases
	tempDir, err := os.MkdirTemp("", "orm-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Get test directory location
	_, filename, _, ok := goruntime.Caller(0)
	if !ok {
		t.Fatalf("Failed to get test file location")
	}
	testFileDir := filepath.Dir(filename)

	// Save current directory and change to temp
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Copy all test files and assert.js to temp directory
	patterns := []string{"*_test.js", "assert.js"}
	for _, pattern := range patterns {
		files, err := filepath.Glob(filepath.Join(testFileDir, pattern))
		if err != nil {
			t.Logf("Warning: Failed to glob %s: %v", pattern, err)
			continue
		}

		for _, srcPath := range files {
			filename := filepath.Base(srcPath)
			dstPath := filepath.Join(tempDir, filename)

			data, err := os.ReadFile(srcPath)
			if err != nil {
				t.Logf("Warning: Failed to read %s: %v", filename, err)
				continue
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				t.Logf("Warning: Failed to write %s: %v", filename, err)
			}
		}
	}

	// Run fromUri specific tests
	t.Run("fromUri", func(t *testing.T) {
		executor := runtime.NewExecutor()

		scriptPath := filepath.Join(tempDir, "schema_test.js")
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			// If test file doesn't exist, create a simple inline test
			testScript := `
const { fromUri } = require('redi/orm');

async function test() {
    const db = fromUri('sqlite://./test.db');
    await db.connect();
    await db.ping();
    await db.close();
    console.log('✓ fromUri test passed');
}

test().catch(err => {
    console.error('✗ fromUri test failed:', err);
    process.exit(1);
});
`
			if err := os.WriteFile(scriptPath, []byte(testScript), 0644); err != nil {
				t.Fatalf("Failed to create test script: %v", err)
			}
		}

		config := &runtime.Config{
			ScriptPath: scriptPath,
			BasePath:   tempDir,
			Version:    "dev",
		}

		exitCode, err := executor.Execute(config)
		if err != nil {
			t.Fatalf("Script execution failed: %v", err)
		}

		if exitCode != 0 {
			t.Fatalf("Script exited with code %d", exitCode)
		}
	})

	_ = ctx // Use context if needed
}
