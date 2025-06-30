package test

import (
	"fmt"
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/engine"
	"github.com/rediwo/redi-orm/schema"
)

func BenchmarkBasicOperations(b *testing.B) {
	// Setup
	tdb := SetupTestDB(&testing.T{})
	defer tdb.Cleanup()

	_ = tdb.CreateUserSchema()

	b.Run("Insert", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			script := fmt.Sprintf(`models.User.add({name: "User%d", email: "user%d@example.com", age: %d})`, i, i, 20+i%50)
			_, err := tdb.ExecuteJS(script)
			if err != nil {
				b.Fatalf("Failed to insert: %v", err)
			}
		}
	})

	// Add some data for other benchmarks
	for i := 0; i < 1000; i++ {
		script := fmt.Sprintf(`models.User.add({name: "BenchUser%d", email: "bench%d@example.com", age: %d})`, i, i, 20+i%50)
		tdb.ExecuteJS(script)
	}

	b.Run("SelectAll", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := tdb.ExecuteJS(`models.User.select().execute()`)
			if err != nil {
				b.Fatalf("Failed to select: %v", err)
			}
		}
	})

	b.Run("SelectWithWhere", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := tdb.ExecuteJS(`models.User.select().where("age", ">", 30).execute()`)
			if err != nil {
				b.Fatalf("Failed to select with where: %v", err)
			}
		}
	})

	b.Run("GetByID", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			id := (i % 1000) + 1
			script := fmt.Sprintf(`models.User.get(%d)`, id)
			_, err := tdb.ExecuteJS(script)
			if err != nil {
				b.Fatalf("Failed to get by ID: %v", err)
			}
		}
	})

	b.Run("Update", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			id := (i % 1000) + 1
			script := fmt.Sprintf(`models.User.set(%d, {age: %d})`, id, 25+i%40)
			_, err := tdb.ExecuteJS(script)
			if err != nil {
				b.Fatalf("Failed to update: %v", err)
			}
		}
	})

	b.Run("Count", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := tdb.ExecuteJS(`models.User.select().count()`)
			if err != nil {
				b.Fatalf("Failed to count: %v", err)
			}
		}
	})
}

func BenchmarkSchemaRegistration(b *testing.B) {
	// Create database
	db, err := database.New(database.Config{
		Type:     database.SQLite,
		FilePath: ":memory:",
	})
	if err != nil {
		b.Fatalf("Failed to create database: %v", err)
	}

	if err := db.Connect(); err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		eng := engine.New(db)

		userSchema := schema.New(fmt.Sprintf("User%d", i)).
			WithTableName(fmt.Sprintf("users_%d", i)).
			AddField(schema.NewField("id").Int().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("name").String().Build()).
			AddField(schema.NewField("email").String().Build())

		err := eng.RegisterSchema(userSchema)
		if err != nil {
			b.Fatalf("Failed to register schema: %v", err)
		}
	}
}

func BenchmarkJavaScriptExecution(b *testing.B) {
	tdb := SetupTestDB(&testing.T{})
	defer tdb.Cleanup()

	tdb.CreateUserSchema()

	// Add some test data
	for i := 0; i < 100; i++ {
		script := fmt.Sprintf(`models.User.add({name: "User%d", email: "user%d@example.com", age: %d})`, i, i, 20+i%50)
		tdb.ExecuteJS(script)
	}

	scripts := []struct {
		name   string
		script string
	}{
		{"SimpleGet", `models.User.get(1)`},
		{"SimpleAdd", `models.User.add({name: "Test", email: "test@example.com", age: 25})`},
		{"SimpleSelect", `models.User.select().execute()`},
		{"ComplexSelect", `models.User.select().where("age", ">", 25).orderBy("name", "ASC").limit(10).execute()`},
		{"Count", `models.User.select().count()`},
		{"Update", `models.User.set(1, {age: 30})`},
	}

	for _, script := range scripts {
		b.Run(script.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := tdb.ExecuteJS(script.script)
				if err != nil {
					b.Fatalf("Failed to execute script: %v", err)
				}
			}
		})
	}
}

func BenchmarkQueryBuilder(b *testing.B) {
	tdb := SetupTestDB(&testing.T{})
	defer tdb.Cleanup()

	tdb.CreateUserSchema()

	// Add test data
	for i := 0; i < 1000; i++ {
		data := map[string]interface{}{
			"name":  fmt.Sprintf("User%d", i),
			"email": fmt.Sprintf("user%d@example.com", i),
			"age":   20 + i%50,
		}
		tdb.DB.Insert("users", data)
	}

	b.Run("SimpleWhere", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			qb := tdb.DB.Select("users", nil).Where("age", ">", 30)
			_, err := qb.Execute()
			if err != nil {
				b.Fatalf("Failed to execute query: %v", err)
			}
		}
	})

	b.Run("ChainedOperations", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			qb := tdb.DB.Select("users", []string{"name", "email"}).
				Where("age", ">=", 25).
				Where("age", "<=", 45).
				OrderBy("name", "ASC").
				Limit(50)
			_, err := qb.Execute()
			if err != nil {
				b.Fatalf("Failed to execute chained query: %v", err)
			}
		}
	})

	b.Run("Count", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			qb := tdb.DB.Select("users", nil).Where("age", ">", 40)
			_, err := qb.Count()
			if err != nil {
				b.Fatalf("Failed to count: %v", err)
			}
		}
	})

	b.Run("First", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			qb := tdb.DB.Select("users", nil).Where("age", "=", 25)
			_, err := qb.First()
			if err != nil {
				b.Fatalf("Failed to get first: %v", err)
			}
		}
	})
}

func BenchmarkConcurrentOperations(b *testing.B) {
	tdb := SetupTestDB(&testing.T{})
	defer tdb.Cleanup()

	tdb.CreateUserSchema()

	// Add initial data
	for i := 0; i < 100; i++ {
		script := fmt.Sprintf(`models.User.add({name: "User%d", email: "user%d@example.com", age: %d})`, i, i, 20+i%50)
		tdb.ExecuteJS(script)
	}

	b.Run("ConcurrentReads", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := tdb.ExecuteJS(`models.User.select().limit(10).execute()`)
				if err != nil {
					b.Fatalf("Failed concurrent read: %v", err)
				}
			}
		})
	})

	b.Run("ConcurrentWrites", func(b *testing.B) {
		counter := 0
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				counter++
				script := fmt.Sprintf(`models.User.add({name: "ConcUser%d", email: "conc%d@example.com", age: 25})`, counter, counter)
				_, err := tdb.ExecuteJS(script)
				if err != nil {
					b.Fatalf("Failed concurrent write: %v", err)
				}
			}
		})
	})
}
