package database

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // required for file source
	"github.com/idlab-discover/kebeng/services/assertion/internal/config"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewPostgresDatabase establishes a connection to the PostgreSQL database
func NewPostgresDatabase(cfg *config.Config) (*sqlx.DB, error) {
	return createPostgresDatabaseWithDSN(getPostgresDSN(cfg), cfg)
}

// createPostgresDatabaseWithDSN establishes a connection to PostgreSQL using a DSN (Data Source Name)
func createPostgresDatabaseWithDSN(connectionString string, cfg *config.Config) (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error

	maxRetries := 10
	retryInterval := 3 * time.Second

	for try := 0; try < maxRetries; try++ {
		db, err = sqlx.Connect("postgres", connectionString)
		if err == nil {
			logrus.Info("Connected to PostgreSQL database")

			err = RunPostgresMigrations(db, cfg)
			if err != nil {
				logrus.Errorf("Migration failed: %v", err)
			}

			return db, nil
		}
		logrus.Errorf("Failed to connect to PostgreSQL database at try %d: %v", try, err)
		time.Sleep(retryInterval)
	}
	logrus.Errorf("Failed to connect to PostgreSQL database after %d retries", maxRetries)
	return nil, err
}

// getPostgresDSN returns the Postgres Data Source Name (DSN) string
func getPostgresDSN(cfg *config.Config) string {
	return fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=disable TimeZone=UTC",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBName,
		cfg.DBPassword,
	)
}

// RunPostgresMigrations executes the migrations for PostgreSQL
func RunPostgresMigrations(db *sqlx.DB, cfg *config.Config) error {
	logrus.Info("Running PostgreSQL database migrations")

	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %v", err)
	}

	// Path to your migration files
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", cfg.MigrationPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migrate: %v", err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %v", err)
	}
	logrus.Info("PostgreSQL database migrations ran successfully")
	return nil
}

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
