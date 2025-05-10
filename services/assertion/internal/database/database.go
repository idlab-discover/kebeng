package database

import (
	"context"
	"fmt"

	_ "github.com/golang-migrate/migrate/v4/source/file" // required for file source
	"github.com/idlab-discover/kebeng/services/assertion/internal/config"
	"github.com/sirupsen/logrus"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewMongoDBConnection establishes a connection to MongoDB
func NewMongoDBConnection(cfg *config.Config) (*mongo.Client, error) {
	clientOptions := options.Client().ApplyURI(cfg.MongoDBURI)

	// Establish connection to MongoDB
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Verify if the connection is successful
	err = client.Ping(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	logrus.Info("Connected to MongoDB database")
	return client, nil
}
