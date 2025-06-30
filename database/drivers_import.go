package database

// Import all drivers to register them
import (
	_ "github.com/rediwo/redi-orm/internal/drivers/mysql"
	_ "github.com/rediwo/redi-orm/internal/drivers/postgresql"
	_ "github.com/rediwo/redi-orm/internal/drivers/sqlite"
)