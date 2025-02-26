package database

import (
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // needed for file source
	"github.com/idlab-discover/kebeng/config/configkey"
	"github.com/idlab-discover/kebeng/pkg/models"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	gormPostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func CreateDatabase() (*gorm.DB, error) {
	return CreateDatabaseWithDSN(getDSN())
}

func CreateDatabaseWithDSN(connectionString string) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	maxRetries := 10
	retryInterval := 3 * time.Second

	for try := 0; try < maxRetries; try++ {
		db, err = gorm.Open(gormPostgres.Open(connectionString), &gorm.Config{})
		if err != nil {
			logrus.Errorf("Failed to connect to database at try %d: %v", try, err)
			time.Sleep(retryInterval)
            continue
		}
        logrus.Info("Connected to database")
        DB = db

        err = RunMigrations(db)
        if err != nil {
            logrus.Errorf("Migration failed: %v", err)
        }
        return db, nil
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

func getDSN() string {
	database := viper.GetString(configkey.DatabaseDatabase)
	password := viper.GetString(configkey.DatabasePassword)
	sslMode := viper.GetString(configkey.DatabaseSSLMode)
	timezone := viper.GetString(configkey.DatabaseTimezone)
	host := viper.GetString(configkey.DatabaseHost)
	username := viper.GetString(configkey.DatabaseUsername)
	port := viper.GetInt(configkey.DatabasePort)

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		host, username, password, database, port, sslMode, timezone)

	return dsn
}

// migrates using GORM framework instead of migration files, not used atm
func MigrateWithLog(name string, i interface{}, db *gorm.DB) {
	err := db.AutoMigrate(i)
	if err != nil {
		logrus.Error(err)
		panic("Failed to auto migrate: " + name)
	}
}

func MigrateDatabase(db *gorm.DB) {
	logrus.Info("Migrating database")
	MigrateWithLog("models.Account", &models.Account{}, db)
	MigrateWithLog("models.Key", &models.Key{}, db)
	MigrateWithLog("models.SnapEntry", &models.SnapEntry{}, db)
	MigrateWithLog("models.SnapRevision", &models.SnapRevision{}, db)

	MigrateWithLog("models.SnapTrack", &models.SnapTrack{}, db)
	MigrateWithLog("models.SnapChannel", &models.SnapChannel{}, db)
	MigrateWithLog("models.SnapBranch", &models.SnapBranch{}, db)
}

// runs the migration files in the /migrations folder
func RunMigrations(db *gorm.DB) error {
	logrus.Info("Running database migrations")

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw database connection: %v", err)
	}

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file:///app/migrations",
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
	logrus.Println("Migrations ran successfully")
	return nil
}
