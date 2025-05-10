package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoWrapper is an interface that wraps the MongoDB collection methods
// that we use in our application. This allows us to mock the MongoDB
// collection in our tests.

type SingleResult interface {
	Decode(v interface{}) error
}

type MongoCollection struct {
	col *mongo.Collection
}

func (mc *MongoCollection) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) SingleResult {
	return mc.col.FindOne(ctx, filter, opts...)
}

func (mc *MongoCollection) InsertOne(ctx context.Context, doc interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	return mc.col.InsertOne(ctx, doc, opts...)
}
