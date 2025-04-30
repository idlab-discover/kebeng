package database

import (
	"fmt"
	"time"

	"github.com/idlab-discover/kebeng/services/account/internal/config"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // needed for file source
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

func NewDatabase(cfg *config.Config) (*sqlx.DB, error) {
	return createDatabaseWithDSN(getDSN(cfg), cfg)
}

func createDatabaseWithDSN(connectionString string, cfg *config.Config) (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error

	maxRetries := 10
	retryInterval := 3 * time.Second

	for try := 0; try < maxRetries; try++ {
		db, err = sqlx.Connect("postgres", connectionString)
		if err == nil {
			logrus.Info("Connected to database")

			err = RunMigrations(db, cfg)
			if err != nil {
				logrus.Errorf("Migration failed: %v", err)
			}
			return db, nil
		}
		logrus.Errorf("Failed to connect to database at try %d: %v", try, err)
		time.Sleep(retryInterval)
	}
	logrus.Errorf("Failed to connect to database after %d retries", maxRetries)
	return nil, err
}

func getDSN(cfg *config.Config) string {
	return fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=disable TimeZone=UTC",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBName,
		cfg.DBPassword,
	)
}

// runs the migration files in the /migrations folder
func RunMigrations(db *sqlx.DB, cfg *config.Config) error {
	logrus.Info("Running database migrations")

	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %v", err)
	}

	// the path here is the path in the container where the migration files are stored

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
	logrus.Info("Database migrations ran successfully")
	return nil
}
