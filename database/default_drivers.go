package database

// Import all default drivers to register them automatically
// This makes it convenient for users to use database.New() without manually importing drivers
import (
	_ "github.com/rediwo/redi-orm/drivers/mysql"
	_ "github.com/rediwo/redi-orm/drivers/sqlite"
)
