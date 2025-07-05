package postgresql

import (
	"fmt"

	"github.com/rediwo/redi-orm/test"
)

func init() {
	host := test.GetEnvOrDefault("POSTGRES_TEST_HOST", "localhost")
	user := test.GetEnvOrDefault("POSTGRES_TEST_USER", "testuser")
	password := test.GetEnvOrDefault("POSTGRES_TEST_PASSWORD", "testpass")
	database := test.GetEnvOrDefault("POSTGRES_TEST_DATABASE", "testdb")

	uri := fmt.Sprintf("postgresql://%s:%s@%s:5432/%s?sslmode=disable",
		user, password, host, database)

	test.RegisterTestDatabaseUri("postgresql", uri)
}
