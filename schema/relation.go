package schema

import (
	"fmt"
	"strings"
)

// RelationMetadata contains additional metadata for relations
type RelationMetadata struct {
	// For many-to-many relations
	ThroughTable   string   // Junction table name
	ThroughFields  []string // Fields in junction table
	
	// Inverse relation
	InverseField   string   // Field name in the related model
	InverseRelation *Relation
	
	// Query options
	OnDelete       string   // CASCADE, RESTRICT, SET NULL
	OnUpdate       string   // CASCADE, RESTRICT, SET NULL
}

// GetJunctionTableName generates a junction table name for many-to-many relations
func GetJunctionTableName(modelA, modelB string) string {
	// For same model self-relations
	if modelA == modelB {
		tableA := ModelNameToTableName(modelA)
		singularA := strings.TrimSuffix(tableA, "s")
		return singularA + "_" + tableA
	}
	
	// Sort models alphabetically to ensure consistent naming
	firstModel, secondModel := modelA, modelB
	if strings.ToLower(modelA) > strings.ToLower(modelB) {
		firstModel, secondModel = modelB, modelA
	}
	
	// Convert to table names
	tableA := ModelNameToTableName(firstModel)
	tableB := ModelNameToTableName(secondModel)
	
	// Extract singular form of first table
	singularA := tableA
	if strings.HasSuffix(tableA, "ies") {
		// categories -> category
		singularA = tableA[:len(tableA)-3] + "y"
	} else if strings.HasSuffix(tableA, "s") {
		// users -> user
		singularA = tableA[:len(tableA)-1]
	}
	
	return singularA + "_" + tableB
}

// ValidateRelation validates a relation definition
func ValidateRelation(relation *Relation, currentModel, relatedModel *Schema) error {
	// Check if related model exists
	if relatedModel == nil {
		return fmt.Errorf("related model %s not found", relation.Model)
	}
	
	// Validate foreign key field exists
	switch relation.Type {
	case RelationManyToOne:
		// Foreign key should be in current model
		if _, err := currentModel.GetField(relation.ForeignKey); err != nil {
			return fmt.Errorf("foreign key field %s not found in model %s", relation.ForeignKey, currentModel.Name)
		}
		
	case RelationOneToMany:
		// Foreign key should be in related model
		if _, err := relatedModel.GetField(relation.ForeignKey); err != nil {
			// Try with default foreign key name
			defaultFK := strings.ToLower(currentModel.Name) + "Id"
			if _, err := relatedModel.GetField(defaultFK); err != nil {
				return fmt.Errorf("foreign key field %s not found in model %s", relation.ForeignKey, relatedModel.Name)
			}
		}
		
	case RelationOneToOne:
		// Foreign key can be in either model
		_, errCurrent := currentModel.GetField(relation.ForeignKey)
		_, errRelated := relatedModel.GetField(relation.ForeignKey)
		if errCurrent != nil && errRelated != nil {
			return fmt.Errorf("foreign key field %s not found in either model", relation.ForeignKey)
		}
		
	case RelationManyToMany:
		// No direct foreign key, will use junction table
		if relation.ForeignKey == "" {
			relation.ForeignKey = strings.ToLower(currentModel.Name) + "Id"
		}
		if relation.References == "" {
			relation.References = "id"
		}
	}
	
	// Validate references field exists in related model
	if relation.References != "" {
		if _, err := relatedModel.GetField(relation.References); err != nil {
			return fmt.Errorf("references field %s not found in model %s", relation.References, relatedModel.Name)
		}
	}
	
	return nil
}

// IsArrayFieldType checks if a field type represents an array
func IsArrayFieldType(fieldType FieldType) bool {
	switch fieldType {
	case FieldTypeStringArray, FieldTypeIntArray, FieldTypeInt64Array,
		 FieldTypeFloatArray, FieldTypeBoolArray, FieldTypeDecimalArray,
		 FieldTypeDateTimeArray:
		return true
	default:
		return false
	}
}

// BuildJoinCondition builds the SQL join condition for a relation
func BuildJoinCondition(relation *Relation, currentTable, relatedTable string, currentSchema, relatedSchema *Schema) (string, error) {
	switch relation.Type {
	case RelationManyToOne:
		// JOIN related_table ON current_table.foreign_key = related_table.references
		currentCol, err := currentSchema.GetColumnNameByFieldName(relation.ForeignKey)
		if err != nil {
			return "", err
		}
		relatedCol, err := relatedSchema.GetColumnNameByFieldName(relation.References)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s.%s = %s.%s", currentTable, currentCol, relatedTable, relatedCol), nil
		
	case RelationOneToMany:
		// JOIN related_table ON related_table.foreign_key = current_table.references
		relatedCol, err := relatedSchema.GetColumnNameByFieldName(relation.ForeignKey)
		if err != nil {
			return "", err
		}
		currentCol, err := currentSchema.GetColumnNameByFieldName(relation.References)
		if err != nil {
			// If references not found, try "id"
			currentCol, err = currentSchema.GetColumnNameByFieldName("id")
			if err != nil {
				return "", err
			}
		}
		return fmt.Sprintf("%s.%s = %s.%s", relatedTable, relatedCol, currentTable, currentCol), nil
		
	case RelationOneToOne:
		// Determine which table has the foreign key
		if _, err := currentSchema.GetField(relation.ForeignKey); err == nil {
			// Foreign key in current table
			currentCol, err := currentSchema.GetColumnNameByFieldName(relation.ForeignKey)
			if err != nil {
				return "", err
			}
			relatedCol, err := relatedSchema.GetColumnNameByFieldName(relation.References)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s.%s = %s.%s", currentTable, currentCol, relatedTable, relatedCol), nil
		} else {
			// Foreign key in related table
			relatedCol, err := relatedSchema.GetColumnNameByFieldName(relation.ForeignKey)
			if err != nil {
				return "", err
			}
			currentCol, err := currentSchema.GetColumnNameByFieldName(relation.References)
			if err != nil {
				currentCol = "id" // Default to id
			}
			return fmt.Sprintf("%s.%s = %s.%s", relatedTable, relatedCol, currentTable, currentCol), nil
		}
		
	case RelationManyToMany:
		// This requires two joins through junction table
		return "", fmt.Errorf("many-to-many relations require special handling")
		
	default:
		return "", fmt.Errorf("unknown relation type: %s", relation.Type)
	}
}