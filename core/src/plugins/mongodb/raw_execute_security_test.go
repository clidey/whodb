package mongodb

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestRejectDangerousPipelineStages(t *testing.T) {
	dangerous := []bson.A{
		{bson.D{{Key: "$match", Value: bson.D{{Key: "$where", Value: "true"}}}}},
		{bson.D{{Key: "$project", Value: bson.D{{Key: "x", Value: bson.D{{Key: "$function", Value: bson.D{}}}}}}}},
		{bson.D{{Key: "$out", Value: "stolen"}}},
		{bson.D{{Key: "$merge", Value: "other"}}},
		{bson.D{{Key: "$group", Value: bson.D{{Key: "v", Value: bson.D{{Key: "$accumulator", Value: bson.D{}}}}}}}},
	}
	for i, p := range dangerous {
		if err := rejectDangerousPipelineStages(p); err == nil {
			t.Errorf("case %d: expected dangerous pipeline to be rejected", i)
		}
	}
}

func TestRejectDangerousPipelineStages_AllowsSafe(t *testing.T) {
	safe := bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "status", Value: "active"}}}},
		bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$city"}, {Key: "n", Value: bson.D{{Key: "$sum", Value: 1}}}}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "n", Value: -1}}}},
		bson.D{{Key: "$limit", Value: 10}},
	}
	if err := rejectDangerousPipelineStages(safe); err != nil {
		t.Errorf("expected safe pipeline to be allowed, got %v", err)
	}
}
