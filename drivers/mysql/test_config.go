package mysql

import (
	"fmt"

	"github.com/rediwo/redi-orm/test"
)

func init() {
	host := test.GetEnvOrDefault("MYSQL_TEST_HOST", "localhost")
	user := test.GetEnvOrDefault("MYSQL_TEST_USER", "testuser")
	password := test.GetEnvOrDefault("MYSQL_TEST_PASSWORD", "testpass")
	database := test.GetEnvOrDefault("MYSQL_TEST_DATABASE", "testdb")
	
	uri := fmt.Sprintf("mysql://%s:%s@%s:3306/%s?parseTime=true",
		user, password, host, database)
	
	test.RegisterTestDatabaseUri("mysql", uri)
}
