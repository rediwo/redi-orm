package mongodb

import (
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBCommand represents a MongoDB operation command
type MongoDBCommand struct {
	Operation    string   `json:"operation"`              // "insert", "find", "update", "delete", "aggregate"
	Collection   string   `json:"collection"`             // Collection name
	Documents    []any    `json:"documents,omitempty"`    // For insert operations
	Filter       bson.M   `json:"filter,omitempty"`       // For find/update/delete
	Update       bson.M   `json:"update,omitempty"`       // For update operations
	Pipeline     []bson.M `json:"pipeline,omitempty"`     // For aggregate operations
	Options      bson.M   `json:"options,omitempty"`      // Operation options (limit, skip, sort, etc.)
	Fields       []string `json:"fields,omitempty"`       // Field names for projection
	LastInsertID int64    `json:"lastInsertId,omitempty"` // For passing generated ID from insert query
}

// ToJSON converts the command to JSON string
func (c *MongoDBCommand) ToJSON() (string, error) {
	// Create a copy for JSON marshaling with proper BSON type handling
	jsonCmd := map[string]any{
		"operation":  c.Operation,
		"collection": c.Collection,
	}

	// Add optional fields only if they have values
	if len(c.Documents) > 0 {
		jsonCmd["documents"] = c.Documents
	}

	if c.Filter != nil {
		jsonCmd["filter"] = c.Filter
	}

	if c.Update != nil {
		jsonCmd["update"] = c.Update
	}

	if c.LastInsertID > 0 {
		jsonCmd["lastInsertId"] = c.LastInsertID
	}

	// Handle pipeline with BSON type conversion
	if len(c.Pipeline) > 0 {
		pipeline := make([]any, len(c.Pipeline))
		for i, stage := range c.Pipeline {
			pipeline[i] = convertBSONTypes(stage)
		}
		jsonCmd["pipeline"] = pipeline
	}

	// Handle options with BSON type conversion
	if c.Options != nil {
		jsonCmd["options"] = convertBSONTypes(c.Options)
	}

	if len(c.Fields) > 0 {
		jsonCmd["fields"] = c.Fields
	}

	data, err := json.Marshal(jsonCmd)
	if err != nil {
		return "", fmt.Errorf("failed to marshal MongoDB command: %w", err)
	}
	return string(data), nil
}

// convertBSONTypes converts BSON types to JSON-friendly types
func convertBSONTypes(v any) any {
	switch val := v.(type) {
	case bson.D:
		// Convert ordered document to map
		m := bson.M{}
		for _, elem := range val {
			m[elem.Key] = convertBSONTypes(elem.Value)
		}
		return m
	case bson.M:
		// Recursively convert map values
		m := make(map[string]any)
		for k, v := range val {
			m[k] = convertBSONTypes(v)
		}
		return m
	case []bson.M:
		// Convert slice of maps
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = convertBSONTypes(item)
		}
		return result
	case []any:
		// Recursively convert slice elements
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = convertBSONTypes(item)
		}
		return result
	case map[string]any:
		// Recursively convert map values
		m := make(map[string]any)
		for k, v := range val {
			m[k] = convertBSONTypes(v)
		}
		return m
	default:
		// Return as-is for other types
		return v
	}
}

// FromJSON parses a JSON string into a MongoDBCommand
func (c *MongoDBCommand) FromJSON(jsonStr string) error {
	if err := json.Unmarshal([]byte(jsonStr), c); err != nil {
		return fmt.Errorf("failed to unmarshal MongoDB command: %w", err)
	}
	return nil
}
