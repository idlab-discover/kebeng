package database

import (
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // needed for file source
	"github.com/idlab-discover/kebeng/services/store/internal/config"
	"github.com/sirupsen/logrus"
	gormPostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewDatabase(cfg *config.Config) (*gorm.DB, error) {
	return createDatabaseWithDSN(getDSN(cfg), cfg)
}

func createDatabaseWithDSN(connectionString string, cfg *config.Config) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	maxRetries := 10
	retryInterval := 3 * time.Second

	for try := 0; try < maxRetries; try++ {
		db, err = gorm.Open(gormPostgres.Open(connectionString), &gorm.Config{})
		if err == nil {
			logrus.Info("Connected to database")

			err = runMigrations(db, cfg)
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

// TODO: make this function usefull
// still need to do 2 checks most of the time where this function is called
func CheckDBForErrorOrNoRows(db *gorm.DB) (*gorm.DB, bool) {
	if db.Error != nil {
		logrus.Error(db.Error)
		return db, false
	} else if db.RowsAffected == 0 {
		logrus.Warn("no rows found")
		return db, false
	}

	return db, true
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
func runMigrations(db *gorm.DB, cfg *config.Config) error {
	logrus.Info("Running database migrations")

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw database connection: %v", err)
	}

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
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
