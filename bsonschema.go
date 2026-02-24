// Package bsonschema generates MongoDB $jsonSchema validator documents from
// Go structs using reflection. The generated schemas use bsonType (not JSON
// Schema's "type") and are intended to be applied to a collection via the
// collMod command with validationLevel "strict" and validationAction "error".
//
// Usage:
//
//	schema := bsonschema.SchemaFor[MyModel]()
//	db.RunCommand(ctx, bson.D{
//	    {Key: "collMod", Value: "myCollection"},
//	    {Key: "validator", Value: schema},
//	})
package bsonschema

import (
	"reflect"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Cached reflect.Type values for special-cased types that don't map cleanly
// from reflect.Kind alone (time.Time is a struct but should be "date", and
// bson.ObjectID is a [12]byte array but should be "objectId").
var (
	timeType     = reflect.TypeOf((*time.Time)(nil)).Elem()
	objectIDType = reflect.TypeOf((*bson.ObjectID)(nil)).Elem()
)

// SchemaOf produces a MongoDB $jsonSchema validator document from a Go struct.
// It reads bson struct tags for field names and maps Go types to BSON types.
// All objects are emitted with additionalProperties: false.
func SchemaOf(model any) bson.M {
	return bson.M{
		"$jsonSchema": schemaForType(reflect.TypeOf(model)),
	}
}

// SchemaFor produces a MongoDB $jsonSchema validator document from a Go struct.
// It reads bson struct tags for field names and maps Go types to BSON types.
// All objects are emitted with additionalProperties: false.
func SchemaFor[T any]() bson.M {
	var zero T

	return SchemaOf(zero)
}

// schemaForType returns the bsonType descriptor for a type.
// Pointers are unwrapped, and type aliases (e.g. type Status string) are
// resolved via reflect.Kind so they map to their underlying BSON type.
func schemaForType(t reflect.Type) bson.M {
	if t.Kind() == reflect.Pointer {
		return schemaForType(t.Elem())
	}

	// Check concrete types first -- these are structs/arrays whose Kind()
	// would otherwise cause them to be handled by the generic struct/slice
	// cases below, producing the wrong bsonType.
	switch t {
	case objectIDType:
		return bson.M{"bsonType": "objectId"}
	case timeType:
		return bson.M{"bsonType": "date"}
	}

	// Map Go kinds to BSON types. Type aliases (e.g. "type Status string")
	// resolve to their underlying kind, so they're handled correctly here.
	switch t.Kind() {
	case reflect.String:
		return bson.M{"bsonType": "string"}
	case reflect.Bool:
		return bson.M{"bsonType": "bool"}
	case reflect.Int64:
		return bson.M{"bsonType": "long"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return bson.M{"bsonType": "int"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return bson.M{"bsonType": "long"}
	case reflect.Uint64:
		return bson.M{"bsonType": "long"}
	case reflect.Float32, reflect.Float64:
		return bson.M{"bsonType": "double"}
	case reflect.Struct:
		return schemaForObject(t)
	case reflect.Array, reflect.Slice:
		return bson.M{
			"bsonType": "array",
			"items":    schemaForType(t.Elem()),
		}
	case reflect.Map:
		return bson.M{
			"bsonType":             "object",
			"additionalProperties": schemaForType(t.Elem()),
		}
	default:
		// Fallback for unrecognized kinds; "string" is a safe default
		// since MongoDB stores most scalar values as strings.
		return bson.M{"bsonType": "string"}
	}
}

// schemaForObject builds a bsonType "object" document for a struct type,
// iterating over its exported fields and recursing into nested structs.
func schemaForObject(t reflect.Type) bson.M {
	// If t is a pointer, unwrap recursively
	if t.Kind() == reflect.Pointer {
		return schemaForType(t.Elem())
	}

	// Iterate over all fields
	properties := bson.M{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		name, skip := fieldNameFromTags(field)
		if skip {
			continue
		}

		properties[name] = schemaForType(field.Type)
	}

	return bson.M{
		"bsonType":             "object",
		"properties":           properties,
		"additionalProperties": false,
	}
}

// fieldName extracts the MongoDB field name from a struct field's bson tag
func fieldNameFromTags(f reflect.StructField) (string, bool) {
	// Fetch the "bson" struct tag
	tag := f.Tag.Get("bson")

	// If there is any comma in the struct tag, only look
	// at the value before the comma
	tag, _, _ = strings.Cut(tag, ",")

	// If empty, fall back to the Go field name
	if tag == "" {
		return f.Name, false
	}

	// If "-", omit and skip
	if tag == "-" {
		return "", true
	}

	// Otherwise, return the value from the struct tag
	return tag, false
}
