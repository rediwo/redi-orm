package main

import (
	"context"
	"log"
	"os"

	"github.com/rediwo/redi-orm/database"
	_ "github.com/rediwo/redi-orm/drivers/sqlite"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

func main() {
	// Create a logger that outputs to file
	file, err := os.Create("database.log")
	if err != nil {
		log.Fatal("Failed to create log file:", err)
	}
	defer file.Close()

	fileLogger := utils.NewDefaultLogger("RediORM")
	fileLogger.SetLevel(utils.LogLevelDebug)
	fileLogger.SetOutput(file)

	// Create database with logging enabled
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		log.Fatal(err)
	}

	// Set the file logger on the database
	db.SetLogger(fileLogger)

	ctx := context.Background()

	// Connect to database
	if err := db.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Register a schema
	userSchema := schema.New("User").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true, AutoIncrement: true}).
		AddField(schema.Field{Name: "name", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "email", Type: schema.FieldTypeString, Unique: true}).
		AddField(schema.Field{Name: "age", Type: schema.FieldTypeInt})

	if err := db.RegisterSchema("User", userSchema); err != nil {
		log.Fatal(err)
	}

	// Sync schemas - this will log CREATE TABLE SQL
	if err := db.SyncSchemas(ctx); err != nil {
		log.Fatal(err)
	}

	// Insert a user - this will log INSERT SQL
	user := map[string]any{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   30,
	}

	result, err := db.Model("User").Insert(user).Exec(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Inserted user with ID: %d\n", result.LastInsertID)

	// Query users - this will log SELECT SQL
	var users []map[string]any
	selectQuery := db.Model("User").Select("id", "name", "email")
	selectQuery = selectQuery.WhereCondition(selectQuery.Where("age").GreaterThan(25))
	selectQuery = selectQuery.OrderBy("name", types.ASC)
	err = selectQuery.FindMany(ctx, &users)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Found %d users\n", len(users))

	// Update user - this will log UPDATE SQL
	updateQuery := db.Model("User").Update(map[string]any{"age": 31})
	updateQuery = updateQuery.WhereCondition(updateQuery.Where("email").Equals("john@example.com"))
	updateResult, err := updateQuery.Exec(ctx)

	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Updated %d rows\n", updateResult.RowsAffected)

	// Raw query - this will also be logged
	var count int64
	err = db.Raw("SELECT COUNT(*) FROM users WHERE age > ?", 25).FindOne(ctx, &count)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Users older than 25: %d\n", count)

	// Transaction with logging
	err = db.Transaction(ctx, func(tx types.Transaction) error {
		// All operations inside transaction will be logged
		_, err := tx.Model("User").Insert(map[string]any{
			"name":  "Jane Doe",
			"email": "jane@example.com",
			"age":   28,
		}).Exec(ctx)
		return err
	})

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Example completed successfully!")
}
