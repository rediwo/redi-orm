package postgresql

import (
	"github.com/rediwo/redi-orm/test"
	"github.com/rediwo/redi-orm/types"
)

func init() {
	test.RegisterTestConfig("postgresql", postgresqlTestConfigFactory)
}

func postgresqlTestConfigFactory() types.Config {
	return types.Config{
		Type:     "postgresql",
		Host:     test.GetEnvOrDefault("POSTGRES_TEST_HOST", "localhost"),
		Port:     5432,
		User:     test.GetEnvOrDefault("POSTGRES_TEST_USER", "testuser"),
		Password: test.GetEnvOrDefault("POSTGRES_TEST_PASSWORD", "testpass"),
		Database: test.GetEnvOrDefault("POSTGRES_TEST_DATABASE", "testdb"),
		Options: map[string]string{
			"sslmode": "disable",
		},
	}
}