package sqlite

import (
	"github.com/rediwo/redi-orm/test"
	"github.com/rediwo/redi-orm/types"
)

func init() {
	test.RegisterTestConfig("sqlite", sqliteTestConfigFactory)
}

func sqliteTestConfigFactory() types.Config {
	return types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	}
}