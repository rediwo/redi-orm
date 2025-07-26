package mongodb

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/rediwo/redi-orm/base"
	"github.com/rediwo/redi-orm/sql"
	"github.com/rediwo/redi-orm/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBRawQuery implements RawQuery for MongoDB
type MongoDBRawQuery struct {
	database *mongo.Database
	session  mongo.Session // optional, for transactions
	mongoDb  *MongoDB      // optional, for field mapping
	command  string
	args     []any
}

// NewMongoDBRawQuery creates a new raw query
func NewMongoDBRawQuery(database *mongo.Database, session mongo.Session, mongoDb *MongoDB, command string, args ...any) *MongoDBRawQuery {
	return &MongoDBRawQuery{
		database: database,
		session:  session,
		mongoDb:  mongoDb,
		command:  command,
		args:     args,
	}
}

// Exec executes a MongoDB command
func (q *MongoDBRawQuery) Exec(ctx context.Context) (types.Result, error) {
	// Check if input is SQL statement
	if sql.DetectSQL(q.command) {
		return q.executeSQLCommand(ctx)
	}

	// Parse as MongoDB JSON command
	var cmd MongoDBCommand
	if err := cmd.FromJSON(q.command); err != nil {
		// Log the command for debugging
		fmt.Printf("[MongoDB] Failed to parse command: %s\n", q.command)
		return types.Result{}, fmt.Errorf("failed to parse MongoDB command: %w", err)
	}

	collection := q.database.Collection(cmd.Collection)
	var result types.Result

	// Execute based on operation type
	switch cmd.Operation {
	case "insert":
		return q.executeInsert(ctx, collection, &cmd)
	case "update":
		return q.executeUpdate(ctx, collection, &cmd)
	case "delete":
		return q.executeDelete(ctx, collection, &cmd)
	case "find":
		// Find operations should use Find/FindOne methods instead
		return result, fmt.Errorf("find operations should use Find/FindOne methods")
	case "aggregate":
		// Aggregate operations should use Find method
		return result, fmt.Errorf("aggregate operations should use Find method")
	default:
		return result, fmt.Errorf("unsupported operation: %s", cmd.Operation)
	}
}

// Find executes a query and returns multiple results
func (q *MongoDBRawQuery) Find(ctx context.Context, dest any) error {
	// Check if input is SQL statement
	if sql.DetectSQL(q.command) {
		return q.executeSQLFind(ctx, dest)
	}

	// Parse as MongoDB JSON command
	var cmd MongoDBCommand
	if err := cmd.FromJSON(q.command); err != nil {
		return fmt.Errorf("failed to parse MongoDB command: %w", err)
	}

	collection := q.database.Collection(cmd.Collection)

	switch cmd.Operation {
	case "find":
		return q.executeFind(ctx, collection, &cmd, dest)
	case "aggregate":
		return q.executeAggregate(ctx, collection, &cmd, dest)
	default:
		return fmt.Errorf("operation %s does not support Find", cmd.Operation)
	}
}

// FindOne executes a query and returns a single result
func (q *MongoDBRawQuery) FindOne(ctx context.Context, dest any) error {
	// Check if input is SQL statement
	if sql.DetectSQL(q.command) {
		return q.executeSQLFindOne(ctx, dest)
	}

	// Parse as MongoDB JSON command
	var cmd MongoDBCommand
	if err := cmd.FromJSON(q.command); err != nil {
		// Log the command for debugging
		if q.mongoDb != nil && q.mongoDb.GetLogger() != nil {
			q.mongoDb.GetLogger().Error("MongoDB FindOne - Failed to parse command: %s", q.command)
		}
		return fmt.Errorf("failed to parse MongoDB command: %w", err)
	}

	collection := q.database.Collection(cmd.Collection)

	switch cmd.Operation {
	case "find":
		return q.executeFindOne(ctx, collection, &cmd, dest)
	case "aggregate":
		// For aggregate, we'll use Find and take the first result
		// This is handled in the caller typically
		return q.executeAggregate(ctx, collection, &cmd, dest)
	default:
		return fmt.Errorf("operation %s does not support FindOne", cmd.Operation)
	}
}

// executeInsert handles insert operations
func (q *MongoDBRawQuery) executeInsert(ctx context.Context, collection *mongo.Collection, cmd *MongoDBCommand) (types.Result, error) {
	start := time.Now()
	if len(cmd.Documents) == 0 {
		return types.Result{}, fmt.Errorf("no documents to insert")
	}

	// Convert documents to BSON if needed
	documents := make([]any, len(cmd.Documents))
	for i, doc := range cmd.Documents {
		// Ensure _id field exists
		if docMap, ok := doc.(map[string]any); ok {
			if _, hasID := docMap["_id"]; !hasID {
				docMap["_id"] = primitive.NewObjectID()
			}
			documents[i] = docMap
		} else {
			documents[i] = doc
		}
	}

	var result *mongo.InsertManyResult
	var err error

	// Log the command
	if q.mongoDb != nil && q.mongoDb.GetLogger() != nil {
		dbLogger := base.NewDBLogger(q.mongoDb.GetLogger())
		cmdJSON, _ := cmd.ToJSON()
		defer func() {
			dbLogger.LogCommand(cmdJSON, time.Since(start))
		}()
	}

	if q.session != nil {
		// Use session context directly for transactions
		sessionCtx := mongo.NewSessionContext(ctx, q.session)
		result, err = collection.InsertMany(sessionCtx, documents)
	} else {
		result, err = collection.InsertMany(ctx, documents)
	}

	if err != nil {
		return types.Result{}, fmt.Errorf("failed to insert documents: %w", err)
	}

	// Use LastInsertID from command if available (for sequence-generated IDs)
	lastInsertID := int64(0)
	if cmd.LastInsertID > 0 {
		lastInsertID = cmd.LastInsertID
	}

	return types.Result{
		RowsAffected: int64(len(result.InsertedIDs)),
		LastInsertID: lastInsertID,
	}, nil
}

// executeUpdate handles update operations
func (q *MongoDBRawQuery) executeUpdate(ctx context.Context, collection *mongo.Collection, cmd *MongoDBCommand) (types.Result, error) {
	start := time.Now()
	if cmd.Update == nil {
		return types.Result{}, fmt.Errorf("update requires update document")
	}

	// Use empty filter if none provided (updates all documents)
	filter := cmd.Filter
	if filter == nil {
		filter = bson.M{}
	}

	// Ensure update document has proper MongoDB update operators
	updateDoc := cmd.Update
	if _, hasOperator := updateDoc["$set"]; !hasOperator {
		// Wrap in $set if no operators present
		updateDoc = bson.M{"$set": updateDoc}
	}

	// Log the command
	if q.mongoDb != nil && q.mongoDb.GetLogger() != nil {
		dbLogger := base.NewDBLogger(q.mongoDb.GetLogger())
		cmdJSON, _ := cmd.ToJSON()
		defer func() {
			dbLogger.LogCommand(cmdJSON, time.Since(start))
		}()
	}

	var result *mongo.UpdateResult
	var err error

	if q.session != nil {
		// Use session context directly for transactions
		sessionCtx := mongo.NewSessionContext(ctx, q.session)
		result, err = collection.UpdateMany(sessionCtx, filter, updateDoc)
	} else {
		result, err = collection.UpdateMany(ctx, filter, updateDoc)
	}

	if err != nil {
		return types.Result{}, fmt.Errorf("failed to update documents: %w", err)
	}

	return types.Result{
		RowsAffected: result.MatchedCount, // Use MatchedCount to align with SQL behavior
	}, nil
}

// executeDelete handles delete operations
func (q *MongoDBRawQuery) executeDelete(ctx context.Context, collection *mongo.Collection, cmd *MongoDBCommand) (types.Result, error) {
	start := time.Now()
	if cmd.Filter == nil {
		return types.Result{}, fmt.Errorf("delete requires a filter")
	}

	// Log the command
	if q.mongoDb != nil && q.mongoDb.GetLogger() != nil {
		dbLogger := base.NewDBLogger(q.mongoDb.GetLogger())
		cmdJSON, _ := cmd.ToJSON()
		defer func() {
			dbLogger.LogCommand(cmdJSON, time.Since(start))
		}()
	}

	var result *mongo.DeleteResult
	var err error

	if q.session != nil {
		// Use session context directly for transactions
		sessionCtx := mongo.NewSessionContext(ctx, q.session)
		result, err = collection.DeleteMany(sessionCtx, cmd.Filter)
	} else {
		result, err = collection.DeleteMany(ctx, cmd.Filter)
	}

	if err != nil {
		return types.Result{}, fmt.Errorf("failed to delete documents: %w", err)
	}

	return types.Result{
		RowsAffected: result.DeletedCount,
	}, nil
}

// executeFind handles find operations
func (q *MongoDBRawQuery) executeFind(ctx context.Context, collection *mongo.Collection, cmd *MongoDBCommand, dest any) error {
	// Check if collection exists by trying to list collections
	database := collection.Database()
	collectionNames, listErr := database.ListCollectionNames(ctx, bson.M{"name": collection.Name()})
	if listErr == nil && len(collectionNames) == 0 {
		// Collection doesn't exist - return error that matches test expectations
		return fmt.Errorf("collection %s does not exist", cmd.Collection)
	}

	opts := options.Find()

	// Apply options from command
	if cmd.Options != nil {
		if limit, ok := cmd.Options["limit"].(int64); ok {
			opts.SetLimit(limit)
		} else if limitData := cmd.Options["limit"]; limitData != nil {
			// Handle different numeric types from JSON unmarshaling
			if limitFloat, ok := limitData.(float64); ok {
				limit := int64(limitFloat)
				opts.SetLimit(limit)
			} else {
				fmt.Printf("[MongoDB Find] Limit data type issue: %T = %v\n", limitData, limitData)
			}
		}
		if skip, ok := cmd.Options["skip"].(int64); ok {
			opts.SetSkip(skip)
		} else if skipData := cmd.Options["skip"]; skipData != nil {
			// Handle different numeric types from JSON unmarshaling
			if skipFloat, ok := skipData.(float64); ok {
				skip := int64(skipFloat)
				opts.SetSkip(skip)
			} else {
				fmt.Printf("[MongoDB Find] Skip data type issue: %T = %v\n", skipData, skipData)
			}
		}
		if sort, ok := cmd.Options["sort"].(bson.D); ok {
			opts.SetSort(sort)
		} else if sortData := cmd.Options["sort"]; sortData != nil {
			// Handle sort data that comes from JSON unmarshaling
			if sortSlice, ok := sortData.([]any); ok {
				// Convert []interface{} to bson.D
				var sortDoc bson.D
				for _, item := range sortSlice {
					if itemMap, ok := item.(map[string]any); ok {
						for key, value := range itemMap {
							if key == "Key" && value != nil {
								keyStr := value.(string)
								// Find the corresponding Value
								for k2, v2 := range itemMap {
									if k2 == "Value" {
										sortDoc = append(sortDoc, bson.E{Key: keyStr, Value: v2})
										break
									}
								}
							}
						}
					}
				}
				if len(sortDoc) > 0 {
					opts.SetSort(sortDoc)
				}
			} else if sortMap, ok := sortData.(map[string]any); ok {
				// Handle simple map format
				var sortDoc bson.D
				for field, direction := range sortMap {
					if dirFloat, ok := direction.(float64); ok {
						sortDoc = append(sortDoc, bson.E{Key: field, Value: int(dirFloat)})
					} else {
						sortDoc = append(sortDoc, bson.E{Key: field, Value: direction})
					}
				}
				if len(sortDoc) > 0 {
					opts.SetSort(sortDoc)
				}
			} else {
				fmt.Printf("[MongoDB Find] Sort data type issue: %T = %v\n", sortData, sortData)
			}
		}
	}

	// Apply projection if fields specified
	if len(cmd.Fields) > 0 {
		projection := bson.M{}
		hasID := false
		for _, field := range cmd.Fields {
			projection[field] = 1
			if field == "_id" || field == "id" {
				hasID = true
			}
		}
		// Exclude _id if not explicitly requested
		if !hasID {
			projection["_id"] = 0
		}
		opts.SetProjection(projection)
	}

	var cursor *mongo.Cursor
	var err error

	if q.session != nil {
		// Use session context directly for transactions
		sessionCtx := mongo.NewSessionContext(ctx, q.session)
		cursor, err = collection.Find(sessionCtx, cmd.Filter, opts)
	} else {
		cursor, err = collection.Find(ctx, cmd.Filter, opts)
	}

	if err != nil {
		return fmt.Errorf("failed to execute find: %w", err)
	}
	defer cursor.Close(ctx)

	// First decode to generic documents
	var docs []bson.M
	err = cursor.All(ctx, &docs)
	if err != nil {
		return fmt.Errorf("failed to decode cursor results: %w", err)
	}

	// For raw queries, keep original column names instead of mapping to schema field names
	// This allows users to access fields using database column names as expected in raw SQL

	// Now convert documents to the destination type
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer")
	}

	destVal = destVal.Elem()
	if destVal.Kind() != reflect.Slice {
		return fmt.Errorf("destination must be a pointer to slice")
	}

	// Create a new slice of the appropriate type
	sliceType := destVal.Type()
	elemType := sliceType.Elem()
	newSlice := reflect.MakeSlice(sliceType, 0, len(docs))

	// Convert each document to the destination type
	for _, doc := range docs {
		elem := reflect.New(elemType)

		// Use custom decoder for structs with db tags
		if err := decodeBSONWithDBTags(doc, elem.Interface()); err != nil {
			// Fallback to regular BSON unmarshaling
			bsonBytes, err := bson.Marshal(doc)
			if err != nil {
				return fmt.Errorf("failed to marshal document: %w", err)
			}

			if err := bson.Unmarshal(bsonBytes, elem.Interface()); err != nil {
				return fmt.Errorf("failed to unmarshal document: %w", err)
			}
		}

		newSlice = reflect.Append(newSlice, elem.Elem())
	}

	destVal.Set(newSlice)
	return nil
}

// executeFindOne handles find one operations
func (q *MongoDBRawQuery) executeFindOne(ctx context.Context, collection *mongo.Collection, cmd *MongoDBCommand, dest any) error {
	opts := options.FindOne()

	// Apply options from command
	if cmd.Options != nil {
		if skip, ok := cmd.Options["skip"].(int64); ok {
			opts.SetSkip(skip)
		}
		if sort, ok := cmd.Options["sort"].(bson.D); ok {
			opts.SetSort(sort)
		}
	}

	// Apply projection if fields specified
	if len(cmd.Fields) > 0 {
		projection := bson.M{}
		hasID := false
		for _, field := range cmd.Fields {
			projection[field] = 1
			if field == "_id" || field == "id" {
				hasID = true
			}
		}
		// Exclude _id if not explicitly requested
		if !hasID {
			projection["_id"] = 0
		}
		opts.SetProjection(projection)
	}

	var result *mongo.SingleResult

	if q.session != nil {
		// Use session context directly for transactions
		sessionCtx := mongo.NewSessionContext(ctx, q.session)
		result = collection.FindOne(sessionCtx, cmd.Filter, opts)
	} else {
		result = collection.FindOne(ctx, cmd.Filter, opts)
	}

	// Decode result
	var doc bson.M
	if err := result.Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			// Convert to standard "no rows" error for compatibility
			return fmt.Errorf("no documents found")
		}
		return fmt.Errorf("failed to decode result: %w", err)
	}

	// For raw queries, keep original column names instead of mapping to schema field names
	// This allows users to access fields using database column names as expected in raw SQL

	// Use custom decoder for structs with db tags
	return decodeBSONWithDBTags(doc, dest)
}

// executeAggregate handles aggregation pipeline operations
func (q *MongoDBRawQuery) executeAggregate(ctx context.Context, collection *mongo.Collection, cmd *MongoDBCommand, dest any) error {
	if len(cmd.Pipeline) == 0 {
		return fmt.Errorf("aggregate requires a pipeline")
	}

	// Check if collection exists by trying to list collections
	database := collection.Database()
	collectionNames, listErr := database.ListCollectionNames(ctx, bson.M{"name": collection.Name()})
	if listErr == nil && len(collectionNames) == 0 {
		// Collection doesn't exist - return error that matches test expectations
		return fmt.Errorf("collection %s does not exist", cmd.Collection)
	}

	opts := options.Aggregate()

	var cursor *mongo.Cursor
	var err error

	if q.session != nil {
		// Use session context directly for transactions
		sessionCtx := mongo.NewSessionContext(ctx, q.session)
		cursor, err = collection.Aggregate(sessionCtx, cmd.Pipeline, opts)
	} else {
		cursor, err = collection.Aggregate(ctx, cmd.Pipeline, opts)
	}

	if err != nil {
		return fmt.Errorf("failed to execute aggregate: %w", err)
	}
	defer cursor.Close(ctx)

	// Handle single result for FindOne
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() == reflect.Ptr && destValue.Elem().Kind() != reflect.Slice {
		// Single result expected
		var results []bson.M
		if err := cursor.All(ctx, &results); err != nil {
			return err
		}
		if len(results) == 0 {
			return fmt.Errorf("no documents found")
		}

		// Convert BSON types to standard Go types for compatibility
		result := convertBSONToGoTypes(results[0])

		// For raw queries, keep original column names instead of mapping to schema field names
		// This allows users to access fields using database column names as expected in raw SQL

		// For map[string]any destination, assign directly
		if destMap, ok := dest.(*map[string]any); ok {
			*destMap = result.(map[string]any)
			return nil
		}

		// For single value types (int64, float64, etc.), extract the first value from the result document
		if q.isSingleValueType(destValue.Elem()) {
			return q.extractSingleValue(result, dest)
		}

		// For other types, use BSON marshaling
		bsonBytes, err := bson.Marshal(result)
		if err != nil {
			return err
		}
		return bson.Unmarshal(bsonBytes, dest)
	}

	// Multiple results
	err = cursor.All(ctx, dest)
	if err != nil {
		return fmt.Errorf("failed to decode aggregate results: %w", err)
	}

	return nil
}

// executeSQLCommand parses and executes a SQL command
func (q *MongoDBRawQuery) executeSQLCommand(ctx context.Context) (types.Result, error) {
	// Parse SQL statement
	parser := sql.NewParser(q.command)
	stmt, err := parser.Parse()
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to parse SQL: %w", err)
	}

	// Translate SQL to MongoDB command
	if q.mongoDb == nil {
		return types.Result{}, fmt.Errorf("MongoDB instance is required for SQL translation")
	}

	translator := NewMongoDBSQLTranslator(q.mongoDb)
	// Set arguments for parameter substitution
	if len(q.args) > 0 {
		translator.SetArgs(q.args)
	}
	mongoCmd, err := translator.TranslateToCommand(stmt)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to translate SQL to MongoDB command: %w", err)
	}

	// Execute the translated MongoDB command
	return q.executeMongoDBCommand(ctx, mongoCmd)
}

// executeSQLFind executes SQL query for Find operations
func (q *MongoDBRawQuery) executeSQLFind(ctx context.Context, dest any) error {
	// Parse SQL statement
	parser := sql.NewParser(q.command)
	stmt, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse SQL: %w", err)
	}

	// Only SELECT statements are supported for Find
	selectStmt, ok := stmt.(*sql.SelectStatement)
	if !ok {
		return fmt.Errorf("only SELECT statements are supported for Find operations")
	}

	// Translate SQL to MongoDB command
	if q.mongoDb == nil {
		return fmt.Errorf("MongoDB instance is required for SQL translation")
	}

	translator := NewMongoDBSQLTranslator(q.mongoDb)
	// Set arguments for parameter substitution
	if len(q.args) > 0 {
		translator.SetArgs(q.args)
	}
	mongoCmd, err := translator.TranslateToCommand(selectStmt)
	if err != nil {
		return fmt.Errorf("failed to translate SQL to MongoDB command: %w", err)
	}

	// Process subqueries if present
	mongoCmd, err = q.processSubqueries(ctx, mongoCmd, translator)
	if err != nil {
		return fmt.Errorf("failed to process subqueries: %w", err)
	}

	// Execute the translated MongoDB command
	collection := q.database.Collection(mongoCmd.Collection)
	switch mongoCmd.Operation {
	case "find":
		return q.executeFind(ctx, collection, mongoCmd, dest)
	case "aggregate":
		return q.executeAggregate(ctx, collection, mongoCmd, dest)
	default:
		return fmt.Errorf("unsupported operation for Find: %s", mongoCmd.Operation)
	}
}

// executeSQLFindOne executes SQL query for FindOne operations
func (q *MongoDBRawQuery) executeSQLFindOne(ctx context.Context, dest any) error {
	// Parse SQL statement
	parser := sql.NewParser(q.command)
	stmt, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse SQL: %w", err)
	}

	// Only SELECT statements are supported for FindOne
	selectStmt, ok := stmt.(*sql.SelectStatement)
	if !ok {
		return fmt.Errorf("only SELECT statements are supported for FindOne operations")
	}

	// Translate SQL to MongoDB command
	if q.mongoDb == nil {
		return fmt.Errorf("MongoDB instance is required for SQL translation")
	}

	translator := NewMongoDBSQLTranslator(q.mongoDb)
	// Set arguments for parameter substitution
	if len(q.args) > 0 {
		translator.SetArgs(q.args)
	}
	mongoCmd, err := translator.TranslateToCommand(selectStmt)
	if err != nil {
		return fmt.Errorf("failed to translate SQL to MongoDB command: %w", err)
	}

	// Execute the translated MongoDB command
	collection := q.database.Collection(mongoCmd.Collection)
	switch mongoCmd.Operation {
	case "find":
		return q.executeFindOne(ctx, collection, mongoCmd, dest)
	case "aggregate":
		return q.executeAggregate(ctx, collection, mongoCmd, dest)
	default:
		return fmt.Errorf("unsupported operation for FindOne: %s", mongoCmd.Operation)
	}
}

// executeMongoDBCommand executes a MongoDB command
func (q *MongoDBRawQuery) executeMongoDBCommand(ctx context.Context, cmd *MongoDBCommand) (types.Result, error) {
	collection := q.database.Collection(cmd.Collection)

	// Execute based on operation type
	switch cmd.Operation {
	case "insert":
		return q.executeInsert(ctx, collection, cmd)
	case "find":
		// For find operations in Exec, we can't return the documents
		// Just return a success result
		return types.Result{RowsAffected: 1}, nil
	case "aggregate":
		// For aggregate operations in Exec, similar to find
		return types.Result{RowsAffected: 1}, nil
	case "update":
		return q.executeUpdate(ctx, collection, cmd)
	case "delete":
		return q.executeDelete(ctx, collection, cmd)
	default:
		return types.Result{}, fmt.Errorf("unsupported operation: %s", cmd.Operation)
	}
}

// isSingleValueType checks if the target type is a single primitive value
func (q *MongoDBRawQuery) isSingleValueType(destValue reflect.Value) bool {
	switch destValue.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.Bool, reflect.String:
		return true
	default:
		return false
	}
}

// extractSingleValue extracts the first value from an aggregation result document
func (q *MongoDBRawQuery) extractSingleValue(result any, dest any) error {
	resultMap, ok := result.(map[string]any)
	if !ok {
		return fmt.Errorf("expected map result for single value extraction, got %T", result)
	}

	// For aggregation queries, there should be exactly one field in the result
	// Extract the first (and typically only) value
	var value any
	for _, v := range resultMap {
		value = v
		break // Take the first value
	}

	if value == nil {
		return fmt.Errorf("no value found in aggregation result")
	}

	// Convert the value to the target type using reflection
	destValue := reflect.ValueOf(dest).Elem()
	sourceValue := reflect.ValueOf(value)

	// Handle type conversion
	switch destValue.Kind() {
	case reflect.Int64:
		switch v := value.(type) {
		case int32:
			destValue.SetInt(int64(v))
		case int64:
			destValue.SetInt(v)
		case float64:
			destValue.SetInt(int64(v))
		default:
			return fmt.Errorf("cannot convert %T to int64", value)
		}
	case reflect.Float64:
		switch v := value.(type) {
		case float64:
			destValue.SetFloat(v)
		case float32:
			destValue.SetFloat(float64(v))
		case int32:
			destValue.SetFloat(float64(v))
		case int64:
			destValue.SetFloat(float64(v))
		default:
			return fmt.Errorf("cannot convert %T to float64", value)
		}
	case reflect.Int:
		switch v := value.(type) {
		case int32:
			destValue.SetInt(int64(v))
		case int64:
			destValue.SetInt(v)
		case float64:
			destValue.SetInt(int64(v))
		default:
			return fmt.Errorf("cannot convert %T to int", value)
		}
	case reflect.String:
		destValue.SetString(fmt.Sprintf("%v", value))
	case reflect.Bool:
		if b, ok := value.(bool); ok {
			destValue.SetBool(b)
		} else {
			return fmt.Errorf("cannot convert %T to bool", value)
		}
	default:
		// For other types, try to assign directly if compatible
		if sourceValue.Type().AssignableTo(destValue.Type()) {
			destValue.Set(sourceValue)
		} else {
			return fmt.Errorf("cannot assign %T to %s", value, destValue.Type().String())
		}
	}

	return nil
}

// convertBSONToGoTypes converts BSON-specific types to standard Go types
func convertBSONToGoTypes(v any) any {
	switch val := v.(type) {
	case primitive.A: // BSON array
		// Convert to []any
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = convertBSONToGoTypes(item)
		}
		return result
	case bson.M: // BSON document
		// Recursively convert nested documents
		result := make(map[string]any)
		for k, v := range val {
			result[k] = convertBSONToGoTypes(v)
		}
		return result
	case map[string]any:
		// Recursively convert map values
		result := make(map[string]any)
		for k, v := range val {
			result[k] = convertBSONToGoTypes(v)
		}
		return result
	case []any:
		// Recursively convert slice elements
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = convertBSONToGoTypes(item)
		}
		return result
	case primitive.DateTime:
		// Convert BSON DateTime to time.Time
		return val.Time()
	case int32:
		// Convert int32 to int64 for consistency
		return int64(val)
	case float32:
		// Convert float32 to float64 for consistency
		return float64(val)
	default:
		// Return as-is for other types
		return v
	}
}

// processSubqueries processes subquery markers in the MongoDB command and executes subqueries
func (q *MongoDBRawQuery) processSubqueries(ctx context.Context, mongoCmd *MongoDBCommand, translator *MongoDBSQLTranslator) (*MongoDBCommand, error) {
	// Create subquery executor
	executor := NewSubqueryExecutor(q.mongoDb, translator)

	// Process different command types
	switch mongoCmd.Operation {
	case "aggregate":
		return q.processSubqueriesInAggregation(ctx, mongoCmd, executor)
	case "find":
		return q.processSubqueriesInFind(ctx, mongoCmd, executor)
	default:
		return mongoCmd, nil // No subquery processing needed
	}
}

// processSubqueriesInAggregation processes subqueries in aggregation pipelines
func (q *MongoDBRawQuery) processSubqueriesInAggregation(ctx context.Context, mongoCmd *MongoDBCommand, executor *SubqueryExecutor) (*MongoDBCommand, error) {
	if len(mongoCmd.Pipeline) == 0 {
		return mongoCmd, nil
	}

	// Process each stage in the pipeline
	for i, stage := range mongoCmd.Pipeline {
		newStage, err := q.processSubqueriesInStage(ctx, stage, executor)
		if err != nil {
			return nil, err
		}
		mongoCmd.Pipeline[i] = newStage
	}

	return mongoCmd, nil
}

// processSubqueriesInFind processes subqueries in find operations
func (q *MongoDBRawQuery) processSubqueriesInFind(ctx context.Context, mongoCmd *MongoDBCommand, executor *SubqueryExecutor) (*MongoDBCommand, error) {
	if mongoCmd.Filter == nil {
		return mongoCmd, nil
	}

	newFilter, err := q.processSubqueriesInBSON(ctx, mongoCmd.Filter, executor)
	if err != nil {
		return nil, err
	}

	mongoCmd.Filter = newFilter
	return mongoCmd, nil
}

// processSubqueriesInStage processes subqueries in a single aggregation stage
func (q *MongoDBRawQuery) processSubqueriesInStage(ctx context.Context, stage bson.M, executor *SubqueryExecutor) (bson.M, error) {
	newStage := bson.M{}

	for key, value := range stage {
		if key == "$match" {
			// Process $match stage which may contain subqueries
			if matchValue, ok := value.(bson.M); ok {
				newMatch, err := q.processSubqueriesInBSON(ctx, matchValue, executor)
				if err != nil {
					return nil, err
				}
				newStage[key] = newMatch
			} else {
				newStage[key] = value
			}
		} else {
			// Copy other stages as-is
			newStage[key] = value
		}
	}

	return newStage, nil
}

// processSubqueriesInBSON processes subqueries in BSON documents
func (q *MongoDBRawQuery) processSubqueriesInBSON(ctx context.Context, doc bson.M, executor *SubqueryExecutor) (bson.M, error) {
	newDoc := bson.M{}

	for key, value := range doc {
		if key == "__subquery__" {
			// This is a subquery marker - process it
			if subqueryInfo, ok := value.(bson.M); ok {
				field := subqueryInfo["field"].(string)
				operator := subqueryInfo["operator"].(string)
				subquery := subqueryInfo["subquery"].(*sql.SelectStatement)

				// Execute the subquery
				values, err := executor.ExecuteSubquery(ctx, subquery, q.args)
				if err != nil {
					return nil, fmt.Errorf("failed to execute subquery: %w", err)
				}

				// Create the MongoDB condition
				newDoc[field] = bson.M{operator: values}
			} else {
				return nil, fmt.Errorf("invalid subquery marker format")
			}
		} else if nestedDoc, ok := value.(bson.M); ok {
			// Recursively process nested documents
			newNested, err := q.processSubqueriesInBSON(ctx, nestedDoc, executor)
			if err != nil {
				return nil, err
			}
			newDoc[key] = newNested
		} else {
			// Copy other values as-is
			newDoc[key] = value
		}
	}

	return newDoc, nil
}
