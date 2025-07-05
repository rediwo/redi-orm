package sqlite

import (
	"os"
	"sync"

	"github.com/rediwo/redi-orm/test"
)

var (
	tempFileOnce sync.Once
	tempFileURI  string
)

func init() {
	// Create a temporary file URI once and reuse it
	tempFileOnce.Do(func() {
		tempFile, err := os.CreateTemp("", "sqlite-test-*.db")
		if err != nil {
			panic("failed to create temp file for SQLite: " + err.Error())
		}
		tempFile.Close()
		tempFileURI = "sqlite://" + tempFile.Name()
	})
	
	test.RegisterTestDatabaseUri("sqlite", tempFileURI)
}
