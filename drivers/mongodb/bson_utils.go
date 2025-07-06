package mongodb

import (
	"reflect"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// decodeBSONWithDBTags decodes a BSON document into a struct using db tags instead of bson tags
func decodeBSONWithDBTags(doc bson.M, dest any) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr || destVal.Elem().Kind() != reflect.Struct {
		// For non-struct types, use regular BSON unmarshaling
		bsonBytes, err := bson.Marshal(doc)
		if err != nil {
			return err
		}
		return bson.Unmarshal(bsonBytes, dest)
	}

	destVal = destVal.Elem()
	destType := destVal.Type()

	// Build a map of db tag to field index
	dbTagToField := make(map[string]int)
	for i := 0; i < destType.NumField(); i++ {
		field := destType.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag != "" && dbTag != "-" {
			dbTagToField[dbTag] = i
		}
	}

	// Map values from document to struct fields
	for key, value := range doc {
		fieldIdx, found := dbTagToField[key]
		if !found {
			// Try to match by field name (case-insensitive)
			for i := 0; i < destType.NumField(); i++ {
				field := destType.Field(i)
				if strings.EqualFold(field.Name, key) {
					fieldIdx = i
					found = true
					break
				}
			}
		}

		if found && destVal.Field(fieldIdx).CanSet() {
			if err := setFieldValue(destVal.Field(fieldIdx), value); err != nil {
				// Continue on error to set other fields
				continue
			}
		}
	}

	return nil
}

// setFieldValue sets a struct field value from a BSON value
func setFieldValue(field reflect.Value, value any) error {
	if value == nil {
		// Set zero value for nil
		field.Set(reflect.Zero(field.Type()))
		return nil
	}

	valueVal := reflect.ValueOf(value)
	fieldType := field.Type()

	// Direct assignment if types match
	if valueVal.Type().AssignableTo(fieldType) {
		field.Set(valueVal)
		return nil
	}

	// Handle special conversions
	switch field.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64:
		switch v := value.(type) {
		case int32:
			field.SetInt(int64(v))
		case int64:
			field.SetInt(v)
		case float64:
			field.SetInt(int64(v))
		default:
			if valueVal.CanConvert(fieldType) {
				field.Set(valueVal.Convert(fieldType))
			}
		}
	case reflect.Float32, reflect.Float64:
		switch v := value.(type) {
		case float32:
			field.SetFloat(float64(v))
		case float64:
			field.SetFloat(v)
		case int32:
			field.SetFloat(float64(v))
		case int64:
			field.SetFloat(float64(v))
		default:
			if valueVal.CanConvert(fieldType) {
				field.Set(valueVal.Convert(fieldType))
			}
		}
	case reflect.String:
		field.SetString(valueVal.String())
	case reflect.Bool:
		if b, ok := value.(bool); ok {
			field.SetBool(b)
		}
	case reflect.Struct:
		// Handle time.Time specially
		if fieldType == reflect.TypeOf(time.Time{}) {
			switch v := value.(type) {
			case time.Time:
				field.Set(reflect.ValueOf(v))
			case string:
				if t, err := time.Parse(time.RFC3339, v); err == nil {
					field.Set(reflect.ValueOf(t))
				}
			}
		}
	case reflect.Ptr:
		// Handle pointer fields
		if value != nil {
			newPtr := reflect.New(fieldType.Elem())
			if err := setFieldValue(newPtr.Elem(), value); err == nil {
				field.Set(newPtr)
			}
		}
	default:
		// Try direct conversion as last resort
		if valueVal.CanConvert(fieldType) {
			field.Set(valueVal.Convert(fieldType))
		}
	}

	return nil
}
