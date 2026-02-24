package bsonschema_test

import (
	"testing"
	"time"

	"github.com/lucaspopp0/bsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type CustomString string

type Inner struct {
	Name  string `bson:"name"`
	Count int32  `bson:"count"`
}

type ArrayItem struct {
	Label string `bson:"label"`
	Value int    `bson:"value"`
}

type testStruct struct {
	ID              bson.ObjectID `bson:"_id,omitempty"`
	Name            string        `bson:"name"`
	Version         int64         `bson:"version"`
	SmallInt        int32         `bson:"smallInt"`
	RegularInt      int           `bson:"regularInt"`
	Score           float64       `bson:"score"`
	Active          bool          `bson:"active"`
	CreatedAt       time.Time     `bson:"createdAt"`
	OptionalTime    *time.Time    `bson:"optionalTime"`
	OptionalName    *string       `bson:"optionalName"`
	Status          CustomString  `bson:"status"`
	Nested          Inner         `bson:"nested"`
	NestedPtr       *Inner        `bson:"nestedPtr"`
	Tags            []string      `bson:"tags"`
	Items           []ArrayItem   `bson:"items"`
	Skipped         string        `bson:"-"`
	unexportedField string        //nolint:unused
}

func TestSchemaFor_WrapsInJsonSchema(t *testing.T) {
	result := bsonschema.SchemaFor[testStruct]()

	schema, ok := result["$jsonSchema"].(bson.M)
	require.True(t, ok, "$jsonSchema key must be present")
	assert.Equal(t, "object", schema["bsonType"])
	assert.Equal(t, false, schema["additionalProperties"])
}

func TestSchemaFor_PrimitiveTypes(t *testing.T) {
	props := extractProperties[testStruct](t)

	tests := map[string]string{
		"_id":        "objectId",
		"name":       "string",
		"version":    "long",
		"smallInt":   "int",
		"regularInt": "int",
		"score":      "double",
		"active":     "bool",
		"createdAt":  "date",
	}

	for field, expectedType := range tests {
		t.Run(field, func(t *testing.T) {
			prop, ok := props[field].(bson.M)
			require.True(t, ok, "field %s must exist in properties", field)
			assert.Equal(t, expectedType, prop["bsonType"], "field %s", field)
		})
	}
}

func TestSchemaFor_PointerFieldsUnwrapped(t *testing.T) {
	props := extractProperties[testStruct](t)

	optTime, ok := props["optionalTime"].(bson.M)
	require.True(t, ok)
	assert.Equal(t, "date", optTime["bsonType"])

	optName, ok := props["optionalName"].(bson.M)
	require.True(t, ok)
	assert.Equal(t, "string", optName["bsonType"])
}

func TestSchemaFor_StringAlias(t *testing.T) {
	props := extractProperties[testStruct](t)

	status, ok := props["status"].(bson.M)
	require.True(t, ok)
	assert.Equal(t, "string", status["bsonType"])
}

func TestSchemaFor_NestedStruct(t *testing.T) {
	props := extractProperties[testStruct](t)

	nested, ok := props["nested"].(bson.M)
	require.True(t, ok)
	assert.Equal(t, "object", nested["bsonType"])
	assert.Equal(t, false, nested["additionalProperties"])

	innerProps, ok := nested["properties"].(bson.M)
	require.True(t, ok)
	assert.Contains(t, innerProps, "name")
	assert.Contains(t, innerProps, "count")
}

func TestSchemaFor_NestedPointerStruct(t *testing.T) {
	props := extractProperties[testStruct](t)

	nested, ok := props["nestedPtr"].(bson.M)
	require.True(t, ok)
	assert.Equal(t, "object", nested["bsonType"])
	assert.Equal(t, false, nested["additionalProperties"])
}

func TestSchemaFor_SliceOfPrimitives(t *testing.T) {
	props := extractProperties[testStruct](t)

	tags, ok := props["tags"].(bson.M)
	require.True(t, ok)
	assert.Equal(t, "array", tags["bsonType"])

	items, ok := tags["items"].(bson.M)
	require.True(t, ok)
	assert.Equal(t, "string", items["bsonType"])
}

func TestSchemaFor_SliceOfStructs(t *testing.T) {
	props := extractProperties[testStruct](t)

	items, ok := props["items"].(bson.M)
	require.True(t, ok)
	assert.Equal(t, "array", items["bsonType"])

	itemSchema, ok := items["items"].(bson.M)
	require.True(t, ok)
	assert.Equal(t, "object", itemSchema["bsonType"])
	assert.Equal(t, false, itemSchema["additionalProperties"])
}

func TestSchemaFor_SkippedAndUnexportedFields(t *testing.T) {
	props := extractProperties[testStruct](t)

	assert.NotContains(t, props, "Skipped")
	assert.NotContains(t, props, "-")
	assert.NotContains(t, props, "unexportedField")
}

func TestSchemaFor_PropertyCount(t *testing.T) {
	props := extractProperties[testStruct](t)
	// 15 fields: _id, name, version, smallInt, regularInt, score, active,
	// createdAt, optionalTime, optionalName, status, nested, nestedPtr, tags, items
	assert.Len(t, props, 15)
}

func TestSchemaFor_TopLevelSlice(t *testing.T) {
	result := bsonschema.SchemaFor[[]string]()

	schema, ok := result["$jsonSchema"].(bson.M)
	require.True(t, ok, "$jsonSchema key must be present")
	assert.Equal(t, "array", schema["bsonType"])

	items, ok := schema["items"].(bson.M)
	require.True(t, ok, "items key must be present for array")
	assert.Equal(t, "string", items["bsonType"])
}

func TestSchemaFor_TopLevelSliceOfStructs(t *testing.T) {
	result := bsonschema.SchemaFor[[]Inner]()

	schema, ok := result["$jsonSchema"].(bson.M)
	require.True(t, ok, "$jsonSchema key must be present")
	assert.Equal(t, "array", schema["bsonType"])

	items, ok := schema["items"].(bson.M)
	require.True(t, ok, "items key must be present for array")
	assert.Equal(t, "object", items["bsonType"])
	assert.Equal(t, false, items["additionalProperties"])
}

func TestSchemaFor_TopLevelMap(t *testing.T) {
	result := bsonschema.SchemaFor[map[string]int]()

	schema, ok := result["$jsonSchema"].(bson.M)
	require.True(t, ok, "$jsonSchema key must be present")
	assert.Equal(t, "object", schema["bsonType"])

	additional, ok := schema["additionalProperties"].(bson.M)
	require.True(t, ok, "additionalProperties must be a schema for map value type")
	assert.Equal(t, "int", additional["bsonType"])
}

func TestSchemaFor_TopLevelMapOfStructs(t *testing.T) {
	result := bsonschema.SchemaFor[map[string]Inner]()

	schema, ok := result["$jsonSchema"].(bson.M)
	require.True(t, ok, "$jsonSchema key must be present")
	assert.Equal(t, "object", schema["bsonType"])

	additional, ok := schema["additionalProperties"].(bson.M)
	require.True(t, ok, "additionalProperties must be a schema for map value type")
	assert.Equal(t, "object", additional["bsonType"])
	assert.Equal(t, false, additional["additionalProperties"])
}

type extendedIntStruct struct {
	TinyInt  int8    `bson:"tinyInt"`
	SmallInt int16   `bson:"smallInt"`
	UTiny    uint8   `bson:"uTiny"`
	USmall   uint16  `bson:"uSmall"`
	UMedium  uint32  `bson:"uMedium"`
	ULarge   uint64  `bson:"uLarge"`
	UDefault uint    `bson:"uDefault"`
	SmallFlt float32 `bson:"smallFloat"`
}

func TestSchemaFor_ExtendedIntTypes(t *testing.T) {
	props := extractProperties[extendedIntStruct](t)

	tests := map[string]string{
		"tinyInt":    "int",
		"smallInt":   "int",
		"uTiny":      "long",
		"uSmall":     "long",
		"uMedium":    "long",
		"uLarge":     "long",
		"uDefault":   "long",
		"smallFloat": "double",
	}

	for field, expectedType := range tests {
		t.Run(field, func(t *testing.T) {
			prop, ok := props[field].(bson.M)
			require.True(t, ok, "field %s must exist in properties", field)
			assert.Equal(t, expectedType, prop["bsonType"], "field %s", field)
		})
	}
}

func TestSchemaFor_FixedLengthArray(t *testing.T) {
	type withArray struct {
		Coords [3]float64 `bson:"coords"`
		Flags  [8]bool    `bson:"flags"`
	}

	props := extractProperties[withArray](t)

	coords, ok := props["coords"].(bson.M)
	require.True(t, ok)
	assert.Equal(t, "array", coords["bsonType"])

	items, ok := coords["items"].(bson.M)
	require.True(t, ok)
	assert.Equal(t, "double", items["bsonType"])

	flags, ok := props["flags"].(bson.M)
	require.True(t, ok)
	assert.Equal(t, "array", flags["bsonType"])

	flagItems, ok := flags["items"].(bson.M)
	require.True(t, ok)
	assert.Equal(t, "bool", flagItems["bsonType"])
}

func extractProperties[T any](t *testing.T) bson.M {
	t.Helper()
	result := bsonschema.SchemaFor[T]()
	schema, ok := result["$jsonSchema"].(bson.M)
	require.True(t, ok)
	props, ok := schema["properties"].(bson.M)
	require.True(t, ok)
	return props
}
