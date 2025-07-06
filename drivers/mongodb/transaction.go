package mongodb

import (
	"context"
	"fmt"

	"github.com/rediwo/redi-orm/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBTransaction implements the Transaction interface for MongoDB
type MongoDBTransaction struct {
	session mongo.Session
	db      *MongoDB
	ctx     context.Context
}

// NewMongoDBTransaction creates a new MongoDB transaction
func NewMongoDBTransaction(session mongo.Session, db *MongoDB) *MongoDBTransaction {
	return &MongoDBTransaction{
		session: session,
		db:      db,
	}
}

// Model creates a new model query within the transaction
func (t *MongoDBTransaction) Model(modelName string) types.ModelQuery {
	// Create a MongoDB model query
	mongoModelQuery := NewMongoDBModelQuery(t.db, modelName)
	// Wrap it to use the session
	return &transactionModelQuery{
		ModelQuery: mongoModelQuery,
		session:    t.session,
		db:         t.db,
	}
}

// Raw creates a new raw query within the transaction
func (t *MongoDBTransaction) Raw(sql string, args ...any) types.RawQuery {
	// MongoDB doesn't use SQL, so this would be a raw MongoDB command
	return NewMongoDBRawQuery(t.db.client.Database(t.db.dbName), t.session, t.db, sql, args...)
}

// Commit commits the transaction
func (t *MongoDBTransaction) Commit(ctx context.Context) error {
	return t.session.CommitTransaction(ctx)
}

// Rollback rolls back the transaction
func (t *MongoDBTransaction) Rollback(ctx context.Context) error {
	return t.session.AbortTransaction(ctx)
}

// Savepoint creates a savepoint (not supported in MongoDB)
func (t *MongoDBTransaction) Savepoint(ctx context.Context, name string) error {
	return fmt.Errorf("savepoints are not supported in MongoDB")
}

// RollbackTo rolls back to a savepoint (not supported in MongoDB)
func (t *MongoDBTransaction) RollbackTo(ctx context.Context, name string) error {
	return fmt.Errorf("savepoints are not supported in MongoDB")
}

// CreateMany performs batch insert within the transaction
func (t *MongoDBTransaction) CreateMany(ctx context.Context, modelName string, data []any) (types.Result, error) {
	collection := t.db.client.Database(t.db.dbName).Collection(t.db.getCollectionName(modelName))

	// Convert data to documents
	documents := make([]any, len(data))
	copy(documents, data)

	opts := options.InsertMany()
	result, err := collection.InsertMany(mongo.NewSessionContext(ctx, t.session), documents, opts)
	if err != nil {
		return types.Result{}, err
	}

	return types.Result{
		RowsAffected: int64(len(result.InsertedIDs)),
	}, nil
}

// UpdateMany performs batch update within the transaction
func (t *MongoDBTransaction) UpdateMany(ctx context.Context, modelName string, condition types.Condition, data any) (types.Result, error) {
	collection := t.db.client.Database(t.db.dbName).Collection(t.db.getCollectionName(modelName))

	// Convert condition to MongoDB filter
	filter := t.db.conditionToFilter(condition)

	// Convert data to update document
	update := bson.M{"$set": data}

	opts := options.Update()
	result, err := collection.UpdateMany(mongo.NewSessionContext(ctx, t.session), filter, update, opts)
	if err != nil {
		return types.Result{}, err
	}

	return types.Result{
		RowsAffected: result.ModifiedCount,
	}, nil
}

// DeleteMany performs batch delete within the transaction
func (t *MongoDBTransaction) DeleteMany(ctx context.Context, modelName string, condition types.Condition) (types.Result, error) {
	collection := t.db.client.Database(t.db.dbName).Collection(t.db.getCollectionName(modelName))

	// Convert condition to MongoDB filter
	filter := t.db.conditionToFilter(condition)

	opts := options.Delete()
	result, err := collection.DeleteMany(mongo.NewSessionContext(ctx, t.session), filter, opts)
	if err != nil {
		return types.Result{}, err
	}

	return types.Result{
		RowsAffected: result.DeletedCount,
	}, nil
}

// transactionModelQuery wraps a ModelQuery to use the transaction session
type transactionModelQuery struct {
	types.ModelQuery
	session mongo.Session
	db      *MongoDB
}

// Select creates a select query that uses the transaction session
func (t *transactionModelQuery) Select(fields ...string) types.SelectQuery {
	baseSelect := t.ModelQuery.Select(fields...)
	return &transactionSelectQuery{
		SelectQuery: baseSelect,
		session:     t.session,
		db:          t.db,
	}
}

// Insert creates an insert query that uses the transaction session
func (t *transactionModelQuery) Insert(data any) types.InsertQuery {
	baseInsert := t.ModelQuery.Insert(data)
	return &transactionInsertQuery{
		InsertQuery: baseInsert,
		session:     t.session,
		db:          t.db,
	}
}

// Update creates an update query that uses the transaction session
func (t *transactionModelQuery) Update(data any) types.UpdateQuery {
	baseUpdate := t.ModelQuery.Update(data)
	return &transactionUpdateQuery{
		UpdateQuery: baseUpdate,
		session:     t.session,
		db:          t.db,
	}
}

// Delete creates a delete query that uses the transaction session
func (t *transactionModelQuery) Delete() types.DeleteQuery {
	baseDelete := t.ModelQuery.Delete()
	return &transactionDeleteQuery{
		DeleteQuery: baseDelete,
		session:     t.session,
		db:          t.db,
	}
}

// transactionSelectQuery wraps a SelectQuery to use the transaction session
type transactionSelectQuery struct {
	types.SelectQuery
	session mongo.Session
	db      *MongoDB
}

// FindMany executes the query within the transaction
func (t *transactionSelectQuery) FindMany(ctx context.Context, dest any) error {
	// Get the MongoDB-specific select query
	mongoSelect, ok := t.SelectQuery.(*MongoDBSelectQuery)
	if !ok {
		return fmt.Errorf("expected MongoDBSelectQuery, got %T", t.SelectQuery)
	}

	// Build the command
	sql, _, err := mongoSelect.BuildSQL()
	if err != nil {
		return fmt.Errorf("failed to build command: %w", err)
	}

	// Create a raw query with session
	rawQuery := NewMongoDBRawQuery(t.db.client.Database(t.db.dbName), t.session, t.db, sql)
	return rawQuery.Find(ctx, dest)
}

// FindOne executes the query within the transaction and returns a single result
func (t *transactionSelectQuery) FindOne(ctx context.Context, dest any) error {
	// Get the MongoDB-specific select query
	mongoSelect, ok := t.SelectQuery.(*MongoDBSelectQuery)
	if !ok {
		return fmt.Errorf("expected MongoDBSelectQuery, got %T", t.SelectQuery)
	}

	// Build the command
	sql, _, err := mongoSelect.BuildSQL()
	if err != nil {
		return fmt.Errorf("failed to build command: %w", err)
	}

	// Create a raw query with session
	rawQuery := NewMongoDBRawQuery(t.db.client.Database(t.db.dbName), t.session, t.db, sql)
	return rawQuery.FindOne(ctx, dest)
}

// transactionInsertQuery wraps an InsertQuery to use the transaction session
type transactionInsertQuery struct {
	types.InsertQuery
	session mongo.Session
	db      *MongoDB
}

// Exec executes the insert within the transaction
func (t *transactionInsertQuery) Exec(ctx context.Context) (types.Result, error) {
	// Get the MongoDB-specific insert query
	mongoInsert, ok := t.InsertQuery.(*MongoDBInsertQuery)
	if !ok {
		return types.Result{}, fmt.Errorf("expected MongoDBInsertQuery, got %T", t.InsertQuery)
	}

	// Build the command
	sql, args, err := mongoInsert.BuildSQL()
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to build command: %w", err)
	}

	// Create a raw query with session
	rawQuery := NewMongoDBRawQuery(t.db.client.Database(t.db.dbName), t.session, t.db, sql, args...)
	return rawQuery.Exec(ctx)
}

// transactionUpdateQuery wraps an UpdateQuery to use the transaction session
type transactionUpdateQuery struct {
	types.UpdateQuery
	session mongo.Session
	db      *MongoDB
}

// WhereCondition overrides the base WhereCondition to maintain transaction wrapper
func (t *transactionUpdateQuery) WhereCondition(condition types.Condition) types.UpdateQuery {
	baseQuery := t.UpdateQuery.WhereCondition(condition)
	return &transactionUpdateQuery{
		UpdateQuery: baseQuery,
		session:     t.session,
		db:          t.db,
	}
}

// Exec executes the update within the transaction
func (t *transactionUpdateQuery) Exec(ctx context.Context) (types.Result, error) {
	// Get the MongoDB-specific update query
	mongoUpdate, ok := t.UpdateQuery.(*MongoDBUpdateQuery)
	if !ok {
		return types.Result{}, fmt.Errorf("expected MongoDBUpdateQuery, got %T", t.UpdateQuery)
	}

	// Build the command
	sql, args, err := mongoUpdate.BuildSQL()
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to build command: %w", err)
	}

	// Create a raw query with session
	rawQuery := NewMongoDBRawQuery(t.db.client.Database(t.db.dbName), t.session, t.db, sql, args...)
	return rawQuery.Exec(ctx)
}

// transactionDeleteQuery wraps a DeleteQuery to use the transaction session
type transactionDeleteQuery struct {
	types.DeleteQuery
	session mongo.Session
	db      *MongoDB
}

// WhereCondition overrides the base WhereCondition to maintain transaction wrapper
func (t *transactionDeleteQuery) WhereCondition(condition types.Condition) types.DeleteQuery {
	baseQuery := t.DeleteQuery.WhereCondition(condition)
	return &transactionDeleteQuery{
		DeleteQuery: baseQuery,
		session:     t.session,
		db:          t.db,
	}
}

// Exec executes the delete within the transaction
func (t *transactionDeleteQuery) Exec(ctx context.Context) (types.Result, error) {
	// Get the MongoDB-specific delete query
	mongoDelete, ok := t.DeleteQuery.(*MongoDBDeleteQuery)
	if !ok {
		return types.Result{}, fmt.Errorf("expected MongoDBDeleteQuery, got %T", t.DeleteQuery)
	}

	// Build the command
	sql, args, err := mongoDelete.BuildSQL()
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to build command: %w", err)
	}

	// Create a raw query with session
	rawQuery := NewMongoDBRawQuery(t.db.client.Database(t.db.dbName), t.session, t.db, sql, args...)
	return rawQuery.Exec(ctx)
}
