package mongodb

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func TestMarshalMongoDocumentJSONPreservesFieldOrder(t *testing.T) {
	doc := bson.D{
		{Key: "z", Value: int32(1)},
		{Key: "nested", Value: bson.D{
			{Key: "b", Value: int32(2)},
			{Key: "a", Value: int32(1)},
		}},
		{Key: "arr", Value: bson.A{
			bson.D{
				{Key: "y", Value: int32(1)},
				{Key: "x", Value: int32(2)},
			},
		}},
		{Key: "_id", Value: "1"},
		{Key: "a", Value: int32(3)},
	}

	gotBytes, err := marshalMongoDocumentJSON(doc)
	if err != nil {
		t.Fatalf("marshalMongoDocumentJSON returned error: %v", err)
	}

	want := `{"z":1,"nested":{"b":2,"a":1},"arr":[{"y":1,"x":2}],"_id":"1","a":3}`
	if string(gotBytes) != want {
		t.Fatalf("expected %s, got %s", want, string(gotBytes))
	}
}
