package mysql

import (
	"github.com/rediwo/redi-orm/test"
	"github.com/rediwo/redi-orm/types"
)

func init() {
	test.RegisterTestConfig("mysql", mysqlTestConfigFactory)
}

func mysqlTestConfigFactory() types.Config {
	return types.Config{
		Type:     "mysql",
		Host:     test.GetEnvOrDefault("MYSQL_TEST_HOST", "localhost"),
		Port:     3306,
		User:     test.GetEnvOrDefault("MYSQL_TEST_USER", "testuser"),
		Password: test.GetEnvOrDefault("MYSQL_TEST_PASSWORD", "testpass"),
		Database: test.GetEnvOrDefault("MYSQL_TEST_DATABASE", "testdb"),
		Options: map[string]string{
			"parseTime": "true",
		},
	}
}